package main

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func makeSubmissionRequest(t *testing.T, accept string) *http.Request {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	if err := mw.WriteField("consent", "true"); err != nil {
		t.Fatal(err)
	}
	part, err := mw.CreateFormFile("sample", "fixture.bin")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("harmless receipt test fixture\n")); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/submissions", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	return req
}

func TestBrowserSubmissionGetsHTMLReceipt(t *testing.T) {
	root := t.TempDir()
	cfg := testConfig(root, 16<<20)
	rr := httptest.NewRecorder()
	newHandler(cfg).ServeHTTP(rr, makeSubmissionRequest(t, "text/html,application/xhtml+xml"))

	if rr.Code != http.StatusCreated {
		t.Fatalf("got status %d", rr.Code)
	}
	if got := rr.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/html") {
		t.Fatalf("got content type %q", got)
	}
	body := rr.Body.String()
	for _, want := range []string{"Submission received", "Submission ID", "SHA-256", "Size", "Received", "Submit another sample"} {
		if !strings.Contains(body, want) {
			t.Fatalf("HTML receipt missing %q", want)
		}
	}
}

func TestAPISubmissionKeepsJSONReceipt(t *testing.T) {
	root := t.TempDir()
	cfg := testConfig(root, 16<<20)
	rr := httptest.NewRecorder()
	newHandler(cfg).ServeHTTP(rr, makeSubmissionRequest(t, "application/json"))

	if rr.Code != http.StatusCreated {
		t.Fatalf("got status %d", rr.Code)
	}
	if got := rr.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("got content type %q", got)
	}
	var receipt submissionReceipt
	if err := json.Unmarshal(rr.Body.Bytes(), &receipt); err != nil {
		t.Fatal(err)
	}
	if len(receipt.ID) != 32 || len(receipt.SHA256) != 64 || receipt.Size == 0 || receipt.ReceivedAt == "" {
		t.Fatalf("invalid receipt: %+v", receipt)
	}
}
