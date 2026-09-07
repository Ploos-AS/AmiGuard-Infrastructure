# M5 Edge and Abuse Hardening

M5 adds application-level anti-abuse controls before public sample intake is enabled.

## Current controls

The upload endpoint now enforces two independent protections whenever uploads are enabled:

- per-client fixed-window rate limiting: 6 upload attempts per minute
- global upload concurrency limiting: at most 2 simultaneous uploads

Rejected requests return HTTP 429 and a `Retry-After` header.

The existing request controls remain in force:

- upload body is bounded by `AMIGUARD_MAX_UPLOAD_BYTES` plus a small multipart allowance
- default sample limit is 16 MiB
- server read-header, read, write and idle timeouts are finite
- maximum request headers are limited to 16 KiB
- exactly one `sample` part is accepted
- explicit `consent=true` is required
- unknown multipart fields are rejected
- no retrieval endpoint exists

## Client identity behind Caddy

The application may use `X-Forwarded-For` only when the immediate TCP peer is loopback. This matches the production design where Caddy is the only peer of the application on `127.0.0.1:8080`.

If a client can connect directly to the application from a non-loopback peer, forwarded headers are ignored for rate-limit identity. The production firewall and Quadlet must continue to keep the application port loopback-only.

## Why the controls live in the application

The production VPS currently uses the Ubuntu Noble Caddy package. M5 does not depend on optional third-party Caddy rate-limit modules. Keeping the abuse guard in the Go service makes the contract portable across the qualified Caddy package and preserves the minimal host footprint.

## Public intake state

Public uploads remain disabled. The production Quadlet must continue to set:

`AMIGUARD_UPLOAD_ENABLED=false`

M5 does not authorize changing that value.

## Qualification required before public enable

Before public intake is enabled, the production VPS must be rebuilt with the M5 image and qualified while keeping the public endpoint closed. Qualification must verify:

- image built from the exact M5 commit
- closed production service remains healthy
- public upload endpoint remains HTTP 404
- a separate loopback-only qualification instance can demonstrate HTTP 429 rate limiting
- concurrency limiting is covered by automated tests
- Caddy still proxies only to `127.0.0.1:8080`
- no new public listeners are introduced

Further milestones still need egress minimization, log/privacy review, retention/deletion, manual export verification, backup policy, incident recovery and final public-intake qualification.
