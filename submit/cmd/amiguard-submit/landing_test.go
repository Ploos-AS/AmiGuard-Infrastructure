package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEnabledLandingPageHasProductionSubmissionForm(t *testing.T) {
	cfg := config{uploadEnabled: true, quarantineRoot: t.TempDir(), maxUploadBytes: 16 << 20}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	newHandler(cfg).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("got status %d", rr.Code)
	}
	body := rr.Body.String()
	for _, want := range []string{
		`method="post"`,
		`action="/api/v1/submissions"`,
		`enctype="multipart/form-data"`,
		`name="sample"`,
		`name="consent"`,
		`value="true"`,
		"Maximum sample size: 16 MiB",
		"90 days",
		"not made available through a public download endpoint",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("enabled landing page missing %q", want)
		}
	}
	for _, forbidden := range []string{
		"controlled qualification",
		"must not be exposed publicly",
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("enabled landing page still contains qualification wording %q", forbidden)
		}
	}
}

func TestDisabledLandingPageHasNoUploadForm(t *testing.T) {
	cfg := config{uploadEnabled: false, quarantineRoot: t.TempDir(), maxUploadBytes: 16 << 20}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	newHandler(cfg).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("got status %d", rr.Code)
	}
	body := rr.Body.String()
	if strings.Contains(body, `<form`) || strings.Contains(body, `name="sample"`) {
		t.Fatal("disabled landing page exposes an upload form")
	}
	if !strings.Contains(body, "currently closed") {
		t.Fatal("disabled landing page does not clearly state closed status")
	}
}
