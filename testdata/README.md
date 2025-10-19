# Test Data

## Structure

- `fixtures/` - Static test fixtures used by unit tests
  - `sample.ledger` - Parser test fixture with various transaction formats
- `sessions/` - Development test session files

## Manual Testing Workflow

To use the sample session file for manual TUI testing, either copy the session file to your working directory, or set the working directory to the sessions directory.

The session file will be loaded when Teller starts, allowing you to test the UI with pre-populated mock transaction data.
