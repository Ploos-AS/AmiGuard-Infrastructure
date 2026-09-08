package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testConfig(root string, max int64) config {
	return config{
		uploadEnabled: true, quarantineRoot: root, maxUploadBytes: max,
		quarantineMinFreeBytes: 64 << 20, quarantineMaxUsedPercent: 95,
	}
}

func TestEnvFallback(t *testing.T) {
	t.Setenv("AMIGUARD_TEST_VALUE", "")
	if got := env("AMIGUARD_TEST_VALUE", "fallback"); got != "fallback" {
		t.Fatalf("got %q", got)
	}
	t.Setenv("AMIGUARD_TEST_VALUE", "set")
	if got := env("AMIGUARD_TEST_VALUE", "fallback"); got != "set" {
		t.Fatalf("got %q", got)
	}
}

func TestUploadsDisabledByDefault(t *testing.T) {
	cfg := config{addr: "127.0.0.1:8080", quarantineRoot: t.TempDir(), maxUploadBytes: 1024}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/submissions", strings.NewReader("x"))
	rr := httptest.NewRecorder()
	newHandler(cfg).ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("got status %d", rr.Code)
	}
}

func TestUnknownSubmissionPathIsNotRetrievable(t *testing.T) {
	cfg := config{quarantineRoot: t.TempDir(), maxUploadBytes: 1024}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/submissions/example", nil)
	rr := httptest.NewRecorder()
	newHandler(cfg).ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("got status %d", rr.Code)
	}
}

func TestSubmissionWritesOpaqueSampleAndMetadata(t *testing.T) {
	root := t.TempDir()
	payload := []byte("AmiGuard M3 harmless qualification fixture\n")
	body, contentType := multipartBody(t, true, "historic-virus-name.bin", payload)
	cfg := testConfig(root, 4096)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/submissions", body)
	req.Header.Set("Content-Type", contentType)
	rr := httptest.NewRecorder()
	newHandler(cfg).ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("got status %d: %s", rr.Code, rr.Body.String())
	}

	var receipt submissionReceipt
	if err := json.Unmarshal(rr.Body.Bytes(), &receipt); err != nil {
		t.Fatal(err)
	}
	if len(receipt.ID) != 32 {
		t.Fatalf("unexpected id %q", receipt.ID)
	}
	wantHash := sha256.Sum256(payload)
	if receipt.SHA256 != hex.EncodeToString(wantHash[:]) || receipt.Size != int64(len(payload)) {
		t.Fatalf("unexpected receipt: %+v", receipt)
	}

	samplePath := filepath.Join(root, receipt.ID+".sample")
	stored, err := os.ReadFile(samplePath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(stored, payload) {
		t.Fatal("stored sample bytes changed")
	}
	info, err := os.Stat(samplePath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("sample mode = %o", info.Mode().Perm())
	}

	metadataPath := filepath.Join(root, receipt.ID+".json")
	metadataBytes, err := os.ReadFile(metadataPath)
	if err != nil {
		t.Fatal(err)
	}
	var metadata map[string]any
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		t.Fatal(err)
	}
	if metadata["kind"] != "amiguard-quarantine-submission" || metadata["executed"] != false || metadata["extracted"] != false {
		t.Fatalf("unexpected metadata: %#v", metadata)
	}
	if strings.Contains(string(metadataBytes), "historic-virus-name.bin") {
		t.Fatal("client filename leaked into metadata")
	}
	if _, err := os.Stat(filepath.Join(root, "historic-virus-name.bin")); !os.IsNotExist(err) {
		t.Fatal("client filename was used as a storage path")
	}
}

func TestSubmissionRequiresConsentAndLeavesNoFiles(t *testing.T) {
	root := t.TempDir()
	body, contentType := multipartBody(t, false, "sample.bin", []byte("fixture"))
	cfg := testConfig(root, 4096)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/submissions", body)
	req.Header.Set("Content-Type", contentType)
	rr := httptest.NewRecorder()
	newHandler(cfg).ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("got status %d", rr.Code)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("quarantine not empty: %v", entries)
	}
}

func TestSubmissionRejectsOversizeAndLeavesNoFiles(t *testing.T) {
	root := t.TempDir()
	body, contentType := multipartBody(t, true, "sample.bin", bytes.Repeat([]byte("x"), 65))
	cfg := testConfig(root, 64)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/submissions", body)
	req.Header.Set("Content-Type", contentType)
	rr := httptest.NewRecorder()
	newHandler(cfg).ServeHTTP(rr, req)
	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("got status %d: %s", rr.Code, rr.Body.String())
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("quarantine not empty: %v", entries)
	}
}

func TestSubmissionRejectsUnexpectedField(t *testing.T) {
	root := t.TempDir()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	if err := mw.WriteField("consent", "true"); err != nil {
		t.Fatal(err)
	}
	if err := mw.WriteField("password", "secret"); err != nil {
		t.Fatal(err)
	}
	part, err := mw.CreateFormFile("sample", "sample.bin")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte("fixture"))
	_ = mw.Close()

	cfg := testConfig(root, 4096)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/submissions", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rr := httptest.NewRecorder()
	newHandler(cfg).ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("got status %d", rr.Code)
	}
}

func TestQuarantineCapacityGuardFailsClosed(t *testing.T) {
	root := t.TempDir()
	body, contentType := multipartBody(t, true, "sample.bin", []byte("fixture"))
	cfg := testConfig(root, 4096)
	cfg.quarantineMinFreeBytes = ^uint64(0) - 4096

	req := httptest.NewRequest(http.MethodPost, "/api/v1/submissions", body)
	req.Header.Set("Content-Type", contentType)
	rr := httptest.NewRecorder()
	newHandler(cfg).ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("got status %d: %s", rr.Code, rr.Body.String())
	}
	if rr.Header().Get("Retry-After") != "3600" {
		t.Fatalf("Retry-After = %q", rr.Header().Get("Retry-After"))
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("capacity rejection wrote files: %v", entries)
	}
}

func TestRandomIDsAreOpaque(t *testing.T) {
	a, err := randomID()
	if err != nil {
		t.Fatal(err)
	}
	b, err := randomID()
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != 32 || len(b) != 32 || a == b {
		t.Fatalf("bad IDs %q %q", a, b)
	}
}

func multipartBody(t *testing.T, consent bool, filename string, payload []byte) (*bytes.Buffer, string) {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	if err := mw.WriteField("consent", map[bool]string{true: "true", false: "false"}[consent]); err != nil {
		t.Fatal(err)
	}
	part, err := mw.CreateFormFile("sample", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	return &body, mw.FormDataContentType()
}
