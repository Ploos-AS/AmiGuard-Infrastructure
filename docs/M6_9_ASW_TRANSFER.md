# M6.9 — Verified VPS → dedicated ASW transfer

## Scope

M6.9 defines and qualifies the transport boundary between the production AmiGuard VPS and the dedicated ASW workstation.

The transfer is deliberately **operator-driven**. The repository does not contain an automatic network pull, automatic SSH command, automatic ASW import, automatic analysis start, or automatic signature publication path.

The production-side flow is:

1. Select one or more submissions explicitly with `amiguard-admin route` into a local handoff root.
2. Prepare a verified transport bundle with `scripts/asw_transfer_bundle.py prepare`.
3. Transfer that bundle using an operator-approved transport such as `rsync`/SSH, removable media, or another authenticated channel.
4. On the ASW workstation, run `scripts/asw_transfer_bundle.py verify` against the received copy before exposing it to the ASW importer.
5. Import the verified `.sample` + `.route.json` pairs with the ASW `tools/asw_route_import.py` importer.
6. Confirm the ASW acknowledgement reports `analysis_started: false` before any separate analysis action.

## Transfer bundle contract

A bundle contains:

```text
transfer-manifest.json
amiga/<submission-id>.sample
amiga/<submission-id>.route.json
atari/<submission-id>.sample
atari/<submission-id>.route.json
mac68k/<submission-id>.sample
mac68k/<submission-id>.route.json
```

Only namespaces represented by selected submissions need to exist.

`transfer-manifest.json` uses schema `amiguard.asw.transfer-bundle/1` and records:

- source role `amiguard-production-vps`;
- destination role `dedicated-asw-workstation`;
- creation time;
- `network_transfer_performed: false` at bundle creation;
- `analysis_started: false`;
- submission ID, public platform and ASW namespace for every entry;
- SHA-256 and size of each sample;
- SHA-256 and size of each route manifest.

The bundle tool does not contain networking code. `network_transfer_performed: false` means the tool itself has not performed a transfer; the operator chooses and executes the transport separately.

## Integrity and fail-closed rules

Preparation and verification must fail closed when any of the following is observed:

- route/sample is a symlink or not a regular file;
- route JSON exceeds the bounded manifest size;
- route schema/kind/field set does not match the routing contract;
- submission ID is malformed;
- unsupported public platform;
- public platform → ASW namespace mismatch;
- sample or route filename does not match the submission ID;
- sample SHA-256 or size differs from the route manifest;
- routing source is not `amiguard-public-quarantine`;
- transfer entry is duplicated;
- transfer path escapes the expected namespace;
- transferred sample or route-manifest digest/size differs from the transfer manifest;
- any bundle or entry claims `analysis_started: true`.

Preparation refuses an existing destination bundle root rather than overwriting it.

Samples and copied route manifests are written mode `0400` in the prepared bundle. The bundle root and namespace directories are private (`0700`).

## Example — production VPS

First create a local handoff with the already-qualified admin command:

```sh
export AMIGUARD_ADMIN_QUARANTINE_ROOT=/data/amiguard/quarantine
mkdir -p /tmp/amiguard-transfer-handoff/{amiga,atari,mac68k}
amiguard-admin route <submission-id> /tmp/amiguard-transfer-handoff
```

Then prepare a bundle:

```sh
python3 scripts/asw_transfer_bundle.py prepare \
  --handoff-root /tmp/amiguard-transfer-handoff \
  --bundle-root /tmp/amiguard-asw-transfer-001
```

Expected result:

```text
PREPARED <n> verified transfer entries -> /tmp/amiguard-asw-transfer-001
```

The operator may then transfer the complete directory using an approved authenticated transport. Do not transfer only the sample while omitting the route and transfer manifests.

## Example — dedicated ASW workstation

Before ASW import:

```sh
python3 scripts/asw_transfer_bundle.py verify \
  --bundle-root /path/to/received/amiguard-asw-transfer-001
```

Expected result:

```text
VERIFIED <n> transfer entries; analysis_started=false
```

Only after this verification should the operator pass individual `.sample` and `.route.json` pairs to the ASW route importer.

## Security boundary

M6.9 does **not** create a trusted path merely because SSH or another encrypted transport is used. The receiving ASW verifies the content identity independently after transport.

The public VPS does not receive credentials that allow it to start analysis on ASW. The transfer bundle does not contain commands, executable hooks, or analysis requests. Route manifests remain data only.

The ASW importer remains responsible for its independent validation and for creating immutable ASW originals, queue records and acknowledgements with `analysis_started: false`.

## Repository qualification

M6.9 repository qualification requires:

- `scripts/asw_transfer_bundle.py` is syntax-valid;
- unit tests prepare and verify a three-platform bundle;
- copied samples and route manifests are read-only;
- sample tampering fails verification;
- symlinked handoff content fails closed;
- `make check` executes the M6.9 tests;
- ordinary CI and closed rootless Podman qualification remain green.

Repository PASS qualifies the transfer contract and tooling. A later live N100 qualification must still execute a real operator-controlled transport to the dedicated workstation and verify the received bundle there.
