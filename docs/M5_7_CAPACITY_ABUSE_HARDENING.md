# M5.7 Capacity and abuse hardening

M5.7 protects the public sample intake against quarantine exhaustion and distributed upload-volume abuse without adding CAPTCHA or third-party browser dependencies.

## Quarantine capacity guard

Before a submission body is accepted, the service checks the filesystem that backs the configured quarantine root. Intake fails closed with HTTP 503 and `Retry-After: 3600` when either configured storage threshold is reached or the filesystem capacity check cannot be completed.

Production contract:

- `AMIGUARD_QUARANTINE_MIN_FREE_BYTES=536870912` (512 MiB)
- `AMIGUARD_QUARANTINE_MAX_USED_PERCENT=85`
- reserve space for one maximum configured upload in addition to the minimum-free floor
- no sample bytes are committed when the capacity guard rejects the request

## Global upload byte budget

Per-client request limiting alone can be bypassed by source-IP rotation. M5.7 therefore adds one process-wide upload-byte budget shared by all clients.

Production contract:

- `AMIGUARD_GLOBAL_UPLOAD_BYTES=268435456` (256 MiB)
- `AMIGUARD_GLOBAL_UPLOAD_WINDOW_SECONDS=3600` (1 hour)
- request charge includes declared HTTP body size, including multipart overhead
- unknown/chunked request sizes are conservatively charged at the maximum request allowance
- rejected or malformed requests are not refunded from the abuse budget
- budget exhaustion returns HTTP 429 with `Retry-After`

Configuration bounds are intentionally constrained:

- global byte budget must be at least one maximum request allowance and at most 8 GiB
- byte-budget window must be between 60 seconds and 86400 seconds
- invalid global-budget configuration fails closed rather than silently weakening protection

## Existing controls retained

M5.7 does not replace existing M5 controls. Public intake still retains:

- 6 upload attempts per minute per client
- maximum 2 concurrent uploads globally
- 16 MiB default sample-size limit
- strict multipart fields and explicit consent
- trusted `X-Forwarded-For` only from loopback Caddy
- write-only public service with no sample retrieval endpoint
- finite HTTP server timeouts

## Qualification state

The implementation and static deployment contract are complete when CI passes on the exact M5.7 HEAD. Production qualification remains a separate gate and must verify the deployed image, active environment values, normal harmless upload success, and fail-closed behavior without deleting or modifying unrelated quarantine submissions.
