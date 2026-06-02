# cost of phases:

## /speckit-specify - $1.57 (1.57) - 3 min 9 sec

/speckit-specify I need to create feture that will collect the workitems in Jira from specified sprint (same like get-sprint-issues), however I need different output. I need to calculate issues (count and sum of story points) with particular label assigned. In short format we need to see only: label name | issues count | sum of storypoints. In full format I need to see all issues in this group (line by line) like: label name | issues count | sum of story points | issue key | Summary | issue SP. The list of labels to aggregate should be configured. The order of labels is important. Provide the report in that configured order. I also need to list all the rest of issues not labeled with any of those confgured labels. This shoudl be a table below in format: Issye key | Summary | SP. Please keep highlevel processing flow by jira components like in get-sprint-issues module. The list of labels for start: ai-assisted, ai-assisted-ba, ai-assisted-dev, ai-assisted-qa.

|       Item        |                          Value                           |
|-------------------|----------------------------------------------------------|
| Feature Directory | specs/001-sprint-label-report                            |
| Spec File         | specs/001-sprint-label-report/spec.md                    |
| Branch            | 001-sprint-label-report                                  |
| Checklist         | specs/001-sprint-label-report/checklists/requirements.md |


## /speckit-plan - $0.52 (2.1) - 3 min

|     Item      |                     Value                      |
|---------------|------------------------------------------------|
| Branch        | 001-sprint-label-report                        |
| Plan          | specs/001-sprint-label-report/plan.md          |
| Research      | specs/001-sprint-label-report/research.md      |
| Data Model    | specs/001-sprint-label-report/data-model.md    |
| CLI Contract  | specs/001-sprint-label-report/contracts/cli.md |
| Quickstart    | specs/001-sprint-label-report/quickstart.md    |
| Agent Context | CLAUDE.md updated to reference plan            |

## /speckit-task - $0.52 (2.62) - 2 min 6 sec

Tasks file: specs/001-sprint-label-report/tasks.md

|             Metric              |  Value  |
|---------------------------------|---------|
| Total tasks                     | 21      |
| Phase 1 (Setup)                 | 2 tasks |
| Phase 2 (Foundational)          | 6 tasks |
| Phase 3 (US1 — Short Format)    | 5 tasks |
| Phase 4 (US2 — Full Format)     | 3 tasks |
| Phase 5 (US3 — Unlabeled Table) | 2 tasks |
| Phase 6 (Polish)                | 3 tasks |

## /speckit-implement - $3.45 (6.07) - 5 min 32 sec

All 20 automated tasks completed (T001–T020). T021 requires live Jira validation.

Files created/modified

|                File                 |                                               Change                                               |
|-------------------------------------|----------------------------------------------------------------------------------------------------|
| cmd/get-sprint-label-report/main.go | New — entry point with full wiring, all 4 renderers, flag validation                               |
| internal/labelreport/aggregator.go  | New — LabelGroup, ComponentReport, Aggregate()                                                     |
| internal/jiraservice/jira.go        | Modified — Labels []string + Components []string added to Issue; populated in LoadIssuesFromSprint |
| internal/config/config.go           | Modified — ReportLabels []string field; parseReportLabels() with default list                      |
| Makefile                            | Modified — build-sprint-label-report, run-sprint-label-report targets; updated build and help      |
