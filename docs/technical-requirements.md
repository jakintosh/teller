# **Ledger Helper TUI: Technical Requirements**

This document outlines the technical requirements for the Ledger Helper TUI application based on the project brief and design documents.

## **1\. General Architecture & Environment**

1.1. Language & Framework: The application SHALL be written in Go. The TUI SHALL be implemented using the bubbletea framework.
1.2. Deployment: The final product MUST be compilable to a single, statically-linked binary with no external runtime dependencies.
1.3. Data Source: The sole source of historical data SHALL be a user-provided, ledger-cli compatible plain text file. The application MUST NOT create or manage its own persistent database.
1.4. Output: The application's only output SHALL be valid ledger-cli formatted text, appended to the user's ledger file.

## **2\. Data Parsing and In-Memory Model**

2.1. Ledger File Parser:
2.1.1. The application MUST implement a parser capable of reading a ledger file on startup.
2.1.2. The parser's primary function is to extract transaction details: date, payee/description, and a list of posting accounts with their associated amounts.
2.1.3. The parser MAY ignore non-essential ledger syntax for the MVP, including comments, tags, and complex directives, to ensure performance.
2.1.4. The parsing process SHOULD run as a background task to avoid blocking the UI on startup.
2.2. In-Memory Data Store ("Intelligence DB"):
2.2.1. The application MUST build an in-memory model from the parsed ledger data to power its suggestion features.
2.2.2. Accounts: All unique account names MUST be stored in a Trie data structure to facilitate efficient, segment-by-segment hierarchical searching (e.g., Expenses:Food:Groceries).
2.2.3. Payees: All unique payee names MUST be stored in a list or map for fast searching and retrieval.
2.2.4. Transaction Templates:
2.2.4.1. The system MUST analyze transactions to identify common structures associated with specific payees.
2.2.4.2. A "template" is defined as the ordered combination of debit accounts (amount ≥ 0) and credit accounts (amount < 0) that appear together in a transaction, independent of amounts.
2.2.4.3. The system SHALL calculate the frequency of each template for a given payee.
2.2.4.4. The system MUST be able to retrieve a list of templates for a payee, ordered from most to least frequent.
2.2.4.5. When a transaction omits an amount for one posting, the system MUST infer the missing value from the other legs and classify the posting on the correct debit or credit side before recording the template.

## **3\. Core Features & Business Logic**

3.1. Inline Calculator:
3.1.1. Input fields for monetary amounts (e.g., debit and credit line amounts) MUST accept basic mathematical expressions as strings.
3.1.2. The calculator MUST support addition, subtraction, multiplication, and division.
3.1.3. The calculator MUST respect order of operations, including the use of parentheses for grouping.
3.1.4. The underlying calculation MUST use a data type that avoids floating-point precision errors for currency (e.g., a decimal library or scaled integers).
3.2. Transaction Generation:
3.2.1. The application MUST be able to convert its internal transaction representation into a valid, formatted ledger-cli text entry.
3.2.2. The output formatting MUST adhere to standard ledger practices, including indented postings and aligned currency amounts.

## **4\. User Interface (TUI) & State Management**

4.1. State Model:
4.1.1. The application state MUST be managed within a single, unified Model struct, consistent with The Elm Architecture as implemented by bubbletea.
4.1.2. The model MUST track the current view (e.g., Batch Review, Transaction Entry), the list of transactions in the current batch, the state of the active form, and all data required for rendering.
4.2. Batch Entry Workflow:
4.2.1. The UI MUST present a "Batch Review" screen displaying all transactions created in the current session.
4.2.2. Users MUST be able to add new transactions to the batch and select existing transactions from the batch for editing.
4.3. Autocomplete Functionality:
4.3.1. The TUI MUST provide autocomplete suggestions in all payee and account input fields.
4.3.2. Single Suggestion: If only one match is found, the remainder of the text MUST be displayed in a dimmed, "ghost text" style. The suggestion MUST be accepted by pressing the Tab key.
4.3.3. Multiple Suggestions: If multiple matches are found, a dropdown list of options MUST be displayed below the input field. The user MUST be able to navigate this list with arrow keys and select an option with Tab or Enter.
4.3.4. Hierarchical Autocomplete: For account fields, autocomplete MUST operate on one colon-separated segment at a time.
4.4. Session Persistence & Recovery:
4.4.1. The current batch of uncommitted transactions MUST be automatically saved to a temporary file (e.g., .ledger-helper-batch.tmp) upon any modification (add, edit, or delete).
4.4.2. On startup, the application MUST check for the existence of this temporary file.
4.4.3. If the file is found, the application MUST prompt the user to restore the previous session. If the user accepts, the batch is loaded from the file.
4.4.4. The temporary file MUST be deleted upon successful writing of the batch to the main ledger file or when the user quits without saving.

4.5. Command Hint Layout:
4.5.1. The contextual action hints displayed at the bottom of each screen SHALL be rendered as a vertically stacked list to improve readability.

4.6. Template Picker Presentation:
4.6.1. The template selection modal SHALL present debit and credit accounts in vertically stacked lists under clear headings.
4.6.2. The template selection modal SHALL automatically scroll the visible window as the highlighted template moves beyond the current viewport.
