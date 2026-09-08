# M5.8 Production Qualification

M5.8 upload-aware abuse accounting was deployed and qualified on the dedicated production VPS at `amiguard.ploos.no` on 2026-09-08.

## Qualified source

- Repository HEAD deployed: `ffd1496447efe4fc9841952fe533fb9664df77c9`
- Exact-HEAD CI: #74, success
- Production image ID: `26df109b872f06c2283e1eb7250d6dd2feb30beadfb90f78ed648a4c4ebc4e79`
- Running container at deployment qualification: `e1c00f84941f1bae28062cf05d789c838566e2085e23e96e517d9bbdb52280e1`

## Active production contract

The running container was inspected after restart and retained the intended production values:

```text
AMIGUARD_GLOBAL_UPLOAD_BYTES=268435456
AMIGUARD_GLOBAL_UPLOAD_WINDOW_SECONDS=3600
AMIGUARD_MAX_UPLOAD_BYTES=16777216
AMIGUARD_QUARANTINE_MAX_USED_PERCENT=85
AMIGUARD_QUARANTINE_MIN_FREE_BYTES=536870912
AMIGUARD_QUARANTINE_ROOT=/data/quarantine
AMIGUARD_SUBMIT_ADDR=0.0.0.0:8080
AMIGUARD_UPLOAD_ENABLED=true
```

Both loopback and public HTTPS health checks returned the expected healthy service response.

## Public upload qualification

A harmless qualification fixture was submitted through the public HTTPS endpoint with JSON receipt negotiation.

```text
fixture: AmiGuard M5.8 production qualification fixture\n
size: 47 bytes
sha256: ab374cd5068d54b5a2882879aeffda5f4c67c45112d22fbfcadcfe101c9e497f
submission id: a20fa1cd8c57407440495f4a3c1726d8
received: 2026-09-08T09:48:11.599466987Z
HTTP status: 201
```

The server receipt returned the exact expected SHA-256 and size.

`amiguard-admin show` reported schema version 1, the expected submission ID/hash/size/time, consent true, executed false and extracted false. `amiguard-admin verify` returned VERIFIED for the same SHA-256 and 47-byte size. A direct SHA-256 of the stored sample also matched exactly.

## Cleanup and final state

Only the M5.8 qualification sample and its metadata were removed from quarantine. The local temporary qualification fixture was also removed. The cleanup check passed. No unrelated submissions were modified.

After cleanup, both loopback and public HTTPS health checks remained healthy.

## Result

**M5.8 production qualification: PASS.**

M5.8 is complete: implementation, tests, exact-HEAD CI, production deployment, public end-to-end qualification, stored-byte verification and qualification-fixture cleanup all passed.
