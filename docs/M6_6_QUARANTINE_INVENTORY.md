# M6.6 — Read-only quarantine inventory

M6.6 adds a pre-deployment inventory and integrity verification pass for the quarantine already present on `amiguard.ploos.no`.

The inventory is intentionally read-only. It performs metadata reads, `lstat`, directory enumeration, sample reads, SHA-256 hashing, and size verification only. It must not create, rename, rewrite, truncate, chmod, or remove quarantine files.

## Command

Run on the production host with public uploads disabled:

```sh
python3 scripts/inventory_quarantine.py /data/amiguard/quarantine
```

For a machine-readable report:

```sh
python3 scripts/inventory_quarantine.py /data/amiguard/quarantine --json > quarantine-inventory.json
```

Write the JSON report outside the quarantine volume.

## What is verified

The scanner understands both supported layouts:

- legacy schema-v1 records directly below the quarantine root; these are treated as Amiga;
- schema-v2 records below `amiga/`, `atari-st/`, and `mac68k/`.

For every candidate record it verifies:

- 32-character lowercase hexadecimal submission ID;
- expected metadata schema and kind;
- submission ID binding;
- platform binding for schema v2;
- consent is true;
- executed and extracted remain false;
- metadata SHA-256 and size are structurally valid;
- metadata and sample are regular non-symlink files;
- actual sample SHA-256 equals metadata SHA-256;
- actual sample size equals metadata size.

It also detects:

- orphan `.sample` or `.json` files;
- malformed quarantine filenames;
- duplicate submission IDs across legacy and platform namespaces;
- invalid/symlinked platform namespaces;
- a symlinked `.incoming` directory;
- residual files left in `.incoming`.

A clean inventory exits 0 and prints `PASS`. Any integrity or layout problem exits 1. Failure to inspect the requested quarantine root exits 2.

## Production use

Before deploying the multi-platform intake:

1. disable uploads;
2. take and verify a backup/snapshot;
3. run the M6.6 inventory against the live quarantine root;
4. preserve the generated report outside quarantine;
5. do not alter failed records merely to make the inventory pass;
6. investigate discrepancies individually and retain original bytes;
7. continue with M6.5/M6.4 deployment only after the inventory result is understood.

M6.6 does not migrate schema-v1 data. Existing uploads remain in place and continue to be readable through the compatibility path in `amiguard-admin`.

## Repository qualification

`make check` runs synthetic M6.6 tests covering:

- valid mixed schema-v1/schema-v2 inventory;
- orphan sample detection;
- post-ingest sample tampering;
- duplicate IDs across legacy and namespaced records;
- residual `.incoming` data;
- symlinked sample rejection.

Repository CI proves the scanner behavior but does not prove the current state of the production quarantine. Production inventory PASS requires the command to be run against the actual host volume.
