# Teller

A terminal-based data entry tool for plain text accounting (ledger-cli format).

## Overview

Teller is a TUI application that provides intelligent autocomplete, transaction templates, and inline calculation to streamline the process of entering transactions into ledger-cli formatted files. It parses your existing ledger file to learn account names, payees, and common transaction patterns, then uses this information to reduce manual typing and prevent errors.

All data remains local. Teller reads from and appends to your ledger file without modifying existing entries.

## Features

- **Hierarchical account autocomplete** using a Trie structure for segment-by-segment completion
- **Payee autocomplete** from transaction history
- **Transaction templates** inferred from common debit/credit patterns per payee, ranked by frequency
- **Inline calculator** in amount fields (e.g., `19.99 * 2 + 5.50`)
- **Auto-balance** to fill remaining amounts with a single keystroke
- **Real-time balance tracking** showing debit/credit totals and remaining balance
- **Session persistence** to `.teller-session.tmp` for crash recovery
- **Batch workflow** for entering multiple transactions before committing to the ledger

### Not Yet Implemented

Features planned in original documentation but not currently available:
- CSV statement import
- Two-stage transaction capture (quick draft → later finalization)

## Installation

Requires Go 1.25.1+

```bash
git clone https://git.sr.ht/~jakintosh/teller
cd teller
make build
```

Binary outputs to `bin/teller`.

## Quick Start

```bash
teller my-finances.ledger
```

Press `n` to create a transaction. Fill in date, payee, and posting details. Press `ctrl+s` to save to batch. Press `w` from batch view to write all transactions to your ledger file.

## How It Works

### Startup Process

1. **Parse** - Reads ledger file and extracts transactions
2. **Learn** - Builds in-memory intelligence database:
   - Trie for hierarchical account names
   - Sorted payee list
   - Transaction templates (debit/credit account patterns grouped by payee and ranked by frequency)
3. **Run** - Launches TUI

Everything rebuilds from the ledger file on each startup. No persistent database.

### Transaction Templates

Teller analyzes transaction history to identify patterns. For each payee, it tracks which accounts appear together and how often. When you enter "Gas Station", it can suggest:

```
Debit:  Expenses:Auto:Gas
Credit: Assets:Credit Card
(frequency: 12)
```

Select a template to pre-fill account fields.

### Hierarchical Autocomplete

Account names are colon-separated (e.g., `Expenses:Food:Groceries`). The Trie structure enables segment-by-segment completion:

- Type `Exp` → suggests `Expenses`
- Type `Expenses:Fo` → suggests `Expenses:Food`, `Expenses:Fuel`
- Type `Expenses:Food:G` → suggests `Expenses:Food:Groceries`

## Usage

### Interface Views

**Batch Review** (home screen)
- Lists current work-in-progress transactions
- `n` - new transaction, `e` - edit selected, `w` - write to ledger, `q` - quit

**Transaction Entry**
- Header: Date, Cleared status, Payee, Comment
- Template selector (if templates available for payee)
- Debit section: positive postings (expenses, asset increases)
- Credit section: negative postings (typically asset decreases)
- Shows running totals and remaining balance

Key bindings:
- `Tab` / `Shift+Tab` - navigate fields
- `ctrl+a` / `ctrl+d` - add/delete posting lines
- `b` - auto-balance (fills empty amount to make transaction sum to zero)
- `ctrl+s` - save transaction to batch
- `Esc` - cancel

**Template Selection**
- Opens when pressing Enter on template button
- `↑`/`↓` to navigate, `Enter` to apply, `Esc` to cancel

### Calculator

Amount fields accept expressions:
- `45.50 + 12.25`
- `19.99 * 3`
- `(100 - 15) * 1.08`

Uses decimal arithmetic to avoid floating-point errors.

## Project Structure

```
cmd/teller/          Main entry point
core/                Transaction and Posting types, ledger formatting
parser/              Ledger file parser
intelligence/        Trie, template inference, payee/account storage
tui/                 Bubble Tea UI implementation
session/             Session persistence to .teller-session.tmp
util/                Expression evaluator
```

### Core Packages

**parser** - Parses ledger-cli format files. Supports `YYYY-MM-DD` and `YYYY/MM/DD` dates, cleared markers (`*`), transaction and posting comments, various amount formats (with or without `$`), and elided amounts (one posting per transaction can omit the amount).

**intelligence** - Builds the in-memory database. `NewIntelligenceDB` iterates through parsed transactions to populate the Trie (for accounts), extract unique payees, and analyze transaction structures. Templates are created by grouping postings into debit (amount ≥ 0) and credit (amount < 0) sets, then tracking frequency per payee.

**tui** - Implements the UI using Bubble Tea. The `Model` struct holds all application state. View rendering and input handling are separated by screen type (batch, transaction, template, confirm). Focus management enables tab navigation between form fields. Suggestions are refreshed on each input change.

**util** - `EvaluateExpression` wraps govaluate to parse mathematical expressions and returns results with 2 decimal places using shopspring/decimal for precision.

## Ledger Format Support

**Supported:**
- Transactions with date, optional cleared marker (`*`), payee
- Postings with account, optional amount, optional comment
- Transaction-level comments
- Amount formats: `123.45`, `$123.45`, various sign positions
- One elided amount per transaction (automatically inferred)

**Not supported:**
- Automated transactions, periodic transactions
- Tags and metadata
- Virtual postings, lot pricing, commodity prices

## Development

Build: `make build`

Test: `go test ./...`

---

Teller is designed for ledger-cli format files. It may not be fully compatible with other plain text accounting systems (hledger, beancount).
