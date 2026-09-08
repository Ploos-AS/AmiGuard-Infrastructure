# AmiGuard Infrastructure

Shared infrastructure contracts and deployment foundations for the AmiGuard ecosystem.

AmiGuard Infrastructure is intentionally separate from scanner/appliance code. It defines stable shared boundaries for storage, component identities, deployment assumptions, health contracts, quarantine intake and qualification gates.

## Status

M0 establishes the repository and infrastructure contract.

M1 adds the first deployable submission-service slice for a small VPS: a minimal Go service, OCI Containerfile, rootless Podman Quadlet and Caddy reverse-proxy/TLS example.

M2 adds static hardening validation and an actual rootless Podman runtime qualification in CI. The target remains 1 vCPU / 1 GiB RAM; the application container is limited to 192 MiB, runs read-only with all capabilities dropped, no-new-privileges enabled, and publishes only to loopback behind Caddy.

M3 implements the controlled quarantine protocol: explicit consent, streaming size limits, 128-bit server-generated IDs, SHA-256 while receiving, `0600` storage, atomic sample/metadata commits, and no retrieval endpoint. Client filenames are neither used as paths nor retained in the M3 receipt metadata.

M4 qualifies the real closed production VPS at `amiguard.ploos.no`: hardened SSH/firewall, DNS/TLS/Caddy, rootless Podman reboot persistence, dedicated 4 GiB `nodev,nosuid,noexec` quarantine storage, rootless UID mapping, writable quarantine bind-mount, and a controlled local end-to-end upload/hash/metadata test.

M5 adds public-intake abuse hardening in the application: upload attempts are rate-limited per client, simultaneous uploads are globally bounded, and forwarded client addresses are trusted only when the immediate peer is loopback Caddy. The existing 16 MiB sample bound, finite HTTP timeouts, consent requirement and strict multipart contract remain in force.

M5.2 adds `amiguard-admin`, a local/SSH-only quarantine CLI for metadata dashboard/list/show, sample hash verification and controlled no-overwrite export. No web-admin or retrieval endpoint is introduced.

M5.3 defines the initial production operations policy for retention/deletion, hostile-content backup handling, verified export, log/privacy hygiene, emergency upload shutdown and the final public go-live gate.

M5.4 replaces qualification-only landing-page wording with a production submission form and consent/privacy/retention contract. The Caddy policy permits only same-origin form submission while retaining a default-deny CSP.

M5.5 records the final public go-live qualification. A harmless fixture was submitted through the public HTTPS endpoint, its server receipt matched the local SHA-256, `amiguard-admin` verified the quarantined bytes and metadata, public retrieval routes remained unavailable, and the fixture was removed after qualification.

M5.6 adds a proper browser receipt after successful form submission while preserving the JSON receipt for API clients through `Accept` negotiation. The change was qualified in CI and then in production over public HTTPS; the receipt showed the server-generated submission ID and SHA-256, the quarantined bytes verified exactly, and the qualification fixture was removed afterward.

M5.7 adds capacity and distributed-abuse protection: quarantine storage fails closed before the dedicated filesystem reaches its configured reserve/high-watermark, and all clients share a configurable global upload-byte budget. The production contract uses a 512 MiB free-space floor, 85% used-space watermark and 256 MiB/hour global request budget. Production qualification passed on the dedicated VPS: normal public intake verified exact bytes and metadata, a controlled storage-pressure test returned HTTP 503 with `Retry-After: 3600`, and the normal production configuration was restored afterward.

M5.8 tightens global byte-budget accounting: the conservative maximum request charge is derived from the configured sample-size limit instead of a fixed 16 MiB assumption, and requests rejected solely because all concurrency slots are occupied no longer consume the global byte budget. The public write-only boundary and production quota values are unchanged.

**Public sample intake is live at `https://amiguard.ploos.no/`.** The active production deployment has `AMIGUARD_UPLOAD_ENABLED=true`. Intake remains write-only from the public side; administration and verified export remain local/SSH-only.

## Validation

Requires Python 3.11+ and Go 1.23+.

```sh
make check
```

With Podman and curl installed, run the rootless runtime gate as a normal non-root user:

```sh
make rootless-qualify
```

Optional local image build:

```sh
make container-build
```

## Scope

This repository owns shared infrastructure definitions and deployment contracts. It does not own malware samples, AmiGuard scanner implementation, historical antivirus engines, signature research conclusions, or the end-user appliance UI. Samples must never be committed here.

Hostile content accepted in production must live only in dedicated quarantine storage outside Git and must never be executed, extracted or served by the intake service.

## Deployment model

```text
Internet
   |
 HTTPS
   |
Caddy (host)
   |
127.0.0.1:8080
   |
rootless Podman
   |
amiguard-submit
   |
/data/quarantine
```

Administration remains separate:

```text
SSH -> amiguard-admin -> quarantine -> verified export -> isolated research environment
```

See `docs/M1_ROOTLESS_SUBMIT_SLICE.md`, `docs/M2_ROOTLESS_RUNTIME_HARDENING.md`, `docs/M3_QUARANTINE_PROTOCOL.md`, `docs/M4_PRODUCTION_VPS_QUALIFICATION.md`, `docs/M5_EDGE_ABUSE_HARDENING.md`, `docs/M5_ADMIN_CLI.md`, `docs/M5_3_PUBLIC_INTAKE_OPERATIONS.md`, `docs/M5_4_PUBLIC_LANDING_PAGE.md`, `docs/M5_5_PUBLIC_GO_LIVE_QUALIFICATION.md`, `docs/M5_6_BROWSER_RECEIPT_QUALIFICATION.md`, `docs/M5_7_CAPACITY_ABUSE_HARDENING.md`, `docs/M5_7_CAPACITY_ABUSE_QUALIFICATION.md` and `docs/M5_8_UPLOAD_BUDGET_ACCOUNTING.md`.

## Next milestone

M5.8 code and documentation are implemented. The remaining M5.8 gates are exact-HEAD CI followed by deployment and focused production runtime qualification on the dedicated VPS. CAPTCHA remains optional and should only be added if real abuse justifies the extra user/privacy cost.

## License

MIT. See `LICENSE`.