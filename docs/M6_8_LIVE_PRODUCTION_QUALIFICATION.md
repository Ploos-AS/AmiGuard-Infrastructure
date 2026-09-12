# M6.8 — Live production multi-platform qualification

## Status

**PASS — 2026-09-12**

This record captures the live qualification of the M6 multi-platform intake deployment at `amiguard.ploos.no`. It is evidence of the operator-run production qualification; it is not a mechanism for automatically deploying or mutating production.

## Qualified deployment

- Service: `amiguard.ploos.no`
- Runtime: rootless Podman/Quadlet under the dedicated `amiguard` Unix user
- Submit image: `localhost/amiguard-submit:local`
- Qualified image ID: `bf665f712c9592d6a6f9a933e831c89320226507ecb82b5fae9abf3f84291943`
- Public reverse proxy: Caddy
- Container listener published only on `127.0.0.1:8080`
- Public platforms: `amiga`, `atari-st`, `mac68k`

The rollout was first started with uploads disabled. Local and public health/landing checks passed before the explicit operator action that enabled uploads. The running container was verified to use the qualified image ID.

## Pre-deployment quarantine evidence

Before the live rollout the read-only inventory reported:

```text
records=3 verified=3 failed=0 legacy_v1=3 bytes=288
platforms amiga=3 atari-st=0 mac68k=0
PASS
```

All three historical records were schema-v1 records and were retained in place. No automatic migration was performed.

A pre-M6 quarantine backup was also taken before cutover.

## Public HTTPS canaries

Three harmless text canaries were submitted through the public HTTPS endpoint after uploads were explicitly enabled.

| Platform | Submission ID | SHA-256 | Size |
| --- | --- | --- | ---: |
| Amiga | `c84a016ff8e64d5f293ea48bf97fa19e` | `6beaed938bfec164fcabd338d706556b0583ce809006fa28d93f36a5b13677f6` | 36 B |
| Atari ST | `39208be41ad40f71958e83072e6f0a12` | `763d6ab2e01283404d85f566570bba8bce1d9d271484908835426102bf513f9f` | 39 B |
| Mac 68k | `256a361b04f9fdf77c0f36092c33e913` | `1f9b2bb3e1d55bd85a6268114f5b7275f2769c0c2b128467f38ad3a270bbad3f` | 37 B |

Each public receipt returned the requested platform, matching SHA-256 and matching size. The landing page exposed all three platform values.

After the canaries, the read-only inventory reported:

```text
records=6 verified=6 failed=0 legacy_v1=3 bytes=400
platforms amiga=4 atari-st=1 mac68k=1
PASS
```

The `.incoming` spool was empty after successful requests.

## Admin verification

`amiguard-admin show` and `verify` succeeded for all three schema-v2 canaries. Their metadata had:

- `schema_version: 2`
- correct explicit platform
- `consent: true`
- `executed: false`
- `extracted: false`
- matching SHA-256 and size

Legacy compatibility was explicitly checked using schema-v1 submission `2d7e0f325459fa8cd12ec1273d61a342`. It verified successfully as platform `amiga`, SHA-256 `275a021bbfb6489e54d471899f7db9d1663fc695ec2fe2a2c4538aabf651fd0f`, size 68 B.

The final production inventory remained `6 verified / 0 failed / 3 legacy_v1`.

## ASW handoff qualification

The three harmless canaries were manually routed to a non-production handoff root. Namespace mapping was verified as:

```text
amiga    -> amiga
atari-st -> atari
mac68k   -> mac68k
```

Each handoff contained one `.sample` and one `.route.json` file.

The handoffs were then imported with the ASW `tools/asw_route_import.py` importer into a disposable non-production ASW root. All three acknowledgements returned:

- `result: ACCEPTED`
- `route_manifest_verified: true`
- `sample_verified: true`
- `analysis_started: false`

Queue records also retained `analysis_started: false`. Imported originals were stored by SHA-256 with filesystem mode `0400`.

No malware analysis was started as part of this production qualification.

## PASS gate

M6.8 live production qualification is PASS because:

- the qualified image was deployed under the intended rootless service account;
- closed-mode local and public health checks passed before enabling uploads;
- all three platform selectors were visible after enabling uploads;
- harmless public HTTPS canaries succeeded for all three platforms;
- namespaced quarantine routing and hashes/sizes were verified;
- `.incoming` had no residue after successful uploads;
- all three schema-v2 canaries passed admin verification;
- a pre-existing schema-v1 record passed compatibility verification;
- the final read-only quarantine inventory passed with zero failed records;
- manual ASW routing produced the intended namespace mappings;
- the ASW importer accepted all three handoffs without automatically starting analysis;
- ASW originals were stored read-only (`0400`).

**M6.8 result: PASS.**

## Operational boundary

This qualification does not authorize automatic transfer of arbitrary public quarantine content into ASW and does not authorize automatic analysis or signature publication. The public quarantine-to-ASW boundary remains an explicit verified operator action.

The next operational milestone is to define and qualify transfer from the production VPS to the dedicated ASW workstation while preserving this explicit trust boundary, provenance, integrity verification and fail-closed behavior.
