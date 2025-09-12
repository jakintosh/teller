# **Architect's Reasoning & Technical Breakdown**

Alright, let's break this down. My goal here is to map the user stories and design decisions to concrete technical components, keeping things as simple and robust as possible. The project brief is clear: this is a power-user tool, so we can make certain assumptions about the operating environment and prioritize efficiency over hand-holding. I'll split my thinking into the two main areas: the "engine" (business logic) and the "dashboard" (the interface).

## **1\. Business Logic: The Engine**

This is the heart of the application. It's responsible for reading the user's financial history, making sense of it, and using that understanding to simplify new entries.

### **A. Ledger File Parsing & Data Ingestion**

The absolute first thing we need to do is read and understand the user's existing ledger file. This file is our "database." The parsing has to be fast and memory-efficient, as these files can grow quite large over years of use.

* **The Challenge:** A full ledger parser that supports every feature of ledger-cli is a massive undertaking. We need to be surgical. What do we *actually* need for our MVP features?  
* **Lean Approach:** We don't need to understand every ledger directive or feature. For auto-completion and template suggestions, we only need to reliably extract a few key pieces of information from each transaction: the date, the payee (description), and the list of postings (account name and amount). We can ignore comments (mostly), tags, and complex directives for now. A line-by-line parser using regular expressions tailored to the user's common transaction format should be sufficient and much faster than building a full abstract syntax tree (AST). We'll treat each transaction as a discrete block of text.  
* **Initial Action:** On startup, we'll do a single, full read of the ledger file. This will happen in the background so the user can start entering data immediately if they are restoring a session.

### **B. The In-Memory "Intelligence" Model**

Once we've parsed the data, we need to store it in a way that makes querying for suggestions fast. We're not using a database, so this all has to live in memory.

* **Payee & Account Storage:** A simple list of all unique payee strings and a separate list of all unique account strings is the baseline. For accounts, we need something that understands the hierarchy. A Trie (prefix tree) is the perfect data structure for this. It will make hierarchical autocomplete (Expenses:Food:Groceries) incredibly efficient. We can populate the Trie by splitting account names by the colon (:) and inserting the segments.  
* **Transaction Template Discovery:** This is the clever bit. The goal is to find common transaction structures for a given payee.  
  * **The Challenge:** How do you define a "template"? It's not just the accounts used, but the *structure* of the postings. For example, a "Gas Mart" transaction might always have a debit to Expenses:Auto:Gas and a credit from Assets:Credit Card. A "Super Grocery Store" transaction might have multiple Expenses postings.  
  * **Lean Approach:** For each unique payee, we can store a list of associated "skeletons." A skeleton is just the list of posting accounts from a past transaction, with the amounts stripped out. For a given payee, "City Market", we might see skeletons like \[Expenses:Groceries, Assets:Checking\] and \[Expenses:Groceries, Expenses:Household, Assets:Credit Card\].  
  * We can then analyze these skeletons. A simple frequency count is the way to go. If we see the skeleton \[Expenses:Groceries, Assets:Checking\] 20 times for "City Market" and another skeleton only twice, the first one is our primary template suggestion. We can store this as a map: map\[payeeString\] \-\> map\[skeletonHash\] \-\> count. The hash allows us to quickly group identical skeletons.

### **C. Core Calculation & Transformation**

* **Inline Calculator:** The design doc calls for an inline calculator that supports parentheses. This is a classic computer science problem. We don't need to write a full math engine. We can find a well-vetted, lightweight Go library for evaluating mathematical expressions from a string. Pulling in a small, single-purpose dependency is much better than rolling our own and dealing with edge cases and security. The key is to ensure it can handle floating-point arithmetic precisely, as we're dealing with money.  
* **Transaction Formatting:** The final output needs to be a perfectly formatted ledger entry. This is essentially a string templating problem. We'll have an internal Go struct representing a transaction (Date, Payee, Postings\[\]). We'll write a ToString() method on this struct that generates the ledger text, ensuring correct indentation for postings and alignment for amounts. This keeps the data representation separate from its text output.

## **2\. Interface: The Dashboard**

This is how the user interacts with the engine. The choice of bubbletea is key here, as it enforces The Elm Architecture (Model-View-Update), which is perfect for a stateful TUI like this.

### **A. State Management**

The entire application's state will live in a single Model struct.

* **Core Model Components:**  
  * currentView: A string or enum (batchView, transactionView, templateSelectView) to control what's being rendered.  
  * batch: A slice of our internal transaction structs (\[\]Transaction). This is the list of transactions being prepared.  
  * currentTransaction: A pointer to the transaction currently being edited in the transactionView. This includes its state (date, payee string, list of postings) and the state of the input fields themselves (cursor position, etc.).  
  * autocompleteModel: A struct containing the state for the autocomplete dropdown (list of suggestions, selected index).  
  * intelligenceDB: The in-memory model we built from parsing the ledger file (the Trie for accounts, the map for templates).  
  * errorState: To display any validation errors to the user.

### **B. Input Handling & UI Logic**

* **The Update Function:** This will be a large switch statement that handles incoming messages (tea.Msg). It will delegate keyboard inputs based on the currentView. For example, if the view is transactionView and the tea.KeyMsg is the letter 'n', it does nothing. But if the view is batchView, it switches the view to transactionView and initializes a new currentTransaction.  
* **Autocomplete Implementation:** This is a critical UI feature. When the user types a character in the Payee or an Account field, the Update function will:  
  1. Update the text in the corresponding field in the Model.  
  2. Query the intelligenceDB (the Trie for accounts, the list for payees) with the new partial string.  
  3. Update the autocompleteModel with the results.  
  4. The View function will then see the updated autocompleteModel and render the dropdown.  
  5. Handling Tab or Enter in the Update function will accept the currently selected suggestion and update the text field.  
* **Session Persistence:** The design calls for saving the batch to a .tmp file. This is a great, simple way to prevent data loss. After *every* state change that modifies the batch (confirming a new transaction, editing one), we'll serialize the batch slice to JSON (or Gob for Go-specificity) and write it to disk. On startup, we check for this file. If it exists, we deserialize it to populate the initial batch. This is far simpler than a formal database.

By keeping the business logic (parsing, analysis) separate from the interface logic (state management, rendering), we can build a clean, testable, and maintainable application. The key is to be disciplined about what data we extract and how we model it in memory, and to fully embrace the stateful nature of the bubbletea framework for the UI.