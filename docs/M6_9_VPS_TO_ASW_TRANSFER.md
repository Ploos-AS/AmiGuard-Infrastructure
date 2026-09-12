# M6.9 — Verified VPS to ASW transfer

## Scope

M6.9 defines the operational boundary for transferring a manually selected, already verified AmiGuard quarantine handoff from the production VPS to the dedicated ASW workstation.

The transfer MUST preserve the M6.8 trust boundary:

- no automatic export of arbitrary public quarantine content;
- no automatic pull from ASW into the public quarantine;
- no automatic analysis after transfer;
- no automatic signature generation or publication;
- explicit operator selection of each submission;
- end-to-end SHA-256 and size verification;
- fail-closed behavior on missing, duplicate, tampered, malformed, or mismatched transfer artifacts.

## Transfer bundle

A transfer bundle contains exactly two files for one submission:

```text
<submission-id>.sample
<submission-id>.route.json
```

The route manifest is the existing M6.1/M7.10 handoff contract and MUST identify:

- `schema_version: 1`
- `kind: amiguard-asw-routing-manifest`
- submission ID
- public platform
- ASW namespace
- SHA-256
- size
- public quarantine provenance

The transfer mechanism MUST NOT rewrite either file.

## Transport

M6.9 does not require one specific transport implementation. A production deployment MAY use an operator-initiated mechanism such as `scp`, `sftp`, `rsync` over SSH, removable media, or another authenticated transport.

The security contract is transport-independent:

1. Build the handoff bundle on the VPS with `amiguard-admin route` into a staging directory that is separate from the quarantine root.
2. Record the route manifest and sample SHA-256 before transport.
3. Transfer only the selected `.sample` and `.route.json` pair.
4. Place received files in a dedicated ASW inbound staging area, not directly into the ASW originals or queue trees.
5. Run an explicit receive validator before the ASW importer.
6. Reject the bundle if filename, platform/namespace mapping, size, or SHA-256 does not match the route manifest.
7. Only after validation, invoke the existing ASW route importer.
8. Confirm the ASW acknowledgement reports `result: ACCEPTED`, `route_manifest_verified: true`, `sample_verified: true`, and `analysis_started: false`.
9. Retain the route manifest and acknowledgement as provenance evidence.

## Required fail-closed cases

The receive gate MUST reject:

- symlinked sample, route manifest, or staging directories;
- missing sample or manifest;
- extra or unexpected files when validating a single-submission bundle;
- malformed JSON;
- unknown platform;
- platform/namespace mismatch;
- invalid submission ID;
- filename/submission-ID mismatch;
- invalid SHA-256 value;
- negative or mismatching size;
- sample SHA-256 mismatch;
- sample size mismatch;
- unexpected route `source`;
- duplicate receipt when the target receipt record already exists.

A failed receive MUST NOT create an ASW queue record and MUST NOT start analysis.

## Operational ownership

The production VPS is responsible for selecting and exporting a submission. The ASW workstation is responsible for receiving, re-verifying, importing, and acknowledging it.

The ASW workstation MUST NOT require direct read access to `/data/amiguard/quarantine` on the public VPS.

## Qualification gate

M6.9 repository qualification is PASS only when:

- a deterministic receive validator exists;
- the validator verifies route-manifest contract, filename binding, platform namespace, SHA-256, size, and provenance;
- positive fixtures pass for Amiga, Atari ST, and Mac 68k;
- tamper, namespace mismatch, missing-file, symlink, and duplicate-receipt cases fail closed;
- the repository CI runs the M6.9 validator/tests;
- documentation continues to state that transfer is operator initiated and analysis remains disabled until a separate explicit action.

A repository PASS proves the transfer contract and fixtures, not that the dedicated N100 workstation has already been physically installed or connected.
