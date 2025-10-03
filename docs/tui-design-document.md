# **Ledger Helper TUI Design Document**

This document outlines the user interface flow, screen layouts, and interaction model for the Go-based ledger-helper application. It defines a comprehensive autocomplete model, including support for hierarchical accounts.

## **1\. User Actions**

* Start the Application
* Initiate a New Transaction
* Pick Transaction Date
* Enter Transaction Payee
* Receive and Select a Template for Debit/Credit Allocation
* Allocate Debits (positive legs)
* Allocate Credits (balancing legs)
* Edit a Transaction Leg
* Use the Inline Calculator (with parenthesis support)
* Confirm a Transaction
* Navigate and Edit Transactions in the Batch
* Review the Batch Summary
* Commit the Batch to the Ledger File
* Quit the Application

## **2\. User Interface Flow Chart**

The application flow follows a double-entry workflow with explicit debit and credit sections, session persistence for data safety, and a robust autocomplete system.

1. **Start State: Application Launch**
   * The user runs ./ledger-helper journal.dat.:
   * **Action:** The application checks for a .ledger-helper-batch.tmp file.:
     * If found, it asks the user if they want to restore the previous session. If yes, the batch is loaded into memory.
     * If not found, or if the user declines, it starts a new session.
   * **Action:** The application parses journal.dat in the background to build its in-memory model.:
   * **Transition:** Moves to the Batch Review Screen.:
2. **State: Batch Review Screen**
   * Displays a list of transactions in the current batch, sorted by date.:
   * **User Input:**:
     * Press n (new transaction) \-\> Transition to Transaction Entry Screen.
     * Use ↑/↓ arrow keys to select a transaction.
     * Press e (edit selected) \-\> Transition to Transaction Entry Screen, populated with the selected transaction's data.
     * Press w (write) \-\> Transition to Confirm Write.
     * Press q (quit) \-\> Transition to Confirm Quit.
3. **State: Transaction Entry Screen**
   * The screen shows a compact header (Date, Payee) followed by two allocation sections: **Debits** and **Credits**.:
   * **Flow Step 1: Date Entry**:
     * Defaults to the date of the previously entered transaction.
     * ←/→ keys select date component (year, month, day).
     * ↑/↓ keys increment/decrement the selected component. Typing a number also works.
   * **Flow Step 2: Payee Entry**:
     * User enters Payee (with autocomplete).
* After the payee is set the app searches for templates tied to that payee and surfaces the result in a focusable control labelled `“X templates available”`, where `X` is the template count.
* The user may press `tab` to move onto this control and press `enter` at any time to open the template picker; focus may also skip past it directly into the debit section when the user wants to continue without a template.
   * **Flow Step 3: Debit Allocation**:
     * Debit lines capture positive postings (e.g., expenses, asset increases).
     * Tab/shift+tab move between account and amount inputs; ctrl+n adds a new debit line; ctrl+d deletes the focused line (at least one debit always remains).
     * Amount inputs accept inline calculator expressions.
   * **Flow Step 4: Credit Allocation**:
     * Credit lines capture balancing legs (negative postings).
     * The UI continually displays **Remaining = Debits – Credits**.
     * With focus on a credit amount, pressing `b` fills the line with the outstanding balance.
     * ctrl+n/ctrl+d behave the same as in the debit section.
     * **Autocomplete Logic (Applies to all fields like Payee and Account):**
       * **Single Suggestion:** When typing provides a single, unambiguous match, the remainder of the suggestion is shown in a dimmed style. The bottom help bar shows [tab]accept. Pressing Tab fills in the suggestion.
       * **Multiple Suggestions:** When typing could match multiple entries, a dropdown list of options appears below the input field. The user can navigate this list with ↑/↓. The help bar shows [↑/↓]navigate [tab/enter]accept. Pressing Tab or Enter accepts the highlighted option.
       * **Hierarchical Account Completion:** For account names with colons (e.g., Expenses:Food:Groceries), completion happens one segment at a time.
         * Typing Exp + Tab completes to Expenses.
         * Typing :Fo + Tab then completes to Expenses:Food.
         * The single and multiple suggestion rules apply to each segment. If typing F could match Food or Fuel, a dropdown would appear for that segment.
   * **Flow Step 5: Confirmation**:
     * Press `ctrl+c` (confirm transaction).
     * **Action:** Debits and credits are validated for balance. On success, the transaction is added to the batch and **the session is persisted to .ledger-helper-batch.tmp**.
     * **Transition:** to Batch Review Screen.
4. **State: Template Selection Screen**
   * The template picker is opened explicitly from the "templates available" control under the payee input. The modal enumerates saved templates, showing debit accounts on the left and credit accounts on the right. Selecting one seeds both sections; `esc` skips, and the control remains available if the user wants to revisit the picker later.:
5. **End States: Confirm Write & Confirm Quit**
   * Completing a "Write" action will delete the .ledger-helper-batch.tmp file, while quitting reminds the user about any unwritten batch items.

## **3\. Keybindings Summary**

* Navigation: `tab` / `shift+tab` move between fields and sections.
* Editing: `ctrl+n` add line in focused section, `ctrl+d` delete focused line (minimum one per section).
* Balancing: `b` fills a credit amount with the remaining balance.
* Confirmation & exit: `ctrl+c` confirm transaction, `esc` return to batch, `ctrl+q` quit.
* Templates: `enter` apply highlighted template, `esc` skip.
