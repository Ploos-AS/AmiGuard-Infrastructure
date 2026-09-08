# M5.7 Production qualification

M5.7 capacity and distributed-abuse hardening was qualified on the dedicated production VPS at `amiguard.ploos.no` on 2026-09-08.

## Qualified source and image

- source HEAD: `2eab58252ef904f0cae1fff8deeed910abbc0456`
- production image tag built on host: `localhost/amiguard-submit:m5.7`
- image ID: `6390d823e3321c085d67f9bf037325a66bb5b9affd0331d535258d58d61ee7e2`
- service: `amiguard-submit.service`
- deployment: rootless Podman behind host Caddy

## Active production contract

The running container was inspected after deployment and after the fail-closed qualification restore. Active values were:

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

The service was active and both loopback and public HTTPS `/healthz` returned the expected healthy response.

## Harmless public upload qualification

Fixture contents:

```text
AmiGuard M5.7 production qualification fixture
```

Observed values:

- size: `47` bytes
- local SHA-256: `be2edb2a7232f0041ed106e8a71b65e8b58714d8c980e687ef91ef939186b524`
- submission ID: `ba3f781afbe9ff70f16beaee1f87d48a`
- received: `2026-09-08T01:37:15.593367065Z`

The public JSON receipt reported the same size and SHA-256. `amiguard-admin show` reported consent=true, executed=false and extracted=false. `amiguard-admin verify` passed, and a direct SHA-256 of the stored sample matched the original fixture exactly.

Only the qualification submission and its local fixture were removed afterward. Unrelated production submissions were not modified.

## Storage-pressure fail-closed qualification

The minimum-free threshold was temporarily raised to `8589934592` bytes, deliberately above the dedicated 4 GiB quarantine filesystem. The running container showed the temporary value before the request.

A harmless public HTTPS submission attempt then returned:

```text
HTTP/2 503
retry-after: 3600

submission storage temporarily unavailable
```

This proves the capacity guard rejects intake before committing a submission when the configured free-space floor cannot be satisfied. The test did not fill the filesystem and did not require modifying unrelated quarantine content.

The production unit was then restored to the normal contract. Final verification showed both the unit and running container using:

```text
AMIGUARD_QUARANTINE_MIN_FREE_BYTES=536870912
```

The service remained active and public `/healthz` remained healthy after restore.

## Result

**PASS.** M5.7 production deployment, normal public intake, exact-byte quarantine verification, and storage-pressure fail-closed behavior are qualified on the dedicated production VPS. The normal production configuration was restored after testing.
