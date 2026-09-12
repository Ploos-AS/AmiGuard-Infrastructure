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
	routingKind           = "amiguard-asw-routing-manifest"
)

var platformNamespaces = map[string]string{
	"amiga":    "amiga",
	"atari-st": "atari",
	"mac68k":   "mac68k",
}

type submissionMetadata struct {
	SchemaVersion int    `json:"schema_version"`
	Kind          string `json:"kind"`
	SubmissionID  string `json:"submission_id"`
	Platform      string `json:"platform,omitempty"`
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

type routingManifest struct {
	SchemaVersion int    `json:"schema_version"`
	Kind          string `json:"kind"`
	SubmissionID  string `json:"submission_id"`
	Platform      string `json:"platform"`
	ASWNamespace  string `json:"asw_namespace"`
	SHA256        string `json:"sha256"`
	Size          int64  `json:"size"`
	ReceivedAt    string `json:"received_at"`
	RoutedAt      string `json:"routed_at"`
	Source        string `json:"source"`
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
	switch args[0] {
	case "dashboard":
		if len(args) != 1 { return errors.New("usage: amiguard-admin dashboard") }
		return dashboard(stdout, root)
	case "list":
		if len(args) != 1 { return errors.New("usage: amiguard-admin list") }
		return list(stdout, root)
	case "show":
		if len(args) != 2 { return errors.New("usage: amiguard-admin show <submission-id>") }
		rec, err := loadRecord(root, args[1]); if err != nil { return err }
		enc := json.NewEncoder(stdout); enc.SetIndent("", "  "); return enc.Encode(rec.Metadata)
	case "verify":
		if len(args) != 2 { return errors.New("usage: amiguard-admin verify <submission-id>") }
		return verifyCommand(stdout, root, args[1])
	case "export":
		if len(args) != 3 { return errors.New("usage: amiguard-admin export <submission-id> <existing-directory>") }
		return exportCommand(stdout, root, args[1], args[2])
	case "route":
		if len(args) != 3 { return errors.New("usage: amiguard-admin route <submission-id> <existing-asw-inbox-root>") }
		return routeCommand(stdout, root, args[1], args[2])
	case "help", "-h", "--help":
		printUsage(stdout); return nil
	default:
		printUsage(stderr); return fmt.Errorf("unknown command %q", args[0])
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
	fmt.Fprintln(w, "  amiguard-admin route <submission-id> <existing-asw-inbox-root>")
}

func dashboard(w io.Writer, root string) error {
	records, err := loadRecords(root); if err != nil { return err }
	var total int64; counts := map[string]int{}
	for _, rec := range records { total += rec.Metadata.Size; counts[rec.Metadata.Platform]++ }
	fmt.Fprintln(w, "AmiGuard Admin")
	fmt.Fprintln(w, "────────────────────────────────────────────────────────────────────────────")
	fmt.Fprintf(w, "Quarantine: %d submissions   %s   amiga=%d atari-st=%d mac68k=%d\n\n", len(records), humanBytes(total), counts["amiga"], counts["atari-st"], counts["mac68k"])
	return writeRecordsTable(w, records)
}

func list(w io.Writer, root string) error { records, err := loadRecords(root); if err != nil { return err }; return writeRecordsTable(w, records) }

func writeRecordsTable(w io.Writer, records []record) error {
	fmt.Fprintln(w, "ID                               PLATFORM  RECEIVED                  SIZE        SHA-256")
	for _, rec := range records {
		fmt.Fprintf(w, "%-32s %-9s %-25s %-11s %.16s…\n", rec.Metadata.SubmissionID, rec.Metadata.Platform, rec.Metadata.ReceivedAt, humanBytes(rec.Metadata.Size), rec.Metadata.SHA256)
	}
	return nil
}

func loadRecords(root string) ([]record, error) {
	if err := validateRoot(root); err != nil { return nil, err }
	var records []record
	for platform := range platformNamespaces {
		dir := filepath.Join(root, platform)
		info, err := os.Lstat(dir)
		if os.IsNotExist(err) { continue }
		if err != nil { return nil, err }
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() { return nil, fmt.Errorf("platform namespace %s must be a non-symlink directory", platform) }
		entries, err := os.ReadDir(dir); if err != nil { return nil, err }
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") || strings.HasPrefix(entry.Name(), ".") { continue }
			id := strings.TrimSuffix(entry.Name(), ".json"); if !validID(id) { continue }
			rec, err := loadRecordAt(root, platform, id); if err != nil { return nil, fmt.Errorf("load %s/%s: %w", platform, id, err) }
			records = append(records, rec)
		}
	}
	// Legacy schema-v1 records at quarantine root remain readable as Amiga.
	entries, err := os.ReadDir(root); if err != nil { return nil, err }
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") || strings.HasPrefix(entry.Name(), ".") { continue }
		id := strings.TrimSuffix(entry.Name(), ".json"); if !validID(id) { continue }
		rec, err := loadLegacyRecord(root, id); if err != nil { return nil, fmt.Errorf("load legacy %s: %w", id, err) }
		records = append(records, rec)
	}
	sort.Slice(records, func(i, j int) bool { return records[i].Metadata.ReceivedAt > records[j].Metadata.ReceivedAt })
	return records, nil
}

func loadRecord(root, id string) (record, error) {
	if err := validateRoot(root); err != nil { return record{}, err }
	if !validID(id) { return record{}, errors.New("submission id must be 32 lowercase hexadecimal characters") }
	var found *record
	for platform := range platformNamespaces {
		metadataPath := filepath.Join(root, platform, id+".json")
		if _, err := os.Lstat(metadataPath); os.IsNotExist(err) { continue }
		rec, err := loadRecordAt(root, platform, id); if err != nil { return record{}, err }
		if found != nil { return record{}, errors.New("submission id exists in multiple platform namespaces") }
		copy := rec; found = &copy
	}
	legacyPath := filepath.Join(root, id+".json")
	if _, err := os.Lstat(legacyPath); err == nil {
		rec, err := loadLegacyRecord(root, id); if err != nil { return record{}, err }
		if found != nil { return record{}, errors.New("submission id exists in both namespaced and legacy quarantine") }
		return rec, nil
	}
	if found == nil { return record{}, os.ErrNotExist }
	return *found, nil
}

func decodeMetadata(path string) (submissionMetadata, error) {
	if err := requireRegularFile(path); err != nil { return submissionMetadata{}, fmt.Errorf("metadata: %w", err) }
	f, err := os.Open(path); if err != nil { return submissionMetadata{}, err }; defer f.Close()
	dec := json.NewDecoder(io.LimitReader(f, 64<<10)); dec.DisallowUnknownFields()
	var metadata submissionMetadata; if err := dec.Decode(&metadata); err != nil { return submissionMetadata{}, fmt.Errorf("decode metadata: %w", err) }
	return metadata, nil
}

func validateMetadata(metadata submissionMetadata, id, platform string, schema int) error {
	if metadata.SchemaVersion != schema || metadata.Kind != metadataKind || metadata.SubmissionID != id { return errors.New("metadata contract mismatch") }
	if schema == 2 && metadata.Platform != platform { return errors.New("metadata platform mismatch") }
	if schema == 1 { metadata.Platform = "amiga" }
	if !validSHA256(metadata.SHA256) || metadata.Size < 0 { return errors.New("invalid metadata hash or size") }
	if _, err := time.Parse(time.RFC3339Nano, metadata.ReceivedAt); err != nil { return errors.New("invalid metadata received_at") }
	if !metadata.Consent || metadata.Executed || metadata.Extracted { return errors.New("unexpected quarantine state") }
	return nil
}

func loadRecordAt(root, platform, id string) (record, error) {
	if _, ok := platformNamespaces[platform]; !ok { return record{}, errors.New("unsupported platform") }
	if !validID(id) { return record{}, errors.New("invalid submission id") }
	dir := filepath.Join(root, platform)
	metadata, err := decodeMetadata(filepath.Join(dir, id+".json")); if err != nil { return record{}, err }
	if err := validateMetadata(metadata, id, platform, 2); err != nil { return record{}, err }
	sample := filepath.Join(dir, id+".sample"); if err := requireRegularFile(sample); err != nil { return record{}, fmt.Errorf("sample: %w", err) }
	return record{Metadata: metadata, Sample: sample}, nil
}

func loadLegacyRecord(root, id string) (record, error) {
	metadata, err := decodeMetadata(filepath.Join(root, id+".json")); if err != nil { return record{}, err }
	if err := validateMetadata(metadata, id, "amiga", 1); err != nil { return record{}, err }
	metadata.Platform = "amiga"
	sample := filepath.Join(root, id+".sample"); if err := requireRegularFile(sample); err != nil { return record{}, fmt.Errorf("sample: %w", err) }
	return record{Metadata: metadata, Sample: sample}, nil
}

func verifyRecord(rec record) (string, int64, error) {
	digest, size, err := hashFile(rec.Sample); if err != nil { return "", 0, err }
	if digest != rec.Metadata.SHA256 || size != rec.Metadata.Size { return "", 0, fmt.Errorf("verification failed: metadata sha256=%s size=%d, sample sha256=%s size=%d", rec.Metadata.SHA256, rec.Metadata.Size, digest, size) }
	return digest, size, nil
}

func verifyCommand(w io.Writer, root, id string) error {
	rec, err := loadRecord(root, id); if err != nil { return err }
	digest, size, err := verifyRecord(rec); if err != nil { return err }
	fmt.Fprintf(w, "VERIFIED %s platform=%s sha256=%s size=%d\n", id, rec.Metadata.Platform, digest, size); return nil
}

func copyVerifiedSample(rec record, destinationPath string) error {
	out, err := os.OpenFile(destinationPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600); if err != nil { return err }
	remove := true
	defer func(){ _ = out.Close(); if remove { _ = os.Remove(destinationPath) } }()
	in, err := os.Open(rec.Sample); if err != nil { return err }
	h := sha256.New(); copied, err := io.Copy(io.MultiWriter(out, h), in); _ = in.Close(); if err != nil { return err }
	if err := out.Sync(); err != nil { return err }; if err := out.Close(); err != nil { return err }
	if copied != rec.Metadata.Size || hex.EncodeToString(h.Sum(nil)) != rec.Metadata.SHA256 { return errors.New("export stream verification failed") }
	digest, size, err := hashFile(destinationPath); if err != nil { return err }
	if digest != rec.Metadata.SHA256 || size != rec.Metadata.Size { return errors.New("post-export verification failed") }
	remove = false; return nil
}

func exportCommand(w io.Writer, root, id, destination string) error {
	rec, err := loadRecord(root, id); if err != nil { return err }
	if err := requireDirectory(destination); err != nil { return fmt.Errorf("export destination: %w", err) }
	if _, _, err := verifyRecord(rec); err != nil { return errors.New("source verification failed; refusing export") }
	destinationPath := filepath.Join(destination, id+".sample")
	if err := copyVerifiedSample(rec, destinationPath); err != nil { return fmt.Errorf("export: %w", err) }
	fmt.Fprintf(w, "EXPORTED %s platform=%s -> %s sha256=%s size=%d\n", id, rec.Metadata.Platform, destinationPath, rec.Metadata.SHA256, rec.Metadata.Size); return nil
}

func routeCommand(w io.Writer, root, id, inboxRoot string) error {
	rec, err := loadRecord(root, id); if err != nil { return err }
	if _, _, err := verifyRecord(rec); err != nil { return errors.New("source verification failed; refusing route") }
	if err := requireDirectory(inboxRoot); err != nil { return fmt.Errorf("ASW inbox root: %w", err) }
	namespace, ok := platformNamespaces[rec.Metadata.Platform]; if !ok { return errors.New("unsupported platform in metadata") }
	namespaceDir := filepath.Join(inboxRoot, namespace)
	if err := requireDirectory(namespaceDir); err != nil { return fmt.Errorf("ASW namespace: %w", err) }
	samplePath := filepath.Join(namespaceDir, id+".sample")
	manifestPath := filepath.Join(namespaceDir, id+".route.json")
	if _, err := os.Lstat(manifestPath); err == nil { return errors.New("routing manifest already exists; refusing duplicate route") } else if !os.IsNotExist(err) { return err }
	if err := copyVerifiedSample(rec, samplePath); err != nil { return fmt.Errorf("route sample: %w", err) }
	manifest := routingManifest{1, routingKind, id, rec.Metadata.Platform, namespace, rec.Metadata.SHA256, rec.Metadata.Size, rec.Metadata.ReceivedAt, time.Now().UTC().Format(time.RFC3339Nano), "amiguard-public-quarantine"}
	if err := writeJSONExclusive(manifestPath, manifest); err != nil { _ = os.Remove(samplePath); return fmt.Errorf("route manifest: %w", err) }
	fmt.Fprintf(w, "ROUTED %s platform=%s namespace=%s sha256=%s size=%d\n", id, rec.Metadata.Platform, namespace, rec.Metadata.SHA256, rec.Metadata.Size)
	return nil
}

func writeJSONExclusive(path string, value any) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600); if err != nil { return err }
	remove := true; defer func(){ _ = f.Close(); if remove { _ = os.Remove(path) } }()
	enc := json.NewEncoder(f); enc.SetEscapeHTML(true); if err := enc.Encode(value); err != nil { return err }
	if err := f.Sync(); err != nil { return err }; if err := f.Close(); err != nil { return err }; remove = false; return nil
}

func hashFile(path string) (string, int64, error) { if err := requireRegularFile(path); err != nil { return "",0,err }; f,err:=os.Open(path); if err!=nil{return "",0,err}; defer f.Close(); h:=sha256.New(); n,err:=io.Copy(h,f); if err!=nil{return "",0,err}; return hex.EncodeToString(h.Sum(nil)),n,nil }
func validateRoot(path string) error { info,err:=os.Lstat(path); if err!=nil{return fmt.Errorf("quarantine root: %w",err)}; if info.Mode()&os.ModeSymlink!=0||!info.IsDir(){return errors.New("quarantine root must be an existing non-symlink directory")}; return nil }
func requireDirectory(path string) error { info,err:=os.Lstat(path); if err!=nil{return err}; if info.Mode()&os.ModeSymlink!=0||!info.IsDir(){return errors.New("must be an existing non-symlink directory")}; return nil }
func requireRegularFile(path string) error { info,err:=os.Lstat(path); if err!=nil{return err}; if info.Mode()&os.ModeSymlink!=0||!info.Mode().IsRegular(){return errors.New("must be a regular non-symlink file")}; return nil }
func validID(value string) bool { if len(value)!=32||strings.ToLower(value)!=value{return false}; decoded,err:=hex.DecodeString(value); return err==nil&&len(decoded)==16 }
func validSHA256(value string) bool { if len(value)!=64||strings.ToLower(value)!=value{return false}; decoded,err:=hex.DecodeString(value); return err==nil&&len(decoded)==32 }
func humanBytes(value int64) string { const unit=int64(1024); if value<unit{return fmt.Sprintf("%d B",value)}; div,exp:=unit,0; for n:=value/unit;n>=unit&&exp<3;n/=unit{div*=unit;exp++}; return fmt.Sprintf("%.1f %ciB",float64(value)/float64(div),"KMGT"[exp]) }
func env(key,fallback string) string { if value:=os.Getenv(key);value!=""{return value};return fallback }
