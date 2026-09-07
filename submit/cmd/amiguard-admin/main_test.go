package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDashboardListShowVerifyExport(t *testing.T) {
	root := t.TempDir()
	destination := t.TempDir()
	id := "0123456789abcdef0123456789abcdef"
	payload := []byte("AmiGuard harmless admin qualification fixture\n")
	writeFixture(t, root, id, payload)
	t.Setenv("AMIGUARD_ADMIN_QUARANTINE_ROOT", root)

	var out bytes.Buffer
	if err := run([]string{"dashboard"}, &out, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Quarantine: 1 submissions") || !strings.Contains(out.String(), id) {
		t.Fatalf("unexpected dashboard: %s", out.String())
	}

	out.Reset()
	if err := run([]string{"list"}, &out, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), id) {
		t.Fatalf("unexpected list: %s", out.String())
	}

	out.Reset()
	if err := run([]string{"show", id}, &out, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"submission_id": "`+id+`"`) {
		t.Fatalf("unexpected show: %s", out.String())
	}

	out.Reset()
	if err := run([]string{"verify", id}, &out, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "VERIFIED "+id) {
		t.Fatalf("unexpected verify: %s", out.String())
	}

	out.Reset()
	if err := run([]string{"export", id, destination}, &out, &out); err != nil {
		t.Fatal(err)
	}
	exported := filepath.Join(destination, id+".sample")
	got, err := os.ReadFile(exported)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatal("exported bytes changed")
	}
	info, err := os.Stat(exported)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("export mode = %o", info.Mode().Perm())
	}
	if !strings.Contains(out.String(), "EXPORTED "+id) {
		t.Fatalf("unexpected export output: %s", out.String())
	}
}

func TestVerifyRejectsTamperedSample(t *testing.T) {
	root := t.TempDir()
	id := "11111111111111111111111111111111"
	writeFixture(t, root, id, []byte("original"))
	if err := os.WriteFile(filepath.Join(root, id+".sample"), []byte("tampered"), 0600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := runWithRoot(root, []string{"verify", id}, &out); err == nil || !strings.Contains(err.Error(), "verification failed") {
		t.Fatalf("expected verification failure, got %v", err)
	}
}

func TestExportRefusesOverwrite(t *testing.T) {
	root := t.TempDir()
	destination := t.TempDir()
	id := "22222222222222222222222222222222"
	writeFixture(t, root, id, []byte("fixture"))
	t.Setenv("AMIGUARD_ADMIN_QUARANTINE_ROOT", root)
	path := filepath.Join(destination, id+".sample")
	if err := os.WriteFile(path, []byte("existing"), 0600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := run([]string{"export", id, destination}, &out, &out); err == nil {
		t.Fatal("expected overwrite refusal")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "existing" {
		t.Fatal("existing export was modified")
	}
}

func TestRejectsInvalidIDAndSymlinkDestination(t *testing.T) {
	root := t.TempDir()
	t.Setenv("AMIGUARD_ADMIN_QUARANTINE_ROOT", root)
	var out bytes.Buffer
	if err := run([]string{"show", "../escape"}, &out, &out); err == nil {
		t.Fatal("expected invalid id rejection")
	}

	target := t.TempDir()
	link := filepath.Join(t.TempDir(), "export-link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	id := "33333333333333333333333333333333"
	writeFixture(t, root, id, []byte("fixture"))
	if err := run([]string{"export", id, link}, &out, &out); err == nil {
		t.Fatal("expected symlink destination rejection")
	}
}

func runWithRoot(root string, args []string, out *bytes.Buffer) error {
	old, had := os.LookupEnv("AMIGUARD_ADMIN_QUARANTINE_ROOT")
	_ = os.Setenv("AMIGUARD_ADMIN_QUARANTINE_ROOT", root)
	defer func() {
		if had {
			_ = os.Setenv("AMIGUARD_ADMIN_QUARANTINE_ROOT", old)
		} else {
			_ = os.Unsetenv("AMIGUARD_ADMIN_QUARANTINE_ROOT")
		}
	}()
	return run(args, out, out)
}

func writeFixture(t *testing.T, root, id string, payload []byte) {
	t.Helper()
	sum := sha256.Sum256(payload)
	metadata := submissionMetadata{
		SchemaVersion: 1,
		Kind:          metadataKind,
		SubmissionID:  id,
		SHA256:        hex.EncodeToString(sum[:]),
		Size:          int64(len(payload)),
		ReceivedAt:    time.Now().UTC().Format(time.RFC3339Nano),
		Consent:       true,
		Executed:      false,
		Extracted:     false,
	}
	if err := os.WriteFile(filepath.Join(root, id+".sample"), payload, 0600); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(metadata)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, id+".json"), append(data, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
}
