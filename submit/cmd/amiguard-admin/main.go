package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	defaultQuarantineRoot = "/data/amiguard/quarantine"
	metadataKind          = "amiguard-quarantine-submission"
)

type submissionMetadata struct {
	SchemaVersion int    `json:"schema_version"`
	Kind          string `json:"kind"`
	SubmissionID  string `json:"submission_id"`
	SHA256        string `json:"sha256"`
	Size          int64  `json:"size"`
	ReceivedAt    string `json:"received_at"`
	Consent       bool   `json:"consent"`
	Executed      bool   `json:"executed"`
	Extracted     bool   `json:"extracted"`
}

type record struct {
	Metadata submissionMetadata
	Sample   string
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "amiguard-admin:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		printUsage(stderr)
		return errors.New("command required")
	}
	root := env("AMIGUARD_ADMIN_QUARANTINE_ROOT", defaultQuarantineRoot)
	cmd := args[0]
	switch cmd {
	case "dashboard":
		if len(args) != 1 {
			return errors.New("usage: amiguard-admin dashboard")
		}
		return dashboard(stdout, root)
	case "list":
		if len(args) != 1 {
			return errors.New("usage: amiguard-admin list")
		}
		return list(stdout, root)
	case "show":
		if len(args) != 2 {
			return errors.New("usage: amiguard-admin show <submission-id>")
		}
		rec, err := loadRecord(root, args[1])
		if err != nil {
			return err
		}
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(rec.Metadata)
	case "verify":
		if len(args) != 2 {
			return errors.New("usage: amiguard-admin verify <submission-id>")
		}
		return verifyCommand(stdout, root, args[1])
	case "export":
		if len(args) != 3 {
			return errors.New("usage: amiguard-admin export <submission-id> <existing-directory>")
		}
		return exportCommand(stdout, root, args[1], args[2])
	case "help", "-h", "--help":
		printUsage(stdout)
		return nil
	default:
		printUsage(stderr)
		return fmt.Errorf("unknown command %q", cmd)
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "AmiGuard quarantine administration (local CLI only)")
	fmt.Fprintln(w, "usage:")
	fmt.Fprintln(w, "  amiguard-admin dashboard")
	fmt.Fprintln(w, "  amiguard-admin list")
	fmt.Fprintln(w, "  amiguard-admin show <submission-id>")
	fmt.Fprintln(w, "  amiguard-admin verify <submission-id>")
	fmt.Fprintln(w, "  amiguard-admin export <submission-id> <existing-directory>")
}

func dashboard(w io.Writer, root string) error {
	records, err := loadRecords(root)
	if err != nil {
		return err
	}
	var total int64
	for _, rec := range records {
		total += rec.Metadata.Size
	}
	fmt.Fprintln(w, "AmiGuard Admin")
	fmt.Fprintln(w, "────────────────────────────────────────────────────────────────")
	fmt.Fprintf(w, "Quarantine: %d submissions   %s\n\n", len(records), humanBytes(total))
	return writeRecordsTable(w, records)
}

func list(w io.Writer, root string) error {
	records, err := loadRecords(root)
	if err != nil {
		return err
	}
	return writeRecordsTable(w, records)
}

func writeRecordsTable(w io.Writer, records []record) error {
	fmt.Fprintln(w, "ID                               RECEIVED                  SIZE        SHA-256")
	for _, rec := range records {
		fmt.Fprintf(w, "%-32s %-25s %-11s %.16s…\n",
			rec.Metadata.SubmissionID,
			rec.Metadata.ReceivedAt,
			humanBytes(rec.Metadata.Size),
			rec.Metadata.SHA256,
		)
	}
	return nil
}

func loadRecords(root string) ([]record, error) {
	if err := validateRoot(root); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read quarantine: %w", err)
	}
	var records []record
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".json")
		if !validID(id) {
			continue
		}
		rec, err := loadRecord(root, id)
		if err != nil {
			return nil, fmt.Errorf("load %s: %w", id, err)
		}
		records = append(records, rec)
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].Metadata.ReceivedAt > records[j].Metadata.ReceivedAt
	})
	return records, nil
}

func loadRecord(root, id string) (record, error) {
	if err := validateRoot(root); err != nil {
		return record{}, err
	}
	if !validID(id) {
		return record{}, errors.New("submission id must be 32 lowercase hexadecimal characters")
	}
	metadataPath := filepath.Join(root, id+".json")
	if err := requireRegularFile(metadataPath); err != nil {
		return record{}, fmt.Errorf("metadata: %w", err)
	}
	f, err := os.Open(metadataPath)
	if err != nil {
		return record{}, fmt.Errorf("open metadata: %w", err)
	}
	defer f.Close()
	dec := json.NewDecoder(io.LimitReader(f, 64<<10))
	dec.DisallowUnknownFields()
	var metadata submissionMetadata
	if err := dec.Decode(&metadata); err != nil {
		return record{}, fmt.Errorf("decode metadata: %w", err)
	}
	if metadata.SchemaVersion != 1 || metadata.Kind != metadataKind || metadata.SubmissionID != id {
		return record{}, errors.New("metadata contract mismatch")
	}
	if !validSHA256(metadata.SHA256) || metadata.Size < 0 {
		return record{}, errors.New("invalid metadata hash or size")
	}
	if _, err := time.Parse(time.RFC3339Nano, metadata.ReceivedAt); err != nil {
		return record{}, errors.New("invalid metadata received_at")
	}
	if !metadata.Consent || metadata.Executed || metadata.Extracted {
		return record{}, errors.New("unexpected quarantine state")
	}
	samplePath := filepath.Join(root, id+".sample")
	if err := requireRegularFile(samplePath); err != nil {
		return record{}, fmt.Errorf("sample: %w", err)
	}
	return record{Metadata: metadata, Sample: samplePath}, nil
}

func verifyCommand(w io.Writer, root, id string) error {
	rec, err := loadRecord(root, id)
	if err != nil {
		return err
	}
	digest, size, err := hashFile(rec.Sample)
	if err != nil {
		return err
	}
	if digest != rec.Metadata.SHA256 || size != rec.Metadata.Size {
		return fmt.Errorf("verification failed: metadata sha256=%s size=%d, sample sha256=%s size=%d",
			rec.Metadata.SHA256, rec.Metadata.Size, digest, size)
	}
	fmt.Fprintf(w, "VERIFIED %s sha256=%s size=%d\n", id, digest, size)
	return nil
}

func exportCommand(w io.Writer, root, id, destination string) error {
	rec, err := loadRecord(root, id)
	if err != nil {
		return err
	}
	if err := requireDirectory(destination); err != nil {
		return fmt.Errorf("export destination: %w", err)
	}
	sourceDigest, sourceSize, err := hashFile(rec.Sample)
	if err != nil {
		return err
	}
	if sourceDigest != rec.Metadata.SHA256 || sourceSize != rec.Metadata.Size {
		return errors.New("source verification failed; refusing export")
	}

	destinationPath := filepath.Join(destination, id+".sample")
	out, err := os.OpenFile(destinationPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("create export: %w", err)
	}
	remove := true
	defer func() {
		_ = out.Close()
		if remove {
			_ = os.Remove(destinationPath)
		}
	}()

	in, err := os.Open(rec.Sample)
	if err != nil {
		return fmt.Errorf("open sample: %w", err)
	}
	h := sha256.New()
	copied, err := io.Copy(io.MultiWriter(out, h), in)
	_ = in.Close()
	if err != nil {
		return fmt.Errorf("copy export: %w", err)
	}
	if err := out.Sync(); err != nil {
		return fmt.Errorf("sync export: %w", err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("close export: %w", err)
	}
	copyDigest := hex.EncodeToString(h.Sum(nil))
	if copied != rec.Metadata.Size || copyDigest != rec.Metadata.SHA256 {
		return errors.New("export stream verification failed")
	}
	destinationDigest, destinationSize, err := hashFile(destinationPath)
	if err != nil {
		return err
	}
	if destinationDigest != rec.Metadata.SHA256 || destinationSize != rec.Metadata.Size {
		return errors.New("post-export verification failed")
	}
	remove = false
	fmt.Fprintf(w, "EXPORTED %s -> %s sha256=%s size=%d\n", id, destinationPath, destinationDigest, destinationSize)
	return nil
}

func hashFile(path string) (string, int64, error) {
	if err := requireRegularFile(path); err != nil {
		return "", 0, err
	}
	f, err := os.Open(path)
	if err != nil {
		return "", 0, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, fmt.Errorf("hash %s: %w", path, err)
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

func validateRoot(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("quarantine root: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return errors.New("quarantine root must be an existing non-symlink directory")
	}
	return nil
}

func requireDirectory(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return errors.New("must be an existing non-symlink directory")
	}
	return nil
}

func requireRegularFile(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return errors.New("must be a regular non-symlink file")
	}
	return nil
}

func validID(value string) bool {
	if len(value) != 32 || strings.ToLower(value) != value {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == 16
}

func validSHA256(value string) bool {
	if len(value) != 64 || strings.ToLower(value) != value {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == 32
}

func humanBytes(value int64) string {
	const unit = int64(1024)
	if value < unit {
		return fmt.Sprintf("%d B", value)
	}
	div, exp := unit, 0
	for n := value / unit; n >= unit && exp < 3; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(value)/float64(div), "KMGT"[exp])
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
