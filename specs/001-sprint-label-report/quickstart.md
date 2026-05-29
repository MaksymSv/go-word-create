# Quickstart: Sprint Label Aggregation Report

**Feature**: Sprint Label Aggregation Report
**Binary**: `bin/get-sprint-label-report`
**Date**: 2026-05-29

## Prerequisites

- Go 1.24.0+ installed
- Jira Cloud account with an API token
- `.env` file configured (copy from `.env.example` and fill in your values)
- `TEAMS` env var set to your Jira component names

## 1. Build

```bash
make build-sprint-label-report
# or directly:
go build -o bin/get-sprint-label-report ./cmd/get-sprint-label-report
```

## 2. Configure Labels (optional)

The default label list is `ai-assisted,ai-assisted-ba,ai-assisted-dev,ai-assisted-qa`.
To override, add to your `.env`:

```env
REPORT_LABELS=ai-assisted,ai-assisted-ba,ai-assisted-dev,ai-assisted-qa
```

Order matters — labels appear in the report in the order listed here.

## 3. Generate a Short-Format Report

```bash
# generates sprint-label-report.docx (default output file)
./bin/get-sprint-label-report -sprint="Sprint 42"

# with a custom output file
./bin/get-sprint-label-report -sprint="Sprint 42" -output="sprint-42-labels.docx"
```

## 4. Generate a Full-Format Report (per-issue breakdown)

```bash
./bin/get-sprint-label-report -sprint="Sprint 42" -format=full -output="sprint-42-full.docx"
```

## 5. Preview in Console (no Word document)

```bash
./bin/get-sprint-label-report -sprint="Sprint 42" -debug
./bin/get-sprint-label-report -sprint="Sprint 42" -format=full -debug
```

## 6. Via Make (after adding Makefile targets)

```bash
make run-sprint-label-report SPRINT="Sprint 42"
make run-sprint-label-report SPRINT="Sprint 42" FORMAT=full
make run-sprint-label-report SPRINT="Sprint 42" LOGONLY=1
```

## Validation Checklist

- [ ] `make build-sprint-label-report` completes without errors
- [ ] `./bin/get-sprint-label-report -sprint="<real sprint>"` produces a `.docx` file
- [ ] Short-format document contains one label table per component with correct row count
- [ ] Full-format document shows per-issue rows with repeated label/count/SP columns
- [ ] Unlabeled issues table is present and contains only issues with no configured label
- [ ] `-debug` flag prints to stdout and does not write a file
- [ ] Changing `REPORT_LABELS` in `.env` changes the label order and set in the output
- [ ] Missing required env var exits with a clear error message on stderr
- [ ] `make test` passes with no regressions in existing tests
