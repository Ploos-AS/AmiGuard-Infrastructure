# M0 — Infrastructure foundation

## Goal

Create a small, deterministic foundation for shared AmiGuard infrastructure before introducing deployable services.

## Architectural boundary

AmiGuard Infrastructure owns infrastructure contracts and deployment definitions shared by AmiGuard-facing services. It must not contain malware samples, quarantined payloads, API credentials, private keys, tokens, or conclusions that belong to scanner/signature research. The AAA scanner remains a separate component in the wider ecosystem.

## M0 deliverables

1. Repository identity and MIT license.
2. Explicit storage namespace rooted at `/data/amiguard`.
3. Machine-readable component inventory.
4. Component validation with no third-party runtime dependency.
5. CI gate invoking the same checks developers run locally.
6. Documentation of what M0 deliberately does not implement.

## Initial logical components

- `intake`: public-facing submission boundary; M0 defines identity, persistence and health contract only.
- `queue`: internal handoff boundary for later worker orchestration.
- `evidence`: internal durable metadata/provenance boundary.

## Security rules

- Never commit malware or suspicious samples.
- Never commit secrets.
- Persistent runtime data lives below `/data/amiguard`.
- Public exposure must be explicit in the manifest.
- Internal components must not become public implicitly.
- Future workers should treat submitted content as hostile and immutable.

## Qualification

M0 is code-qualified when `make check` passes from a clean checkout on Python 3.11+ and GitHub Actions passes the same validation.

## Non-goals

M0 does not yet provide containers, Compose/Quadlet, database schemas, object storage, TLS termination, authentication, malware analysis workers, or production deployment automation. Those belong to subsequent milestones.

## Exit criteria

- all M0 files are present on `main`;
- manifest validation passes;
- CI configuration exists;
- no temporary development branch is required;
- no malware or secret material is present.
