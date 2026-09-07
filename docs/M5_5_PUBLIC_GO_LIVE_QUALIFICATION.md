# M5.5 — Public go-live qualification

Status: PASS.

## Scope

This qualification records the controlled production opening of the AmiGuard sample-intake service at `amiguard.ploos.no` after the M5.4 landing page, edge policy and closed-production checks had passed.

No hostile sample was used. The qualification fixture was harmless test data and was deleted after verification.

## Qualified repository and image

- Repository HEAD used for the M5.4 deployment: `ef8a1cd5c3cabdf6d4dec2d84e2d7be0d9ea79e9`.
- Production image ID: `278cdf039f86df8cd28dbb1f628137a64b7becfa59dbdee030dd8a4c7ef0b383`.
- Container user: `65532:65532`.
- Read-only root filesystem: enabled.
- PID limit: 64.
- `no-new-privileges`: enabled.
- Effective capabilities: none.
- Application publish address: host loopback `127.0.0.1:8080` only.

## Quarantine storage

Production quarantine was verified before opening:

- mount: `/data/amiguard/quarantine`;
- filesystem: ext4;
- options include `nosuid,nodev,noexec`;
- directory owner: `amiguard:amiguard`;
- directory mode: `0700`;
- dashboard state before public qualification: `0 submissions`, `0 B`.

## Edge qualification

The live Caddy configuration was updated from the closed-page CSP to the production form policy:

```text
form-action 'self'
```

`caddy validate` returned `Valid configuration`, Caddy was reloaded successfully, and the live response returned the expected default-deny CSP together with the existing security headers.

Before opening intake:

- `GET /healthz` returned HTTP 200;
- the public landing page was closed;
- public `POST /api/v1/submissions` returned HTTP 404;
- no public sample-retrieval routes were present.

## Production enablement

The active production Quadlet was changed to:

```text
Environment=AMIGUARD_UPLOAD_ENABLED=true
```

The rootless user service was daemon-reloaded and restarted. The enabled landing page then exposed the production same-origin multipart form with the required `sample` and `consent=true` fields.

## Public harmless submission

A harmless 46-byte fixture was submitted through the public HTTPS endpoint.

Fixture SHA-256:

```text
ee31632588a4e56a6bb8a4b226399d7395f0221271b065e84c9a5424482d896e
```

Server receipt:

- submission ID: `a7e349cdeb35bd3ceb4cb9863c9a1f58`;
- SHA-256: `ee31632588a4e56a6bb8a4b226399d7395f0221271b065e84c9a5424482d896e`;
- size: 46 bytes;
- received at: `2026-09-07T23:37:31.432868346Z`.

The receipt SHA-256 matched the locally computed fixture SHA-256 exactly.

## Admin verification

`amiguard-admin show` returned valid schema-v1 quarantine metadata with:

- matching submission ID;
- matching SHA-256;
- matching size;
- `consent=true`;
- `executed=false`;
- `extracted=false`.

`amiguard-admin verify` returned:

```text
VERIFIED a7e349cdeb35bd3ceb4cb9863c9a1f58 sha256=ee31632588a4e56a6bb8a4b226399d7395f0221271b065e84c9a5424482d896e size=46
```

## No-retrieval verification

The following public paths returned HTTP 404 for the submitted ID:

```text
/api/v1/submissions/a7e349cdeb35bd3ceb4cb9863c9a1f58
/samples/a7e349cdeb35bd3ceb4cb9863c9a1f58
/download/a7e349cdeb35bd3ceb4cb9863c9a1f58
```

This confirms that enabling intake did not create a public retrieval surface.

## Cleanup and final state

The harmless qualification sample and its metadata were deleted from quarantine and the temporary local fixture was removed.

Final checks:

- `amiguard-admin dashboard`: `0 submissions`, `0 B`;
- `GET /healthz`: HTTP 200;
- enabled landing page still presents the production form;
- active Quadlet remains `AMIGUARD_UPLOAD_ENABLED=true`.

## Result

M5.5 public go-live qualification: **PASS**.

The AmiGuard public sample-intake service is live at `amiguard.ploos.no` with write-only HTTPS intake, local/SSH-only administration, dedicated quarantine storage, no public retrieval endpoint and the M5 abuse controls in force.
