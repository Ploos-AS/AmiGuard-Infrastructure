# M2 — Rootless runtime hardening

M2 hardens and qualifies the M1 deployment shape before hostile sample upload is implemented.

## Scope

M2 retains the 1 vCPU / 1 GiB VPS target and keeps uploads disabled. It adds two qualification layers:

1. static validation of the committed Containerfile, Quadlet and Caddy security contract;
2. an actual rootless Podman runtime smoke test that builds and starts the container as a non-root user.

## Required runtime properties

The submit container must:

- run rootless under Podman;
- run as explicit container user `65532:65532`;
- use a read-only root filesystem;
- drop all Linux capabilities;
- enable no-new-privileges;
- have a 192 MiB memory limit;
- have a 64 PID limit;
- publish only to host loopback;
- expose no container-engine socket;
- avoid privileged and host-network modes.

Caddy remains the only intended public HTTP/TLS boundary and proxies to `127.0.0.1:8080`.

## Runtime qualification

Run as a normal non-root user with Podman and curl installed:

```sh
make rootless-qualify
```

The qualification script fails closed if the caller is root or Podman reports a non-rootless engine. It builds the OCI image, starts it with the hardening flags, verifies `/healthz`, verifies that the root page still says uploads are disabled, and inspects selected runtime properties.

No sample, malware, quarantine volume, engine socket, or external network service is required for this gate.

## CI qualification

GitHub Actions installs Podman and executes the same `make rootless-qualify` target under the normal runner account. This is a Linux rootless-runtime qualification, not yet a qualification of the purchased production VPS, its kernel, DNS, firewall or TLS setup.

## Production VPS gate

Before enabling uploads on the actual VPS, repeat the rootless qualification there and additionally verify:

- only SSH, HTTP and HTTPS are reachable from the Internet as intended;
- SSH password authentication is disabled;
- root SSH login is disabled;
- system packages are current;
- Caddy obtains and renews the real certificate;
- the Podman service runs under the dedicated unprivileged account;
- no engine socket is mounted or exposed;
- the application port is unreachable externally and reachable only through Caddy;
- memory pressure remains acceptable on the 1 GiB host.

## Exit criteria

M2 is code-qualified when `make check` and `make rootless-qualify` pass in CI. Production-VPS qualification remains a separate operator gate until a real VPS is available.

Uploads remain disabled after M2. The next milestone may introduce quarantine storage and the upload protocol, but it must not enable public hostile-file intake before the production-VPS gate is recorded.
