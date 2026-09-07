# M3 — Quarantine storage and controlled submission protocol

M3 adds the storage and protocol boundary required for later community sample intake. It does **not** open the public submission channel.

## Safety posture

The default deployment remains closed: `AMIGUARD_UPLOAD_ENABLED=false`, and the normal M1/M2 Quadlet mounts no quarantine directory. M3 provides a separate staged Quadlet example for controlled qualification only. Do not expose that staged unit to the Internet until the production VPS hardening gate has passed.

The intake service never executes, parses, extracts, decompresses, scans, or serves submitted bytes. It only streams bytes into quarantine, calculates SHA-256, records minimal metadata, and returns an opaque receipt.

## Protocol

Controlled intake uses:

`POST /api/v1/submissions`

with `multipart/form-data` containing exactly:

- `consent=true`
- one `sample` file part

Unknown form fields are rejected. The client filename is deliberately not used as a storage path and is not retained in M3 metadata.

The default transport limit is 16 MiB. `AMIGUARD_MAX_UPLOAD_BYTES` may be set from 1 byte through 64 MiB. The request body is bounded independently of the file stream to limit multipart overhead.

A successful response is HTTP 201 with JSON containing only the server-generated submission ID, SHA-256, byte size and UTC receipt timestamp.

There is no retrieval API. `/api/v1/submissions/<id>` returns 404.

## Quarantine layout

For submission ID `<id>` the service writes:

- `<id>.sample` — original bytes, unchanged, mode `0600`
- `<id>.json` — minimal receipt metadata, mode `0600`

IDs are 128 bits from `crypto/rand`, hex encoded. Uploaded filenames never become filesystem names.

Sample writing is streamed through SHA-256 into an exclusive temporary file in the same directory. The file is size-limited, `fsync`ed, closed, then atomically renamed. Metadata is written through the same exclusive-temp + fsync + rename pattern. If metadata persistence fails after the sample rename, the sample is removed so the service does not intentionally leave a half-committed submission pair.

The metadata records `executed=false` and `extracted=false`. These are assertions about the intake service contract, not claims about what the submitted file contains.

## Runtime configuration

Closed/default mode:

```text
AMIGUARD_UPLOAD_ENABLED=false
```

Controlled M3 qualification mode additionally requires an existing, non-symlink quarantine directory and:

```text
AMIGUARD_UPLOAD_ENABLED=true
AMIGUARD_QUARANTINE_ROOT=/data/quarantine
AMIGUARD_MAX_UPLOAD_BYTES=16777216
```

See `deploy/quadlet/amiguard-submit-quarantine.container.example`. The normal `amiguard-submit.container` intentionally remains closed and has no writable quarantine mount.

## Qualification

M3 host/CI qualification uses harmless synthetic bytes only. `go test ./...` verifies at least:

1. uploads remain disabled by default;
2. unknown submission/retrieval paths return 404;
3. explicit consent is mandatory;
4. exactly one file is accepted;
5. oversize input is rejected without retained files;
6. client filenames are not used or retained;
7. stored bytes are unchanged;
8. SHA-256 and byte count match the input;
9. sample and metadata permissions are `0600`;
10. metadata says the intake path neither executed nor extracted the content.

The existing M2 rootless Podman runtime qualification remains in CI and exercises the closed deployment profile.

## Production gate still required

Before `AMIGUARD_UPLOAD_ENABLED=true` is used on a public VPS, separately qualify the actual host: SSH policy, firewall, DNS, TLS, loopback-only application exposure, rootless service ownership, quarantine ownership/modes, disk limits, log hygiene, rate limiting, retention/deletion policy, backups, manual export procedure, egress restrictions and incident recovery.

M3 is therefore a **protocol/storage implementation milestone**, not authorization to receive public malware samples.
