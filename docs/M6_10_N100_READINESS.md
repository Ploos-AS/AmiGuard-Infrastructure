# M6.10 — Dedicated N100 ASW readiness

## Status

**REPOSITORY PREPARATION ONLY — LIVE QUALIFICATION DEFERRED**

M6.10 is the eventual live qualification of the operator-controlled production VPS → dedicated N100 ASW path. The physical N100 workstation is not required for this repository-preparation gate, and M6.10 live qualification must not be declared PASS until the real machine is available and the live procedure succeeds.

## Intended N100 boundary

The dedicated workstation receives already-prepared M6.9 transfer bundles. It must not trust transport integrity alone. The received bundle is independently checked with `asw_transfer_bundle.py verify` before any content is exposed to the ASW importer.

The ASW root must provide separate platform namespaces:

```text
<asw-root>/amiga/
<asw-root>/atari/
<asw-root>/mac68k/
```

The route importer remains `asw_route_import.py`. A successful intake acknowledgement must retain `analysis_started=false`. Receipt/import is not authorization to execute a sample.

## Host readiness checklist

Before the live qualification, the operator must establish and record:

- dedicated ASW Unix account and private ASW storage root;
- OS/version and current security updates;
- filesystem free-space check;
- time synchronization;
- authenticated operator-controlled transfer path;
- no public inbound service exposing the ASW intake root;
- separate `amiga`, `atari`, and `mac68k` namespaces;
- local copies/checkouts of the qualified AmiGuard Infrastructure and ASW tooling;
- ability to run Python 3 tooling;
- explicit backup/recovery policy for evidence and immutable originals;
- analysis remains a separate operator action after intake.

Any missing prerequisite must fail closed: do not import the bundle and do not start analysis.

## Live qualification procedure — deferred until N100 is available

1. Select harmless qualification submissions on the production VPS.
2. Route them to a local handoff root with the qualified admin tooling.
3. Prepare an M6.9 transfer bundle.
4. Verify the bundle on the VPS before transport.
5. Transfer the complete bundle through the operator-approved authenticated channel.
6. On the N100, run `asw_transfer_bundle.py verify` against the received copy.
7. Compare the received transfer-manifest digest with the sender-side recorded digest.
8. Import each verified `.sample` + `.route.json` pair with `asw_route_import.py`.
9. Confirm platform mapping: public `amiga` → ASW `amiga`, `atari-st` → `atari`, `mac68k` → `mac68k`.
10. Confirm acknowledgements and queue records report `analysis_started=false`.
11. Confirm immutable originals are read-only and content hashes match the transfer evidence.
12. Confirm no analysis process was started by receipt, verification or import.

If any digest, size, namespace, schema, provenance or state assertion differs, the qualification must fail closed.

## PASS boundary

The repository-preparation gate may PASS in CI without N100 hardware. That only proves this readiness/runbook contract remains present.

The **live M6.10 qualification must not be declared PASS** until the physical dedicated N100 executes the procedure above successfully with harmless samples. No CI simulation, localhost transfer or another generic runner substitutes for that live hardware gate.
