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

func sampleFirstBody(t *testing.T, platform string, payload []byte) (*bytes.Buffer, string) {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("sample", "client.bin")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := mw.WriteField("platform", platform); err != nil {
		t.Fatal(err)
	}
	if err := mw.WriteField("consent", "true"); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	return &body, mw.FormDataContentType()
}

func TestSampleBeforePlatformLandsInSelectedNamespace(t *testing.T) {
	for _, platform := range []string{"amiga", "atari-st", "mac68k"} {
		t.Run(platform, func(t *testing.T) {
			root := t.TempDir()
			body, contentType := sampleFirstBody(t, platform, []byte("harmless order fixture\n"))
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
			if receipt.Platform != platform {
				t.Fatalf("platform=%q", receipt.Platform)
			}
			if _, err := os.Stat(filepath.Join(root, platform, receipt.ID+".sample")); err != nil {
				t.Fatal(err)
			}
			incoming, err := os.ReadDir(filepath.Join(root, ".incoming"))
			if err != nil {
				t.Fatal(err)
			}
			if len(incoming) != 0 {
				t.Fatalf("incoming spool not empty: %v", incoming)
			}
			for _, other := range []string{"amiga", "atari-st", "mac68k"} {
				if other == platform {
					continue
				}
				if _, err := os.Stat(filepath.Join(root, other)); !os.IsNotExist(err) {
					t.Fatalf("unexpected namespace %s created", other)
				}
			}
		})
	}
}

func TestUnsupportedPlatformAfterSampleLeavesNoPlatformData(t *testing.T) {
	root := t.TempDir()
	body, contentType := sampleFirstBody(t, "../../escape", []byte("harmless rejected fixture\n"))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/submissions", body)
	req.Header.Set("Content-Type", contentType)
	rr := httptest.NewRecorder()
	newHandler(testConfig(root, 4096)).ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	for _, platform := range []string{"amiga", "atari-st", "mac68k"} {
		if _, err := os.Stat(filepath.Join(root, platform)); !os.IsNotExist(err) {
			t.Fatalf("unexpected durable platform namespace %s", platform)
		}
	}
	incoming, err := os.ReadDir(filepath.Join(root, ".incoming"))
	if err != nil {
		t.Fatal(err)
	}
	if len(incoming) != 0 {
		t.Fatalf("rejected upload left incoming data: %v", incoming)
	}
}

func TestIncomingSpoolRejectsSymlink(t *testing.T) {
	root := t.TempDir()
	target := t.TempDir()
	if err := os.Symlink(target, filepath.Join(root, ".incoming")); err != nil {
		t.Fatal(err)
	}
	body, contentType := sampleFirstBody(t, "amiga", []byte("fixture"))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/submissions", body)
	req.Header.Set("Content-Type", contentType)
	rr := httptest.NewRecorder()
	newHandler(testConfig(root, 4096)).ServeHTTP(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	entries, err := os.ReadDir(target)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("symlink target received data: %v", entries)
	}
}
