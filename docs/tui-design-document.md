# **Ledger Helper TUI Design Document**

This document outlines the user interface flow, screen layouts, and interaction model for the Go-based ledger-helper application. It defines a comprehensive autocomplete model, including support for hierarchical accounts.

## **1\. User Actions**

* Start the Application
* Initiate a New Transaction
* Pick Transaction Date
* Enter Transaction Payee
* Allocate Debits (positive legs)
* Allocate Credits (balancing legs)
* Receive and Select a Template for Debit/Credit Allocation
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
     * If found, it asks the user if they want to restore the previous session. If yes, the batch is loaded into memory.:w
     * If not found, or if the user declines, it starts a new session.:w
   * **Action:** The application parses journal.dat in the background to build its in-memory model.:
   * **Transition:** Moves to the Batch Review Screen.:
2. **State: Batch Review Screen**
   * Displays a list of transactions in the current batch, sorted by date.:
   * **User Input:**:
     * Press n (new transaction) \-\> Transition to Transaction Entry Screen.:w
     * Use ↑/↓ arrow keys to select a transaction.:w
     * Press e (edit selected) \-\> Transition to Transaction Entry Screen, populated with the selected transaction's data.:w
     * Press w (write) \-\> Transition to Confirm Write.:w
     * Press q (quit) \-\> Transition to Confirm Quit.:w
3. **State: Transaction Entry Screen**
   * The screen shows a compact header (Date, Payee) followed by two allocation sections: **Debits** and **Credits**.:
   * **Flow Step 1: Date Entry**:
     * Defaults to the date of the previously entered transaction.:w
     * ←/→ keys select date component (year, month, day).:w
     * ↑/↓ keys increment/decrement the selected component. Typing a number also works.:w
   * **Flow Step 2: Payee Entry**:
     * User enters Payee (with autocomplete).:w
     * After the payee is set the app searches for templates tied to that payee.:w
     * If templates exist a modal opens to select one; otherwise focus moves to the first debit line.:w
   * **Flow Step 3: Debit Allocation**:
     * Debit lines capture positive postings (e.g., expenses, asset increases).:w
     * Tab/shift+tab move between account and amount inputs; ctrl+n adds a new debit line; ctrl+d deletes the focused line (at least one debit always remains).:w
     * Amount inputs accept inline calculator expressions.:w
   * **Flow Step 4: Credit Allocation**:
     * Credit lines capture balancing legs (negative postings).:w
     * The UI continually displays **Remaining = Debits – Credits**.:w
     * With focus on a credit amount, pressing `b` fills the line with the outstanding balance.:w
     * ctrl+n/ctrl+d behave the same as in the debit section.:w
     * **Autocomplete Logic (Applies to all fields like Payee and Account):**:w
       * **Single Suggestion:** When typing provides a single, unambiguous match, the remainder of the suggestion is shown in a dimmed style. The bottom help bar shows [tab]accept. Pressing Tab fills in the suggestion.:w

       * **Multiple Suggestions:** When typing could match multiple entries, a dropdown list of options appears below the input field. The user can navigate this list with ↑/↓. The help bar shows [↑/↓]navigate [tab/enter]accept. Pressing Tab or Enter accepts the highlighted option.:w

       * **Hierarchical Account Completion:** For account names with colons (e.g., Expenses:Food:Groceries), completion happens one segment at a time.:w

         * Typing Exp + Tab completes to Expenses.:w

         * Typing :Fo + Tab then completes to Expenses:Food.:w

         * The single and multiple suggestion rules apply to each segment. If typing F could match Food or Fuel, a dropdown would appear for that segment.:w

   * **Flow Step 5: Confirmation**:
     * Press `ctrl+c` (confirm transaction).:w
     * **Action:** Debits and credits are validated for balance. On success, the transaction is added to the batch and **the session is persisted to .ledger-helper-batch.tmp**.:w
     * **Transition:** to Batch Review Screen.:w
4. **State: Template Selection Screen**
   * After the payee is captured, a modal enumerates saved templates, showing debit accounts on the left and credit accounts on the right. Selecting one seeds both sections; `esc` skips.:
5. **End States: Confirm Write & Confirm Quit**
   * Completing a "Write" action will delete the .ledger-helper-batch.tmp file, while quitting reminds the user about any unwritten batch items.

## **3\. Keybindings Summary**

* Navigation: `tab` / `shift+tab` move between fields and sections.
* Editing: `ctrl+n` add line in focused section, `ctrl+d` delete focused line (minimum one per section).
* Balancing: `b` fills a credit amount with the remaining balance.
* Confirmation & exit: `ctrl+c` confirm transaction, `esc` return to batch, `ctrl+q` quit.
* Templates: `enter` apply highlighted template, `esc` skip.
