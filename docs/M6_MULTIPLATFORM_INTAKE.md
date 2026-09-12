# M6 — Multi-platform malware intake

AmiGuard's public intake endpoint remains the single submission gateway for retro-malware research while routing samples into platform-specific quarantine namespaces.

## Supported platforms

- `amiga` — existing AmiGuard path and backward-compatible default
- `atari-st` — Atari ST/STE family, routed to the AtariSandbox/ASW Atari backend
- `mac68k` — classic 68k Macintosh family, routed to the MacSandbox/ASW mac68k backend

The public service MUST NOT infer a platform from file contents or filenames. A browser/API submitter selects a platform explicitly. To preserve compatibility with existing clients, an omitted platform is treated as `amiga`.

## Storage and trust boundary

Each accepted sample is written beneath its platform namespace:

`<quarantine-root>/<platform>/<submission-id>.sample`

with adjacent metadata:

`<quarantine-root>/<platform>/<submission-id>.json`

The original client filename is never used as a storage path or retained in metadata. Samples are not executed, extracted, or made retrievable by the public intake service.

Metadata schema version 2 adds `platform` and keeps the immutable sample SHA-256, size, receipt time, consent, and execution/extraction denial state.

## Routing contract

Downstream export/import tooling selects a platform namespace explicitly:

- `amiga` -> ASW namespace `amiga` -> AmiSandbox
- `atari-st` -> ASW namespace `atari` -> AtariSandbox
- `mac68k` -> ASW namespace `mac68k` -> MacSandbox

Public quarantine and analysis remain separated. No sample is automatically pulled from the public service into an analysis workstation, and no analysis result or signature is automatically published.

## Qualification gate

Repository qualification must prove:

1. omitted `platform` remains compatible with the existing Amiga submission path;
2. all supported explicit platforms are accepted;
3. unknown platform values fail closed before durable storage;
4. platform namespaces cannot escape the quarantine root;
5. metadata and receipts bind the selected platform;
6. original filenames remain absent from durable storage and metadata.
