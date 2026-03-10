# Snapshots Output Summary

## Goal

Enable `dtctl query ... -o snapshot` to enrich each record with a decoded `parsed_snapshot` built from:

- `snapshot.data` (base64 protobuf payload)
- `snapshot.string_map` (string cache index mapping)

The implementation target was parity with the existing reference behavior (Variant2 / namespace conversion), not heuristic reconstruction.

## What Was Implemented

### 1) Snapshot output mode

- `-o snapshot` now enriches records with a single user-facing field: `parsed_snapshot`.
- Intermediate fields from earlier prototypes were removed from final output.

### 2) Typed protobuf decode path (reference-aligned)

The parser was migrated from schema-less wire decoding to typed decoding using:

- `pkg/proto/rookout` (local copied protobuf definitions)
- `AugReportMessage`
- `Arguments2` (`Variant2` root)

Current flow:

1. Decode `snapshot.data` from base64.
2. Unmarshal into `AugReportMessage`.
3. Load string cache from `snapshot.string_map` (when supplied).
4. Build cache helpers (`strings`, `buffers`).
5. Convert `Variant2` recursively to dict-like output (reference style).

### 3) Variant conversion behavior

`variant2ToDict` now covers main and edge branches and emits reference-style keys such as:

- `@OT`
- `@CT`
- `@value`
- `@OS`
- `@attributes`
- `@max_depth`

Handled variant types include:

- `NONE`, `UNDEFINED`
- `INT`, `LONG`, `LARGE_INT`
- `DOUBLE`
- `STRING`, `MASKED`
- `BINARY`
- `TIME`
- `ENUM`
- `LIST`, `SET`, `MAP`
- `OBJECT`, `UKNOWN_OBJECT`, `NAMESPACE`
- `TRACEBACK`
- `ERROR`, `COMPLEX`, `LIVETAIL`
- `FORMATTED_MESSAGE`, `DYNAMIC`, `MAX_DEPTH`

### 4) Correctness decisions made during implementation

- Removed variable-name heuristics and hardcoded local/object shaping from old prototype code.
- Rejected recursive “base64-inside-string” parsing because it is not part of the reference behavior.
- Kept output contract stable around `parsed_snapshot`.

## Files Updated

- `pkg/output/snapshot.go`
  - Typed snapshot decode and Variant2 conversion.
  - Cache handling and recursive dict conversion.
  - Timestamp formatting and large integer safety behavior.

- `pkg/output/snapshot_test.go`
  - Switched to typed fixture payloads (`AugReportMessage` marshaling).
  - Added edge-case regression coverage (formatted message, large int, set/reverse order, error, timestamp).

- `pkg/proto/rookout/`
  - Added local copied generated protobuf files required for snapshot decoding.

- `go.mod`, `go.sum`
  - Removed external dependency on `dynatrace.com/protocols/v11` after switching to local protobuf package.

## Validation Performed

- Unit tests:
  - `go test ./pkg/output` passes.

- Formatting/build checks:
  - `gofmt` applied to snapshot output files.

- Runtime check example:
  - `./dtctl query "fetch application.snapshots | sort timestamp desc | limit 1" -o snapshot > snapshot.out4.json`
  - Output includes populated `parsed_snapshot` with nested decoded content.

## Current Status

✅ `-o snapshot` is fully wired and producing typed decoded snapshot output.

✅ Parser is aligned to reference-style Variant2 namespace conversion approach.

✅ Edge-case handling and tests are in place for the most relevant variant categories.

## Known Remaining Risk (Low)

Some very rare production-only Variant2 patterns may still require minor parity polish if observed in new live data samples. The core path and major edge branches are implemented and validated.

## PR Changelog (Ready to Paste)

### Summary

Adds snapshot output support for DQL query results via `-o snapshot`, decoding `snapshot.data` + `snapshot.string_map` into a structured `parsed_snapshot` field using typed protobuf conversion.

### Why

- Snapshot records include encoded payloads that are not readable in raw form.
- Users need decoded snapshot content directly in CLI output for analysis/troubleshooting.
- Typed conversion provides deterministic structure and consistent output semantics.

### What Changed

- Added typed snapshot decode flow in `pkg/output/snapshot.go`:
  - base64 decode `snapshot.data`
  - unmarshal to `rookout.AugReportMessage`
  - apply `snapshot.string_map` into string cache
  - convert `Variant2` recursively to dict output (`@OT`, `@CT`, `@value`, `@OS`, `@attributes`, `@max_depth`)
- Added/expanded variant coverage, including edge branches:
  - `LARGE_INT`, `SET`, `ERROR`, `COMPLEX`, `LIVETAIL`, `FORMATTED_MESSAGE`, time formatting alignment
- Updated tests in `pkg/output/snapshot_test.go`:
  - migrated to typed protobuf fixtures
  - added edge-case regression test coverage
- Added local protobuf package in `pkg/proto/rookout/` and switched imports from external module to local package.
- Removed external `dynatrace.com/protocols/v11` dependency from `go.mod` / `go.sum`.

### Validation

- `gofmt -w pkg/output/snapshot.go pkg/output/snapshot_test.go`
- `go test ./pkg/output`
- Runtime smoke check:
  - `./dtctl query "fetch application.snapshots | sort timestamp desc | limit 1" -o snapshot > snapshot.out4.json`

### Behavioral Notes

- Output contract remains `parsed_snapshot`.
- Recursive decoding of base64-looking strings was intentionally not introduced (not part of reference behavior).

### Risk / Follow-up

- Low risk for common paths; rare production-only Variant2 shapes may still need incremental parity tuning as encountered.