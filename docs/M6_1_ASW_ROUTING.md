# M6.1 — Platform-aware ASW routing

M6.1 adds an explicit, local-only export boundary between the public AmiGuard quarantine and ASW inboxes. Public intake still never executes samples and never pushes them automatically into an analysis workstation.

## Command

`amiguard-admin route <submission-id> <existing-asw-inbox-root>`

The command:

1. locates the submission in the platform-specific quarantine namespace;
2. validates the metadata contract;
3. re-hashes the quarantined sample and verifies its exact size;
4. maps the public platform identifier to the ASW namespace;
5. requires that the destination root and destination namespace already exist and are non-symlink directories;
6. copies the sample using exclusive-create semantics;
7. verifies the copied bytes again;
8. writes an exclusive-create routing manifest adjacent to the copied sample.

No shell execution, emulator launch, extraction, network transfer, or automatic ASW analysis is performed.

## Platform mapping

- `amiga` -> `amiga` -> AmiSandbox
- `atari-st` -> `atari` -> AtariSandbox
- `mac68k` -> `mac68k` -> MacSandbox

Schema-v1 quarantine records from before M6 are treated as Amiga for backward compatibility. Schema-v2 records must carry a platform value matching the namespace in which they are stored.

## ASW inbox contract

For submission `<id>` and ASW namespace `<namespace>`, routing creates:

- `<asw-inbox-root>/<namespace>/<id>.sample`
- `<asw-inbox-root>/<namespace>/<id>.route.json`

The manifest schema is `schema_version: 1`, `kind: amiguard-asw-routing-manifest` and binds:

- submission ID
- public platform ID
- ASW namespace
- SHA-256
- exact sample size
- original receipt timestamp
- route timestamp
- source identity `amiguard-public-quarantine`

A pre-existing sample or routing manifest causes routing to fail closed. This makes repeated routing explicit instead of silently overwriting an earlier handoff.

## Trust boundary

This is still an export/handoff mechanism, not a trust shortcut. ASW must independently validate the routing manifest and sample hash before importing the sample into its immutable originals store. A routed sample must not be considered analysed, infected, verified, approved, signed, or publishable merely because routing succeeded.

## Qualification gate

Repository tests must prove all three platform mappings, byte/hash binding, rejection of tampered quarantine samples, refusal when the target namespace is missing, refusal of duplicate routing, and backward-compatible schema-v1 Amiga routing.
