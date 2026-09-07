# M5.3 Public intake operations

This document defines the minimum operational policy for enabling public sample intake at `amiguard.ploos.no`.

Public upload must remain disabled until the final go-live qualification has passed.

## Scope

The submission service is a write-only intake channel for potentially hostile files. It does not execute, extract, parse, scan, or serve submitted bytes. Administration is local/SSH-only through `amiguard-admin`; there is no web-admin surface and no public retrieval endpoint.

## Data retained

For each accepted submission the service retains only:

- a server-generated 128-bit submission ID;
- the submitted file bytes under an opaque `.sample` name;
- SHA-256 and byte size;
- UTC receipt time;
- explicit consent state;
- `executed=false` and `extracted=false` state.

The client-provided filename is not retained in quarantine metadata and is never used as a storage path.

Infrastructure logs may contain ordinary network/request metadata such as source IP address, timestamps, request path, status code and user-agent. Logs must never contain sample bytes.

## Retention and deletion

Unreviewed submissions have a default operational retention target of 90 days. They may be deleted earlier when obviously irrelevant, duplicate, malformed, abusive, or no longer needed.

A sample selected for legitimate malware research may be retained longer only when needed for AmiGuard research and must remain in malware-designated storage. It must not be copied into ordinary document storage, Git repositories, CI artifacts, cloud-sync folders, or general backups.

Deletion is currently an explicit operator action. Before deleting a submission, the operator must identify both files by validated submission ID:

```text
<id>.sample
<id>.json
```

The two files are deleted together. Afterwards `amiguard-admin dashboard` or `list` is used to verify that the submission is no longer present.

## Backup policy

The production quarantine filesystem is excluded from ordinary host backups by default.

Any future quarantine backup must be explicitly designed and documented as hostile-content storage with access controls equivalent to the primary quarantine. A normal VPS backup destination must not silently become a malware archive.

Configuration, deployment files and documentation may be backed up normally; sample bytes may not.

## Verified export

Research export is always explicit and operator initiated.

1. Run `amiguard-admin verify <id>` against the production quarantine.
2. Export with `amiguard-admin export <id> <existing-directory>`.
3. Record the SHA-256 reported by the admin tool.
4. Transfer the exported file to the isolated research environment using an operator-controlled channel.
5. Recompute SHA-256 at the destination and require an exact match before analysis.
6. Treat the exported file as hostile content at every stage.

The intake service itself never exports or retrieves a submission.

## Log and privacy hygiene

Operational logging follows data minimisation:

- do not log request bodies or sample bytes;
- do not add client filenames to application logs or quarantine metadata;
- do not log multipart form contents;
- keep access-log retention no longer than operationally necessary;
- restrict journal and web-server log access to administrators;
- never expose logs through the public application.

Before public enablement, the production Caddy and systemd/journald configuration must be reviewed to confirm that no request-body logging is enabled.

## Incident response

The first response to suspected abuse, storage pressure, unexpected application behaviour, or a submission-handling incident is to close intake.

1. Set `AMIGUARD_UPLOAD_ENABLED=false` in the production Quadlet.
2. Run `systemctl --user daemon-reload` as the `amiguard` service account if the unit changed.
3. Restart `amiguard-submit.service`.
4. Verify `GET /healthz` returns 200.
5. Verify public `POST /api/v1/submissions` returns 404.
6. Preserve the quarantine filesystem unless deletion is specifically required by the incident.
7. Inspect service/Caddy logs and filesystem capacity before re-enabling intake.

Closing uploads must not require rebuilding the image.

## Go-live gate

Public intake may be enabled only after all of the following are true:

- the current submit image is deployed and qualified with uploads off;
- rootless runtime hardening is still active;
- the dedicated quarantine filesystem is mounted `nodev,nosuid,noexec`;
- `amiguard-admin` verify and export have passed against a harmless loopback-only qualification submission;
- the harmless qualification data and temporary qualification container have been removed;
- the public landing page contains production submission instructions and explicit consent wording rather than qualification text;
- Caddy/journald log hygiene has been reviewed;
- the retention, deletion, backup, export and incident procedures in this document are accepted for initial production;
- a final harmless public submission is successfully received, verified and removed after enablement;
- public retrieval remains impossible.

Only then may the production Quadlet set `AMIGUARD_UPLOAD_ENABLED=true`.