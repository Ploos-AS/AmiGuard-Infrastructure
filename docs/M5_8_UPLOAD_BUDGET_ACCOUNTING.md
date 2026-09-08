# M5.8 Upload-aware abuse accounting

M5.8 closes two correctness gaps in the M5.7 global upload-byte budget without changing the public write-only boundary or the production quota values.

## Goals

The shared upload-byte budget must track the configured maximum upload size rather than a hard-coded 16 MiB assumption, and requests rejected only because the global concurrency limit is already full must not consume the byte budget.

## Configured maximum request charge

The abuse guard now derives its maximum conservative request charge from the active sample-size limit:

```text
max request charge = AMIGUARD_MAX_UPLOAD_BYTES + 128 KiB multipart allowance
```

With the current production value `AMIGUARD_MAX_UPLOAD_BYTES=16777216`, the effective maximum request charge remains 16 MiB + 128 KiB. If an operator intentionally changes the sample-size limit, unknown/chunked requests and oversized declared request lengths are charged against the byte budget using the corresponding configured maximum instead of the old fixed 16 MiB value.

`AMIGUARD_GLOBAL_UPLOAD_BYTES` must be at least one derived maximum request charge. Invalid combinations fail closed rather than silently under-accounting requests.

## Concurrency ordering

The concurrency slot is now acquired before rate/budget accounting. If all upload workers are already occupied, the request is rejected immediately with HTTP 429 and `Retry-After: 1` and does not consume the global byte budget.

Once a request has acquired an upload slot, the existing abuse semantics remain conservative: requests that reach an upload worker and are then rejected or malformed are not refunded from the shared byte budget.

When a later per-client or global-budget check rejects a request, the acquired concurrency slot is released before returning.

## Existing production contract unchanged

M5.8 does not change the deployed M5.7 values:

- `AMIGUARD_MAX_UPLOAD_BYTES=16777216`
- `AMIGUARD_GLOBAL_UPLOAD_BYTES=268435456`
- `AMIGUARD_GLOBAL_UPLOAD_WINDOW_SECONDS=3600`
- maximum 2 concurrent uploads globally
- 6 upload attempts per minute per client
- 512 MiB quarantine free-space floor
- 85% quarantine used-space watermark

No CAPTCHA, browser dependency, public retrieval endpoint or web administration is introduced.

## Qualification

Tests cover:

- a non-default upload limit producing a matching maximum abuse-budget charge and minimum valid global budget;
- invalid global budget below the derived request allowance failing closed;
- unknown-length requests using the derived conservative charge;
- concurrency-limit rejection leaving the global byte counter unchanged;
- existing per-client, global-budget and trusted-client-IP behavior.

CI #72 passed on exact code HEAD `3a19594e282ee8dfa47bb907e60c48eec00f89a9` before this documentation update.

M5.8 requires a final exact-HEAD CI pass after documentation is committed. Production deployment is a separate gate because the running M5.7 image does not contain these accounting-order and configurable-limit corrections yet.
