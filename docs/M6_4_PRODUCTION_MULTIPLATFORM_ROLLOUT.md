# M6.4 — Production multi-platform rollout

M6.4 prepares `amiguard.ploos.no` for the platform-aware intake and ASW handoff introduced in M6–M6.3. The deployment remains fail-closed and preserves legacy schema-v1 quarantine records in place.

## Upload staging boundary

Unclassified multipart sample bytes are written only to:

`<quarantine-root>/.incoming/.incoming-<submission-id>`

The `.incoming` directory is mode `0700`, must be a real directory, and symlinks are rejected. Multipart field order is intentionally irrelevant: `sample` may appear before `platform`. The service does not create a durable platform namespace until the complete request has been parsed and the platform has been validated.

After successful validation, the sample is atomically renamed on the same quarantine filesystem into one of:

- `<quarantine-root>/amiga/<id>.sample`
- `<quarantine-root>/atari-st/<id>.sample`
- `<quarantine-root>/mac68k/<id>.sample`

A rejected request must leave no sample in a platform namespace and no residual file in `.incoming`.

## Existing production data

Existing pre-M6 records use schema version 1 and live directly under the quarantine root as `<id>.sample` plus `<id>.json`. Do **not** bulk-move or rewrite those records during deployment.

`amiguard-admin` remains the compatibility boundary:

- root-level schema-v1 records are read as Amiga;
- schema-v2 records are read from platform namespaces;
- both forms are independently re-hashed before export/routing;
- duplicate IDs across legacy and namespaced records fail closed.

This means production migration is additive. New submissions use schema v2 while old records remain immutable until retention removes them through an explicitly qualified operation.

## Deployment sequence

1. Stop or disable public uploads (`AMIGUARD_UPLOAD_ENABLED=false`) while preserving the quarantine volume.
2. Take and verify a backup/snapshot of the quarantine root before replacing the service binary/image.
3. Verify there are no unexpected symlinks in the quarantine root or platform directories.
4. Deploy the M6.4 intake build with the same quarantine volume mounted on a single filesystem.
5. Start with uploads disabled and verify `/healthz`.
6. Verify `amiguard-admin list`, `show`, and `verify` against at least one legacy schema-v1 record if one exists.
7. Ensure ASW handoff directories already exist for `amiga`, `atari`, and `mac68k`; the public service must not create ASW namespaces.
8. Enable uploads.
9. Submit one harmless fixture for each platform through the public interface/API and verify schema-v2 metadata and namespace placement.
10. Route the harmless fixtures with `amiguard-admin route`, import them on ASW, verify acknowledgements/queues, then remove only the qualification fixtures according to the normal retention policy.

Do not perform a production migration by renaming all old root-level files into `amiga/`. The compatibility reader deliberately avoids requiring this risky bulk mutation.

## Rollback

If qualification fails:

1. disable uploads immediately;
2. leave quarantine bytes untouched;
3. roll back the service binary/image while keeping the same quarantine volume;
4. preserve `.incoming` for inspection only if a failed process left data there; do not promote such bytes manually into a platform namespace without re-verification;
5. re-run legacy record verification before re-enabling uploads.

A rollback must never delete or rewrite existing schema-v1 or schema-v2 samples merely to restore service availability.

## Production qualification gate

M6.4 repository qualification must prove:

- multipart field order does not change platform routing;
- sample-first Atari and Mac submissions never stage under Amiga;
- unsupported platform supplied after the sample is rejected with no durable platform data;
- `.incoming` is empty after accepted and rejected requests;
- a symlinked `.incoming` path is rejected;
- existing M6 platform tests and all prior repository checks remain green.

Repository CI is necessary but not sufficient to claim that `amiguard.ploos.no` itself has been deployed. The live service is production-qualified only after the deployment sequence above has been executed against the actual host and quarantine volume.
