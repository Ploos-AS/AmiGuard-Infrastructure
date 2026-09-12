# M6.5 — Production multi-platform deployment qualification

## Scope

M6.5 prepares the existing `amiguard.ploos.no` deployment for the M6 multi-platform intake contract without automatically mutating historical quarantine data. It qualifies the checked-in deployment contract and defines the operator sequence for a controlled production rollout.

Supported public intake platforms are:

- `amiga`
- `atari-st`
- `mac68k`

The public service remains behind Caddy. The submit container remains bound only to `127.0.0.1:8080`, runs rootless with the existing hardening controls, and keeps `/data/amiguard/quarantine` mounted at `/data/quarantine`.

## Legacy production data

Do not move, rewrite, or delete existing root-level schema-v1 submissions during this rollout.

The M6.1 admin tooling is backward-compatible with those records and treats schema-v1 records as Amiga submissions. New schema-v2 submissions are written below their platform namespaces. This allows production to transition without a destructive data migration.

No automatic migration of schema-v1 records is part of M6.5.

Before deployment, record a filesystem-level backup or snapshot of `/data/amiguard/quarantine` and record counts and hashes sufficient to prove that legacy root-level records were not changed by the rollout.

## Deployment sequence

1. Confirm the repository commit intended for production has green `make check` and rootless Podman qualification.
2. Back up `/data/amiguard/quarantine` before replacing the service image or unit.
3. Confirm existing root-level schema-v1 `.json` and `.sample` pairs are present and readable by `amiguard-admin list`, `show`, and `verify`.
4. Build or pull the qualified submit image from the approved commit.
5. Install/reload the Quadlet with `AMIGUARD_UPLOAD_ENABLED=false` and restart the service.
6. Verify `/healthz` through loopback and through the public HTTPS reverse proxy.
7. Verify the landing page renders with platform selection for Amiga, Atari ST and classic Macintosh 68k.
8. Confirm `/data/quarantine/.incoming` is either absent before first upload or is an ordinary non-symlink directory after initialization. It must never be a symlink.
9. Enable uploads only after the closed smoke checks have passed. Treat this as an explicit operator action; the repository default remains `AMIGUARD_UPLOAD_ENABLED=false`.
10. Submit harmless canary fixtures for `amiga`, `atari-st`, and `mac68k`. Verify each receipt platform, SHA-256 and size, then verify the durable files exist only below the matching platform namespace.
11. Verify the `.incoming` spool contains no residue after successful requests.
12. Re-run `amiguard-admin list`, `show`, and `verify` for at least one legacy schema-v1 record and one new schema-v2 record per platform.
13. Perform a manual `route` of harmless fixtures to a non-production ASW handoff root and verify the M6.3/M7.11 ASW-side importer accepts all three mappings without starting analysis automatically.
14. Re-count and re-hash the pre-existing root-level schema-v1 set. Any unexplained change is a deployment failure.

## Rollback

The rollback boundary is the submit service and reverse proxy configuration, not the quarantine corpus.

If health, landing page, namespaced storage, admin compatibility, or public proxy checks fail:

1. Set `AMIGUARD_UPLOAD_ENABLED=false` immediately.
2. Restore the previously qualified submit image/unit and reload the rootless user service.
3. Keep `/data/amiguard/quarantine` mounted unchanged; do not restore or rewrite it unless an integrity comparison proves actual data corruption.
4. Re-run legacy `amiguard-admin verify` checks.
5. Confirm Caddy still proxies only to `127.0.0.1:8080` and public TLS is healthy.
6. Record the failed commit, service logs, and qualification evidence before another attempt.

A rollback must not merge platform namespaces back into the root directory and must not delete `.incoming` blindly while a request is active. Disable uploads and stop the submit service before cleaning stale spool files.

## PASS gate

M6.5 repository qualification is PASS only when:

- `scripts/validate_m6_5_deploy_contract.py` passes;
- ordinary repository checks pass;
- the closed rootless Podman runtime qualification passes;
- the checked-in Quadlet remains upload-disabled by default;
- the persistent quarantine mount remains unchanged;
- no automatic schema-v1 migration is introduced.

A green repository M6.5 gate means **deployment-ready**, not that `amiguard.ploos.no` has already been changed. A live production PASS requires the operator sequence above against the actual VPS and public HTTPS endpoint.
