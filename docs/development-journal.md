# Development Journal

## 2025-10-03
- Overhauled transaction entry into debit/credit sections with balancing shortcuts.
- Updated template engine to persist debit and credit account sets.
- Expanded Bubble Tea tests for ctrl+c confirm, balancing, and template seeding.
- Revised documentation to reflect double-entry workflow, new keys, and templates.

## 2025-10-03
- Added a selectable "templates available" control beneath the payee field so users can open the template picker on demand and revisit it later in the entry flow.
- Refreshed template suggestion logic in the TUI to keep counts in sync and updated navigation/tests to cover the new focus target.
- Improved intelligence template extraction by inferring the sign of elided postings, ensuring balanced transactions still contribute debit and credit accounts.
- Updated design and technical documentation to describe the on-demand template picker and missing-amount inference.
