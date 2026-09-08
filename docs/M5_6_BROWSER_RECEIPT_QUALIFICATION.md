# M5.6 Browser Receipt Qualification

## Purpose

M5.6 improves the successful browser submission flow without changing the public write-only intake model. Browser form submissions that advertise HTML support receive an HTML receipt page, while API clients requesting JSON continue to receive the existing JSON receipt.

## Repository qualification

The implementation and tests were qualified by GitHub Actions CI run #54 on exact commit:

`43f74e04bc1c800cda9d365d0bbde35da8405fb8`

Result: `completed / success`.

The automated tests cover both content-negotiated paths:

- browser-style `Accept: text/html,application/xhtml+xml` -> HTTP 201 HTML receipt;
- API `Accept: application/json` -> HTTP 201 JSON receipt.

## Production deployment

Production host: `amiguard.ploos.no`

The M5.6 image was built from the qualified repository state and loaded into the dedicated `amiguard` rootless Podman store.

Active image ID observed during deployment:

`cf98d909ebb21afebe33d7866a2e8512b8cf85a5a4bf85338efa8d9782672bce`

The active Quadlet retained:

`Environment=AMIGUARD_UPLOAD_ENABLED=true`

Post-restart health check:

`https://amiguard.ploos.no/healthz` -> HTTP 200.

## Public HTTPS browser-flow qualification

A harmless 41-byte qualification fixture was submitted through the public HTTPS endpoint using browser-style HTML content negotiation.

Fixture SHA-256:

`9a5227ea0c1d4a56f7c1e69d310208eb6303e345b41310e80b8acbdfafb35127`

Server-generated submission ID:

`9e018dedcbb619640537ad6c05a12f44`

Observed HTML receipt contained:

- `Submission received`;
- submission ID;
- SHA-256;
- `Submit another sample` link.

The stored metadata reported:

- schema version: 1;
- kind: `amiguard-quarantine-submission`;
- size: 41 bytes;
- consent: true;
- executed: false;
- extracted: false;
- received at: `2026-09-08T00:07:09.693318572Z`.

`amiguard-admin verify` returned `VERIFIED` with the exact expected SHA-256 and size. A direct SHA-256 of the quarantined bytes also matched the original fixture hash exactly.

## Cleanup

Only the qualification submission `9e018dedcbb619640537ad6c05a12f44` was removed after verification. Both its `.sample` and `.json` files were confirmed absent after cleanup.

Other quarantine submissions were deliberately left untouched. At the end of qualification, the dashboard reported two unrelated submissions totaling 220 bytes.

## Result

**M5.6 production browser receipt qualification: PASS.**

The user-visible success path now produces a proper receipt page for browsers while preserving JSON compatibility for API clients. No public retrieval, admin, parsing, extraction, execution, or scanning endpoint was introduced.
