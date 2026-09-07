# AmiGuard Infrastructure

Shared infrastructure contracts and deployment foundations for the AmiGuard ecosystem.

AmiGuard Infrastructure is intentionally separate from scanner/appliance code. It defines stable shared boundaries for storage, component identities, deployment assumptions, health contracts, and qualification gates.

## M0 status

M0 establishes the repository and infrastructure contract. It does not deploy production services yet.

Implemented in M0:

- MIT licensing
- documented architecture and ownership boundaries
- machine-readable component manifest
- deterministic manifest validation
- Makefile entry points
- CI validation on pushes and pull requests

## Validation

Requires Python 3.11+.

```sh
make check
```

## Scope

This repository owns shared infrastructure definitions and deployment contracts. It does not own malware samples, AAA scanner implementation, historical antivirus engines, signature research conclusions, or the end-user appliance UI. Samples must never be committed here.

## Next milestone

M1 should turn the M0 contracts into the first deployable service slice while retaining the rule that secrets and malware payloads never enter Git history.

## License

MIT. See `LICENSE`.
