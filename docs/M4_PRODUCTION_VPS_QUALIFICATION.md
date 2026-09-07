# M4 Production VPS Qualification

M4 qualifies the real AmiGuard submission VPS while keeping public sample intake disabled.

## Qualified host

- Hostname: `amiguard.ploos.no`
- OS: Ubuntu 24.04 LTS
- Public edge: Caddy on TCP 80/443 with automatic TLS
- Application: rootless Podman under dedicated `amiguard` account
- Application bind: `127.0.0.1:8080` only
- Public upload state: disabled

## Host hardening evidence

The production host was qualified with:

- key-only SSH
- root SSH disabled
- password and keyboard-interactive SSH disabled
- UFW default-deny incoming
- public listeners limited to SSH and Caddy
- Fail2ban enabled for SSH
- dedicated non-login `amiguard` service account
- rootless Podman with cgroup v2
- systemd user lingering for reboot persistence

## Runtime hardening evidence

The live container was verified with:

- `USER 65532:65532`
- read-only container root filesystem
- `no-new-privileges`
- all Linux capabilities dropped; effective capability set reported `none`
- `PidsLimit=64`
- `MemoryMax=192M`
- rootless user namespace using `keep-id:uid=65532,gid=65532`
- application port published only to `127.0.0.1:8080`
- health endpoint returning HTTP 200
- public upload endpoint returning HTTP 404 while disabled

The service survived reboot and restarted automatically through the user systemd/Quadlet deployment.

## Quarantine filesystem

The production quarantine is a dedicated 4 GiB ext4 loopback filesystem mounted at:

`/data/amiguard/quarantine`

It is mounted with:

- `nodev`
- `nosuid`
- `noexec`

The mount root is mode `0700` and owned by `amiguard:amiguard`. The loopback backing file is root-owned and mode `0600`.

The filesystem mount and the application service were both verified to persist correctly across reboot.

## Container mapping

The production Quadlet maps:

`/data/amiguard/quarantine` -> `/data/quarantine`

The application receives:

- `AMIGUARD_UPLOAD_ENABLED=false`
- `AMIGUARD_QUARANTINE_ROOT=/data/quarantine`
- `AMIGUARD_MAX_UPLOAD_BYTES=16777216`

A rootless Podman write test verified that container UID/GID `65532:65532` can create, read and remove files in the quarantine without weakening the host directory permissions.

## Controlled local upload qualification

A harmless 36-byte qualification fixture was submitted while uploads were enabled temporarily for the test.

Expected SHA-256:

`806540a57f89198ddf62b3a1715638935b83c585304bdb9cc8410f961b2a014c`

The resulting quarantine sample had the same SHA-256. The sample and metadata files were both mode `0600` and owned by `amiguard:amiguard`.

Receipt metadata recorded:

- `consent=true`
- `executed=false`
- `extracted=false`

After qualification, uploads were disabled again, the public endpoint returned HTTP 404, health returned HTTP 200, and the qualification sample and metadata were removed from quarantine.

## DNS and TLS

Public DNS resolves `amiguard.ploos.no` to the VPS IPv4 address. No public AAAA record was present during qualification.

Caddy successfully obtained a publicly trusted certificate and provided:

- HTTP to HTTPS redirect
- HTTP/2 and HTTP/3 edge service
- security headers
- reverse proxy only to `127.0.0.1:8080`

## M4 result

**PASS** for the current closed production deployment.

This does not authorize public sample uploads. Public upload remains an explicit later gate.

## Remaining gates before public intake

Before `AMIGUARD_UPLOAD_ENABLED=true` may become the production default, the platform still needs:

- request/body abuse controls at the edge
- rate limiting or equivalent anti-abuse controls
- application/host egress minimization
- log hygiene and privacy review
- retention/deletion policy
- manual export procedure with SHA-256 verification before and after transfer
- backup policy that treats stored samples as hostile content
- incident recovery/runbook
- final public-intake qualification
