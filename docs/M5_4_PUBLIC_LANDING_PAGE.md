# M5.4 — Public landing page and submission consent

Status: implemented in repository; production deployment and final public qualification remain pending.

## Purpose

M5.4 replaces qualification-only landing-page wording with the production sample-submission contract that will be visible when uploads are enabled.

## Enabled page contract

When `AMIGUARD_UPLOAD_ENABLED=true`, `GET /` presents a minimal same-origin HTML form that POSTs `multipart/form-data` to `/api/v1/submissions`.

The page states:

- samples are accepted for defensive Amiga malware research;
- the submitter must be authorized to provide the material;
- unrelated personal data, credentials and private communications must not be submitted;
- the maximum sample size is 16 MiB;
- the client filename is not retained;
- quarantine metadata consists of the server-generated submission ID, SHA-256, size, receipt time and consent state;
- no public retrieval endpoint exists;
- unclassified submissions are normally retained for up to 90 days, while research evidence may be retained longer when required;
- intake itself does not execute, extract or publicly serve submitted samples.

The form requires both one `sample` file and explicit `consent=true`.

## Disabled page contract

When `AMIGUARD_UPLOAD_ENABLED=false`, `GET /` clearly states that the channel is closed and exposes no upload form. POST `/api/v1/submissions` continues to return 404.

## Edge CSP

The stock-Caddy example permits only same-origin form submission with:

```text
form-action 'self'
```

The previous `form-action 'none'` policy would block the production form and is therefore retired. No script, image, frame or third-party content is required by the landing page.

## Qualification requirements

Before public intake is enabled on `amiguard.ploos.no`:

1. deploy an image containing this M5.4 page while uploads remain disabled;
2. align the live Caddy CSP with the repository contract and validate the Caddy configuration;
3. verify the disabled public page has no form and public POST remains 404;
4. perform a loopback-only enabled qualification against the new image;
5. verify the enabled page contains the form, terms and consent wording;
6. submit one harmless fixture, verify it with `amiguard-admin`, and remove it;
7. review live Caddy/journald logging and confirm sample bytes and client filenames are not logged;
8. only after all remaining go-live gates pass, enable public uploads and perform one final harmless HTTPS submission.

No hostile sample is required for this qualification.
