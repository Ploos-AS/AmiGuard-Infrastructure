package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const defaultMaxUploadBytes int64 = 16 << 20

type health struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

type config struct {
	addr           string
	uploadEnabled  bool
	quarantineRoot string
	maxUploadBytes int64
}

type submissionReceipt struct {
	ID         string `json:"id"`
	SHA256     string `json:"sha256"`
	Size       int64  `json:"size"`
	ReceivedAt string `json:"received_at"`
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	srv := &http.Server{
		Addr:              cfg.addr,
		Handler:           newHandler(cfg),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       30 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}

	log.Printf("amiguard-submit listening on %s (uploads enabled=%t)", cfg.addr, cfg.uploadEnabled)
	log.Fatal(srv.ListenAndServe())
}

func loadConfig() (config, error) {
	cfg := config{
		addr:           env("AMIGUARD_SUBMIT_ADDR", "127.0.0.1:8080"),
		uploadEnabled:  strings.EqualFold(env("AMIGUARD_UPLOAD_ENABLED", "false"), "true"),
		quarantineRoot: env("AMIGUARD_QUARANTINE_ROOT", "/data/quarantine"),
		maxUploadBytes: defaultMaxUploadBytes,
	}
	if raw := os.Getenv("AMIGUARD_MAX_UPLOAD_BYTES"); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value <= 0 || value > 64<<20 {
			return config{}, errors.New("AMIGUARD_MAX_UPLOAD_BYTES must be between 1 and 67108864")
		}
		cfg.maxUploadBytes = value
	}
	if cfg.uploadEnabled {
		info, err := os.Lstat(cfg.quarantineRoot)
		if err != nil {
			return config{}, fmt.Errorf("quarantine root: %w", err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return config{}, errors.New("quarantine root must be an existing non-symlink directory")
		}
	}
	return cfg, nil
}

func newHandler(cfg config) http.Handler {
	mux := http.NewServeMux()
	abuse := newUploadAbuseGuard()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(health{Status: "ok", Service: "amiguard-submit"})
	})
	mux.HandleFunc("/api/v1/submissions", func(w http.ResponseWriter, r *http.Request) {
		if !cfg.uploadEnabled {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if !abuse.begin(w, r) {
			return
		}
		defer abuse.done()
		handleSubmission(w, r, cfg)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		fmt.Fprint(w, landingPage(cfg.uploadEnabled))
	})
	return mux
}

func handleSubmission(w http.ResponseWriter, r *http.Request, cfg config) {
	r.Body = http.MaxBytesReader(w, r.Body, cfg.maxUploadBytes+(128<<10))
	mr, err := r.MultipartReader()
	if err != nil {
		http.Error(w, "multipart/form-data required", http.StatusBadRequest)
		return
	}

	var consent bool
	var sampleSeen bool
	var tempPath string
	var size int64
	var digest string
	id, err := randomID()
	if err != nil {
		http.Error(w, "submission unavailable", http.StatusInternalServerError)
		return
	}
	defer func() {
		if tempPath != "" {
			_ = os.Remove(tempPath)
		}
	}()

	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			http.Error(w, "invalid multipart body", http.StatusBadRequest)
			return
		}
		name := part.FormName()
		switch name {
		case "consent":
			value, readErr := io.ReadAll(io.LimitReader(part, 16))
			_ = part.Close()
			if readErr != nil {
				http.Error(w, "invalid consent field", http.StatusBadRequest)
				return
			}
			consent = strings.TrimSpace(string(value)) == "true"
		case "sample":
			if sampleSeen || part.FileName() == "" {
				_ = part.Close()
				http.Error(w, "exactly one sample file is required", http.StatusBadRequest)
				return
			}
			sampleSeen = true
			var writeErr error
			tempPath, size, digest, writeErr = writeQuarantineTemp(cfg.quarantineRoot, id, part, cfg.maxUploadBytes)
			_ = part.Close()
			if writeErr != nil {
				if errors.Is(writeErr, errUploadTooLarge) {
					http.Error(w, "sample exceeds size limit", http.StatusRequestEntityTooLarge)
				} else {
					http.Error(w, "submission storage failed", http.StatusInternalServerError)
				}
				return
			}
		default:
			_ = part.Close()
			http.Error(w, "unexpected multipart field", http.StatusBadRequest)
			return
		}
	}

	if !consent {
		http.Error(w, "explicit consent is required", http.StatusBadRequest)
		return
	}
	if !sampleSeen || tempPath == "" {
		http.Error(w, "sample file is required", http.StatusBadRequest)
		return
	}

	received := time.Now().UTC().Format(time.RFC3339Nano)
	finalSample := filepath.Join(cfg.quarantineRoot, id+".sample")
	if err := os.Rename(tempPath, finalSample); err != nil {
		http.Error(w, "submission storage failed", http.StatusInternalServerError)
		return
	}
	tempPath = ""

	receipt := submissionReceipt{ID: id, SHA256: digest, Size: size, ReceivedAt: received}
	metadata := struct {
		SchemaVersion int    `json:"schema_version"`
		Kind          string `json:"kind"`
		SubmissionID  string `json:"submission_id"`
		SHA256        string `json:"sha256"`
		Size          int64  `json:"size"`
		ReceivedAt    string `json:"received_at"`
		Consent       bool   `json:"consent"`
		Executed      bool   `json:"executed"`
		Extracted     bool   `json:"extracted"`
	}{1, "amiguard-quarantine-submission", id, digest, size, received, true, false, false}
	if err := writeMetadataAtomic(cfg.quarantineRoot, id, metadata); err != nil {
		_ = os.Remove(finalSample)
		http.Error(w, "submission metadata failed", http.StatusInternalServerError)
		return
	}
	_ = syncDir(cfg.quarantineRoot)

	w.Header().Set("Cache-Control", "no-store")
	if wantsHTMLReceipt(r) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprint(w, receiptPage(receipt))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(receipt)
}

var errUploadTooLarge = errors.New("upload too large")

func writeQuarantineTemp(root, id string, src io.Reader, max int64) (string, int64, string, error) {
	path := filepath.Join(root, ".incoming-"+id)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", 0, "", err
	}
	remove := true
	defer func() {
		_ = f.Close()
		if remove {
			_ = os.Remove(path)
		}
	}()

	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(f, h), io.LimitReader(src, max+1))
	if err != nil {
		return "", 0, "", err
	}
	if n > max {
		return "", 0, "", errUploadTooLarge
	}
	if err := f.Sync(); err != nil {
		return "", 0, "", err
	}
	if err := f.Close(); err != nil {
		return "", 0, "", err
	}
	remove = false
	return path, n, hex.EncodeToString(h.Sum(nil)), nil
}

func writeMetadataAtomic(root, id string, value any) error {
	temp := filepath.Join(root, ".metadata-"+id)
	final := filepath.Join(root, id+".json")
	f, err := os.OpenFile(temp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	remove := true
	defer func() {
		_ = f.Close()
		if remove {
			_ = os.Remove(temp)
		}
	}()
	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(true)
	if err := enc.Encode(value); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(temp, final); err != nil {
		return err
	}
	remove = false
	return nil
}

func syncDir(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Sync()
}

func randomID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
