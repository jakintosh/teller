# **Ledger Helper TUI Design Document**

This document outlines the user interface flow, screen layouts, and interaction model for the Go-based ledger-helper application. It defines a comprehensive autocomplete model, including support for hierarchical accounts.

## **1\. User Actions**

* Start the Application  
* Initiate a New Transaction  
* Pick Transaction Date  
* Enter Transaction Payee  
* Enter Transaction Total and Primary Account(s) (e.g., total charge on a credit card)  
* Receive and Select a Template for Allocating the Total  
* Allocate Total to Legs (Splits)  
* Edit a Transaction Leg  
* Use the Inline Calculator (with parenthesis support)  
* Confirm a Transaction  
* Navigate and Edit Transactions in the Batch  
* Review the Batch Summary  
* Commit the Batch to the Ledger File  
* Quit the Application

## **2\. User Interface Flow Chart**

The application flow incorporates a "total-first" data entry approach, session persistence for data safety, and a robust autocomplete system.

1. **Start State: Application Launch**  
   * The user runs ./ledger-helper journal.dat.  
   * **Action:** The application checks for a .ledger-helper-batch.tmp file.  
     * If found, it asks the user if they want to restore the previous session. If yes, the batch is loaded into memory.  
     * If not found, or if the user declines, it starts a new session.  
   * **Action:** The application parses journal.dat in the background to build its in-memory model.  
   * **Transition:** Moves to the Batch Review Screen.  
2. **State: Batch Review Screen**  
   * Displays a list of transactions in the current batch, sorted by date.  
   * **User Input:**  
     * Press n (new transaction) \-\> Transition to Transaction Entry Screen.  
     * Use ↑/↓ arrow keys to select a transaction.  
     * Press e (edit selected) \-\> Transition to Transaction Entry Screen, populated with the selected transaction's data.  
     * Press w (write) \-\> Transition to Confirm Write.  
     * Press q (quit) \-\> Transition to Confirm Quit.  
3. **State: Transaction Entry Screen**  
   * The screen is divided into a "Header" section (Date, Payee, Total) and a "Splits" section for allocation.  
   * **Flow Step 1: Date Entry**  
     * Defaults to the date of the previously entered transaction.  
     * ←/→ keys select date component (year, month, day).  
     * ↑/↓ keys increment/decrement the selected component. Typing a number also works.  
     * Press Enter to move to Payee.  
   * **Flow Step 2: Payee & Total Entry**  
     * User enters Payee (with autocomplete). Enter moves to Total Amount.  
     * User enters Total Amount. This can be a number or an inline calculation (e.g., (15.50 \* 2\) \+ 7.99). Enter moves to Primary Account.  
     * User enters Primary Account (e.g., Assets:Credit Card). This field defaults to the primary account of the previous transaction. Autocomplete is available.  
     * Press Enter to confirm the header.  
     * **Action:** App searches for templates based on the Payee.  
     * **Transition:** If templates found, show Template Selection. Otherwise, move focus to the Splits section for manual allocation.  
   * **Flow Step 3: Split / Allocation Entry**  
     * The screen **always displays the "Remaining (unallocated) Balance"**.  
     * The first split line defaults to the remaining balance.  
     * User can add new legs. A new leg will default to the same account category as the currently selected leg.  
     * When focus is on the amount field of a leg with no amount entered, and it's the *only* such leg, pressing b (balance) will automatically fill the field with the remaining unallocated balance.  
     * **Autocomplete Logic (Applies to all fields like Payee and Account):**  
       * **Single Suggestion:** When typing provides a single, unambiguous match, the remainder of the suggestion is shown in a dimmed style. The bottom help bar shows \[tab\]accept. Pressing Tab fills in the suggestion.  
       * **Multiple Suggestions:** When typing could match multiple entries, a dropdown list of options appears below the input field. The user can navigate this list with ↑/↓. The help bar shows \[↑/↓\]navigate \[tab/enter\]accept. Pressing Tab or Enter accepts the highlighted option.  
       * **Hierarchical Account Completion:** For account names with colons (e.g., Expenses:Food:Groceries), completion happens one segment at a time.  
         * Typing Exp \+ Tab completes to Expenses.  
         * Typing :Fo \+ Tab then completes to Expenses:Food.  
         * The single and multiple suggestion rules apply to each segment. If typing F could match Food or Fuel, a dropdown would appear for that segment.  
   * **Flow Step 4: Confirmation**  
     * Press c (confirm transaction).  
     * **Action:** The transaction is validated. On success, it is added to the batch and **the session is persisted to .ledger-helper-batch.tmp**.  
     * **Transition:** to Batch Review Screen.  
4. **State: Template Selection Screen**  
   * This modal view appears after entering a payee. It presents a list of suggested transaction templates for the user to quickly apply.  
5. **End States: Confirm Write & Confirm Quit**  
   * Completing a "Write" action will delete the .ledger-helper-batch.tmp file.
