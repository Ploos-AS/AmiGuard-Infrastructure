# M5 Local Admin CLI

`amiguard-admin` is the local operator interface for the AmiGuard quarantine. It is intentionally not a web service and does not expose any listener or retrieval endpoint.

## Security boundary

The public path remains:

```text
Internet -> Caddy -> amiguard-submit -> quarantine
```

The administrative path is local to the VPS:

```text
SSH operator -> amiguard-admin -> quarantine -> verified export directory
```

The tool reads quarantine metadata and opaque sample bytes directly from the host filesystem. It does not parse, extract, execute or scan sample content.

## Commands

```text
amiguard-admin dashboard
amiguard-admin list
amiguard-admin show <submission-id>
amiguard-admin verify <submission-id>
amiguard-admin export <submission-id> <existing-directory>
```

The quarantine root defaults to `/data/amiguard/quarantine`. Tests or alternate deployments may override it with `AMIGUARD_ADMIN_QUARANTINE_ROOT`.

## Validation rules

A submission ID must be exactly 32 lowercase hexadecimal characters. Metadata must satisfy the M3 quarantine contract: schema version 1, the expected kind, matching submission ID, valid SHA-256 and size, valid timestamp, `consent=true`, `executed=false`, and `extracted=false`.

Metadata, sample files, the quarantine root and export destination must not be symlinks where the tool relies on them as trust boundaries.

## Verification

`verify` hashes the stored sample and compares both SHA-256 and byte count with the receipt metadata. A mismatch is a hard failure.

## Export

`export` performs a deliberately conservative copy:

1. validate metadata and quarantine state;
2. hash and size-check the source against metadata;
3. require an existing non-symlink destination directory;
4. create `<submission-id>.sample` with mode `0600` and `O_EXCL` so an existing file is never overwritten;
5. hash while copying;
6. fsync and close the export;
7. hash the completed destination again;
8. require source, stream, destination and metadata SHA-256/size to agree.

If verification fails during export, the incomplete destination is removed.

## Deliberate omissions

This first admin slice has no delete command, no mutation of quarantine metadata, no interactive TUI dependency, no web dashboard, no network service and no automatic sample analysis. Those require separate qualification.

The `dashboard` command is currently a safe terminal snapshot of queue count, total stored bytes and submission metadata. An interactive terminal UI can be added later without changing the quarantine or public-network trust boundaries.
