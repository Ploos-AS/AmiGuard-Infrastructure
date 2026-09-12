package main

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func multipartPlatformBody(t *testing.T, platform string, includePlatform bool, payload []byte) (*bytes.Buffer, string) {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	if includePlatform {
		if err := mw.WriteField("platform", platform); err != nil {
			t.Fatal(err)
		}
	}
	if err := mw.WriteField("consent", "true"); err != nil {
		t.Fatal(err)
	}
	part, err := mw.CreateFormFile("sample", "client-name.bin")
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

func submitPlatform(t *testing.T, root, platform string, includePlatform bool) submissionReceipt {
	t.Helper()
	body, contentType := multipartPlatformBody(t, platform, includePlatform, []byte("harmless multi-platform fixture\n"))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/submissions", body)
	req.Header.Set("Content-Type", contentType)
	rr := httptest.NewRecorder()
	newHandler(testConfig(root, 4096)).ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var receipt submissionReceipt
	if err := json.Unmarshal(rr.Body.Bytes(), &receipt); err != nil {
		t.Fatal(err)
	}
	return receipt
}

func TestOmittedPlatformDefaultsToAmiga(t *testing.T) {
	root := t.TempDir()
	receipt := submitPlatform(t, root, "", false)
	if receipt.Platform != "amiga" {
		t.Fatalf("platform=%q", receipt.Platform)
	}
	if _, err := os.Stat(filepath.Join(root, "amiga", receipt.ID+".sample")); err != nil {
		t.Fatal(err)
	}
}

func TestSupportedPlatformsUseSeparateNamespaces(t *testing.T) {
	for _, platform := range []string{"amiga", "atari-st", "mac68k"} {
		t.Run(platform, func(t *testing.T) {
			root := t.TempDir()
			receipt := submitPlatform(t, root, platform, true)
			if receipt.Platform != platform {
				t.Fatalf("platform=%q", receipt.Platform)
			}
			metadataPath := filepath.Join(root, platform, receipt.ID+".json")
			data, err := os.ReadFile(metadataPath)
			if err != nil {
				t.Fatal(err)
			}
			var metadata map[string]any
			if err := json.Unmarshal(data, &metadata); err != nil {
				t.Fatal(err)
			}
			if metadata["platform"] != platform || metadata["schema_version"] != float64(2) {
				t.Fatalf("metadata=%#v", metadata)
			}
		})
	}
}

func TestUnknownPlatformFailsClosed(t *testing.T) {
	root := t.TempDir()
	body, contentType := multipartPlatformBody(t, "../../escape", true, []byte("fixture"))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/submissions", body)
	req.Header.Set("Content-Type", contentType)
	rr := httptest.NewRecorder()
	newHandler(testConfig(root, 4096)).ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("unexpected durable storage: %v", entries)
	}
}

func TestPlatformPathCannotEscapeRoot(t *testing.T) {
	root := t.TempDir()
	if _, err := platformQuarantineRoot(root, "../mac68k"); err == nil {
		t.Fatal("expected unsupported platform rejection")
	}
}
