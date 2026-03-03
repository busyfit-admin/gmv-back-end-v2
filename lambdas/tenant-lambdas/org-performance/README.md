# Org Performance Lambdas

Organization performance APIs are split into focused lambdas under this folder to simplify isolation and troubleshooting.

## Lambdas

- `manage-performance-cycles`
  - Performance cycles, quarters, meeting notes, cycle/quarter analytics
- `manage-performance-kpis`
  - KPI CRUD, sub-KPIs, KPI value entries
- `manage-performance-okrs`
  - OKR CRUD and key-result updates
- `manage-performance-goals`
  - Goals, value history, teams, sub-items, ladder-up, tasks

## Shared Handler

All lambdas use shared route/auth/service wiring from:

- `common/handler.go`

## Data Tables

- `ORGANIZATION_TABLE` (org metadata/admin checks)
- `ORG_PERFORMANCE_TABLE` (all performance entities)

## API Reference

- Detailed API documentation: `API_DOCUMENTATION.md`
