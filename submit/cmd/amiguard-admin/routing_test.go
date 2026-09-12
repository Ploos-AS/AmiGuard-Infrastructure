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

func writeNamespacedFixture(t *testing.T, root, platform, id string, payload []byte) {
	t.Helper()
	dir := filepath.Join(root, platform)
	if err := os.MkdirAll(dir, 0700); err != nil { t.Fatal(err) }
	sum := sha256.Sum256(payload)
	metadata := submissionMetadata{
		SchemaVersion: 2,
		Kind: metadataKind,
		SubmissionID: id,
		Platform: platform,
		SHA256: hex.EncodeToString(sum[:]),
		Size: int64(len(payload)),
		ReceivedAt: time.Now().UTC().Format(time.RFC3339Nano),
		Consent: true,
		Executed: false,
		Extracted: false,
	}
	if err := os.WriteFile(filepath.Join(dir, id+".sample"), payload, 0600); err != nil { t.Fatal(err) }
	data, err := json.Marshal(metadata); if err != nil { t.Fatal(err) }
	if err := os.WriteFile(filepath.Join(dir, id+".json"), append(data, '\n'), 0600); err != nil { t.Fatal(err) }
}

func TestRouteUsesPlatformNamespaceAndManifest(t *testing.T) {
	cases := []struct{ platform, namespace, id string }{
		{"amiga", "amiga", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		{"atari-st", "atari", "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
		{"mac68k", "mac68k", "cccccccccccccccccccccccccccccccc"},
	}
	for _, tc := range cases {
		t.Run(tc.platform, func(t *testing.T) {
			root := t.TempDir(); inbox := t.TempDir()
			if err := os.Mkdir(filepath.Join(inbox, tc.namespace), 0700); err != nil { t.Fatal(err) }
			payload := []byte("harmless "+tc.platform+" routing fixture\n")
			writeNamespacedFixture(t, root, tc.platform, tc.id, payload)
			var out bytes.Buffer
			if err := runWithRoot(root, []string{"route", tc.id, inbox}, &out); err != nil { t.Fatal(err) }
			if !strings.Contains(out.String(), "namespace="+tc.namespace) { t.Fatalf("unexpected output: %s", out.String()) }
			samplePath := filepath.Join(inbox, tc.namespace, tc.id+".sample")
			got, err := os.ReadFile(samplePath); if err != nil { t.Fatal(err) }
			if !bytes.Equal(got, payload) { t.Fatal("routed bytes changed") }
			manifestBytes, err := os.ReadFile(filepath.Join(inbox, tc.namespace, tc.id+".route.json")); if err != nil { t.Fatal(err) }
			var manifest routingManifest
			if err := json.Unmarshal(manifestBytes, &manifest); err != nil { t.Fatal(err) }
			if manifest.Platform != tc.platform || manifest.ASWNamespace != tc.namespace || manifest.SubmissionID != tc.id || manifest.Kind != routingKind { t.Fatalf("bad manifest: %+v", manifest) }
			sum := sha256.Sum256(payload)
			if manifest.SHA256 != hex.EncodeToString(sum[:]) || manifest.Size != int64(len(payload)) { t.Fatalf("bad hash binding: %+v", manifest) }
		})
	}
}

func TestRouteRejectsTamperedSample(t *testing.T) {
	root := t.TempDir(); inbox := t.TempDir(); id := "dddddddddddddddddddddddddddddddd"
	if err := os.Mkdir(filepath.Join(inbox, "atari"), 0700); err != nil { t.Fatal(err) }
	writeNamespacedFixture(t, root, "atari-st", id, []byte("original"))
	if err := os.WriteFile(filepath.Join(root, "atari-st", id+".sample"), []byte("tampered"), 0600); err != nil { t.Fatal(err) }
	var out bytes.Buffer
	if err := runWithRoot(root, []string{"route", id, inbox}, &out); err == nil || !strings.Contains(err.Error(), "source verification failed") { t.Fatalf("expected verification rejection, got %v", err) }
	entries, err := os.ReadDir(filepath.Join(inbox, "atari")); if err != nil { t.Fatal(err) }
	if len(entries) != 0 { t.Fatalf("tampered sample created route files: %v", entries) }
}

func TestRouteRequiresExistingNamespaceAndRefusesDuplicate(t *testing.T) {
	root := t.TempDir(); inbox := t.TempDir(); id := "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
	writeNamespacedFixture(t, root, "mac68k", id, []byte("fixture"))
	var out bytes.Buffer
	if err := runWithRoot(root, []string{"route", id, inbox}, &out); err == nil { t.Fatal("expected missing namespace rejection") }
	if err := os.Mkdir(filepath.Join(inbox, "mac68k"), 0700); err != nil { t.Fatal(err) }
	if err := runWithRoot(root, []string{"route", id, inbox}, &out); err != nil { t.Fatal(err) }
	if err := runWithRoot(root, []string{"route", id, inbox}, &out); err == nil { t.Fatal("expected duplicate route rejection") }
}

func TestLegacySchemaOneRoutesAsAmiga(t *testing.T) {
	root := t.TempDir(); inbox := t.TempDir(); id := "ffffffffffffffffffffffffffffffff"
	if err := os.Mkdir(filepath.Join(inbox, "amiga"), 0700); err != nil { t.Fatal(err) }
	writeFixture(t, root, id, []byte("legacy amiga fixture"))
	var out bytes.Buffer
	if err := runWithRoot(root, []string{"route", id, inbox}, &out); err != nil { t.Fatal(err) }
	if !strings.Contains(out.String(), "platform=amiga namespace=amiga") { t.Fatalf("unexpected output: %s", out.String()) }
}
