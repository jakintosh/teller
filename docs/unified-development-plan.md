# **Phase 1: The Foundation (Core Data Structures & Parsing)**

The goal of this phase is to build the non-visual, backend foundation of the application. We will define the core data structures that represent financial transactions in memory and implement the logic required to read a user's existing ledger-cli file from disk and parse it into these structures. At the end of this phase, the application will be able to understand ledger data, but it will not yet have a user interface.

`// core/types.go`

`// Posting represents a single entry in a transaction.`  
`type Posting struct {`  
	`Account string // e.g., "Expenses:Food:Groceries"`  
	`Amount  string // e.g., "12.34" (stored as string for precision)`  
`}`

`// Transaction represents a complete financial event.`  
`type Transaction struct {`  
	`Date     time.Time`  
	`Payee    string // e.g., "Super Grocery Store"`  
	`Postings []Posting`  
`}`

`// parser/parser.go`

`// ParseFile reads a ledger-cli file and converts it into Transaction structs.`  
`func ParseFile(filePath string) ([]Transaction, error)`

## **Step 1.1: Project Setup & Core Data Types**

This step focuses on initializing the Go project and defining the primary data structures (Transaction and Posting). These structs translate the abstract concept of a financial transaction from the plain text file into a concrete, type-safe representation that the rest of the application can work with.

### **Tasks**

* Initialize a new Go module for the project.  
* Create a core package to hold the application's central data types.  
* Define the Posting struct with Account (string) and Amount (string) fields.  
* Define the Transaction struct with Date (time.Time), Payee (string), and Postings (\[\]Posting) fields.

### **Documentation**

* **Project Brief:** This document establishes the core need to interact with ledger-cli data.  
* **Ledger Documentation (ledger-cli.org/doc/ledger3.txt):** Refer to this for the canonical structure of a transaction, which these structs are designed to model.

### **Testing**

* Formal testing is not required for this step, as it only involves type definitions. The correctness of these structs will be validated by the parser tests in the next step.

## **Step 1.2: The Ledger File Parser**

The goal of this step is to implement the file-reading logic. The ParseFile function will be the application's entry point for all user data, responsible for opening the ledger file, reading its contents line by line, and transforming the raw text into the structured \[\]Transaction slice defined in the previous step.

### **Tasks**

* Create a parser package to contain the file parsing logic.  
* Implement the ParseFile function, which accepts a file path string.  
* Inside the function, open and read the contents of the file at the given path.  
* Loop through the file line-by-line, using regular expressions to identify the start of a transaction (a line beginning with a date).  
* For each transaction block, extract the date, the payee/description, and all subsequent indented posting lines.  
* For each posting line, parse the account name and the amount.  
* Populate the Transaction and Posting structs with the extracted data and append them to a slice.  
* Return the completed \[\]Transaction slice.

### **Documentation**

* **Ledger Documentation (ledger-cli.org/doc/ledger3.txt):** This is the primary reference for understanding the file format, including valid date formats, comment characters (';'), and how postings are structured.

### **Testing**

* Create a sample.ledger file containing 5-10 varied transactions, including ones with comments, different numbers of postings, and various formatting styles.  
* Write a unit test for ParseFile that reads this sample file.  
* Assert that the number of Transaction structs returned matches the number of transactions in the file.  
* Assert that the Date, Payee, Account, and Amount fields for a few specific transactions are parsed correctly.

# **Phase 2: The "Intelligence" Engine**

The goal of this phase is to build the application's in-memory "brain" from the data parsed in Phase 1\. This engine is not a persistent database but rather a temporary data store, created on startup, that is highly optimized for providing the instant feedback required for features like payee autocomplete, hierarchical account search, and transaction template suggestions.

`// intelligence/trie.go`

`// Trie is a prefix tree for efficient prefix-based string searching.`  
`type Trie struct { /* ... */ }`

`func NewTrie() *Trie`  
`func (t *Trie) Insert(word string)`  
`func (t *Trie) Find(prefix string) []string`

`// intelligence/db.go`

`// TemplateRecord stores a transaction structure and its frequency.`  
`type TemplateRecord struct {`  
	`Accounts  []string`  
	`Frequency int`  
`}`

`// IntelligenceDB is the in-memory data store for all suggestion features.`  
`type IntelligenceDB struct {`  
	`Payees    []string`  
	`Accounts  *Trie`  
	`Templates map[string][]TemplateRecord`  
`}`

`func NewIntelligenceDB(transactions []Transaction) (*IntelligenceDB, error)`  
`func (db *IntelligenceDB) FindPayees(prefix string) []string`  
`func (db *IntelligenceDB) FindAccounts(prefix string) []string`  
`func (db *IntelligenceDB) FindTemplates(payee string) []TemplateRecord`

## **Step 2.1: Baseline IntelligenceDB Structure**

This step focuses on creating the initial IntelligenceDB structure and populating it with the most basic learned data: the list of unique payees. This provides the foundation for all subsequent intelligence features.

### **Tasks**

* Create an intelligence package.  
* Define the IntelligenceDB struct, initially containing just a Payees \[\]string field.  
* Implement the NewIntelligenceDB constructor, which accepts the \[\]Transaction slice.  
* Inside the constructor, iterate through all transactions to populate the Payees slice, ensuring all entries are unique and sorted.  
* Implement the FindPayees(prefix string) \[\]string method to search the Payees slice.

### **Testing**

* Write unit tests for NewIntelligenceDB using a mock slice of transactions.  
* Assert that the Payees slice contains the correct number of unique, sorted payees.  
* Assert that FindPayees returns correct results for a given prefix.

## **Step 2.2: Trie Implementation for Account Autocomplete**

This step focuses on implementing the Trie data structure and integrating it into the IntelligenceDB. The Trie is essential for providing the fast, hierarchical, segment-by-segment autocomplete required for ledger account names.

### **Tasks**

* Implement a Trie data structure within the intelligence package, complete with Insert and Find methods.  
* Add an Accounts \*Trie field to the IntelligenceDB struct.  
* In the NewIntelligenceDB constructor, extend the transaction processing loop to iterate through every posting of every transaction.  
* For each posting, Insert the full account name (e.g., "Expenses:Food:Groceries") into the Accounts Trie.  
* Implement the FindAccounts(prefix string) \[\]string method, which uses the Trie to perform its search.

### **Documentation**

* **Ledger Helper TUI Design Document:** The mockups showing hierarchical account autocomplete (e.g., completing Expenses:Fo to Expenses:Food) are the primary justification for using a Trie data structure.

### **Testing**

* Write dedicated unit tests for the Trie to ensure Insert and Find work correctly, especially with multi-level prefixes.  
* Extend the NewIntelligenceDB tests to verify that the Accounts Trie is populated correctly.  
* Assert that FindAccounts("Expenses:Fo") returns the correct subset of accounts (e.g., \["Expenses:Food"\]).

## **Step 2.3: Transaction Template Inference**

This step implements the application's most advanced intelligence feature: learning common transaction structures. It involves analyzing transactions to find frequently used sets of posting accounts for each payee and ranking them by frequency.

### **Tasks**

* Define a new TemplateRecord struct to hold a slice of account names and their frequency.  
* Add a Templates map\[string\]\[\]TemplateRecord field to the IntelligenceDB struct.  
* In NewIntelligenceDB, add a new analysis loop that processes each transaction:  
  * For a given transaction, get the payee and the list of its posting accounts.  
  * To create a comparable key, sort the list of account names and join them into a single string.  
  * Use this key to find the corresponding TemplateRecord for that payee and increment its Frequency, or create a new record if one doesn't exist.  
* After processing all transactions, perform a final pass to sort each payee's \[\]TemplateRecord slice by Frequency in descending order.  
* Implement the new (db \*IntelligenceDB) FindTemplates(payee string) \[\]TemplateRecord method.

### **Documentation**

* **Project Brief:** References "Story \#4 (Automatic Template Suggestion)" as a core MVP feature.  
* **TUI Design Document:** The user action list includes "Receive and Select a Template for Allocating the Total".

### **Testing**

* Create a unit test with a mock slice of transactions where a payee ("City Market") has two transactions with the template \[Assets:Checking, Expenses:Groceries\] and one with a different template.  
* Call FindTemplates("City Market") on the resulting database.  
* Assert that the method returns two records.  
* Assert that the first record in the slice has a Frequency of 2 and matches the more common template.

# **Phase 3: The User Interface**

The goal of this phase is to build the entire user-facing terminal interface using the bubbletea framework. This involves creating the main application state model, implementing the two primary screens (Batch Review and Transaction Entry), and wiring up all user interactions, including text input, list navigation, and autocomplete display.

`// tui/model.go`

`type viewState int`  
`const (`  
	`batchView viewState = iota`  
	`transactionView`  
`)`

`// Represents a single line in the transaction form's posting list.`  
`type postingLine struct {`  
    `accountInput textinput.Model`  
    `amountInput  textinput.Model`  
`}`

`// Manages the state of the transaction entry form.`  
`type transactionForm struct {`  
    `dateInput    textinput.Model`  
    `payeeInput   textinput.Model`  
    `totalInput   textinput.Model`  
    `postings     []postingLine`  
    `focusedField int // Index or enum to track focus`  
`}`

`// Model is the single source of truth for the TUI application state.`  
`type Model struct {`  
	`db                      *IntelligenceDB`  
	`batch                   []Transaction`  
	`currentView             viewState`  
	`form                    transactionForm`  
	`autocompleteSuggestions []string`  
	`templateSuggestions     []TemplateRecord`  
	`cursor                  int`  
	`err                     error`  
	`statusMessage           string`  
	`statusExpiry            time.Time`  
`}`

`func NewModel(db *IntelligenceDB) Model`  
`func (m Model) Init() tea.Cmd`  
`func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd)`  
`func (m Model) View() string`

## **Step 3.1: TUI Scaffolding & Batch Review Screen**

This step covers building the main application shell and the primary "home" screen. We will set up the main bubbletea model and event loop, and implement the Batch Review screen, which allows the user to see the list of transactions they've entered and to initiate creating or editing a transaction.

### **Tasks**

* Create a tui package and define the main Model struct to hold all application state.  
* Implement the Init, Update, and View methods required by bubbletea.  
* In the application's main function, call the parser, create the IntelligenceDB, and start the bubbletea program with the new model.  
* In the View method, render the list of transactions in Model.batch when currentView is batchView.  
* In the Update method, handle Up/Down arrow keys to move the cursor through the batch list.  
* Handle the n key to switch currentView to transactionView, initializing a new, empty form.  
* Handle the e key to switch currentView to transactionView, pre-populating the form with data from the selected transaction for editing.

### **Documentation**

* **Ledger Helper TUI Design Document:** The "Batch Review Screen" mockup is the primary visual reference.

### **Testing**

* Manually test all keybindings on the Batch Review screen (n, e, q, arrows) to ensure they work as expected.  
* Verify that editing a transaction correctly pre-populates the form.

## **Step 3.2: Transaction Form Workflow & State Management**

The goal of this step is to implement the detailed mechanics of the transaction entry form. This includes managing a dynamic number of posting lines, implementing the "total-first" workflow with a real-time "Remaining" balance calculation, and handling the logic for balancing the final split automatically.

### **Tasks**

* Define a transactionForm struct within the tui.Model to manage the form's state, including a dynamic slice of postingLine structs.  
* Each postingLine struct will contain its own accountInput and amountInput textinput.Model.  
* Use a decimal library (e.g., shopspring/decimal) for all monetary calculations to avoid floating-point errors.  
* In the Update function, after any change to the totalInput or any amountInput, recalculate and update a remainingBalance field in the model.  
* Render the remainingBalance in the View function, providing the user with real-time feedback.  
* Implement logic to handle the b ("balance last split") keypress. This function should calculate the sum of all *other* splits and set the value of the currently focused amount field to the remaining balance.  
* Implement handlers for adding (Ctrl+N) and deleting (Ctrl+D) postingLine entries from the form's state. Ensure focus is managed correctly after these actions.

### **Documentation**

* **Ledger Helper TUI Design Document:** The mockups showing a "Remaining" balance and a \[b\]alance last split keybinding hint are the primary references for this workflow.

### **Testing**

* Manually test the form's functionality:  
  * Enter a total of 100.00. Add three posting lines.  
  * In the first, enter 25.00. Verify the "Remaining" balance updates to 75.00.  
  * In the second, enter 50.00. Verify the "Remaining" balance updates to 25.00.  
  * Focus the third amount field and press b. Verify the field's value is automatically set to 25.00 and the "Remaining" balance is 0.00.  
  * Verify that adding and deleting posting lines works as expected.

## **Step 3.3: Context-Sensitive Input Routing**

This step focuses on implementing the core input routing logic that allows the TUI to differentiate between text entry and application commands. The main Update function will act as a central router, delegating keypresses based on which component is currently focused, rather than relying on separate modes.

### **Tasks**

* In the transactionForm state, add an integer or enum field named focusedField to track which UI element is active (e.g., Payee, Total, Posting 1 Account, etc.).  
* In the Update function, structure the tea.KeyMsg handling as a context-sensitive hierarchy:  
  1. First, check for global commands that always apply (e.g., ctrl+c to quit).  
  2. Second, check for navigation commands like Tab or Shift+Tab. These commands will increment or decrement focusedField and call the .Focus() and .Blur() methods on the relevant components.  
  3. Third, check if the currently focused component is a textinput that is active (.Focused() \== true).  
  4. **If a text input is focused**, delegate the tea.KeyMsg directly to that component's Update method. This allows the component to handle character input internally.  
  5. **If no text input is focused**, treat the keypress as a form-level command (e.g., c for confirm, b for balance).  
* Implement a status/help bar in the View that dynamically displays available commands based on the focusedField.

### **Testing**

* Manually verify the input routing logic:  
  * Focus the Payee field. Type "Coffee Shop". Verify the characters appear in the input and are not interpreted as commands.  
  * Press Tab until no text field is focused. Verify the help text changes to show \[c\]onfirm.  
  * Press c. Verify the transaction is confirmed.  
  * Focus an amount field for a posting. Press b. Verify it triggers the balance logic.  
  * Focus the Payee field again. Press b. Verify it simply types the letter "b" in the field.

# **Phase 4: Final Polish & Integration**

The goal of this final phase is to add the critical features that make the application robust and truly useful. This includes implementing session persistence to prevent data loss, adding an inline calculator to streamline data entry, building the final logic that writes the user's completed batch of transactions back to their ledger file, and implementing a user-friendly error handling system.

`// session/session.go`

`// saveBatch serializes the current batch to a temporary file.`  
`func saveBatch(filePath string, batch []Transaction) error`

`// loadBatch deserializes a batch from a temporary file.`  
`func loadBatch(filePath string) ([]Transaction, error)`

`// util/calculator.go`

`// evaluateExpression evaluates a mathematical string like "19.99 * 2".`  
`func evaluateExpression(expr string) (string, error)`

`// core/types.go (addition)`

`// String formats a transaction into a valid ledger-cli text entry.`  
`func (t *Transaction) String() string`

`// ledger/writer.go`

`// commitBatch appends a batch of transactions to the ledger file.`  
`func commitBatch(ledgerFile string, batch []Transaction) error`

## **Step 4.1: Session Persistence**

This step implements a crucial data safety feature. To prevent the user from losing their work if the application crashes or is closed accidentally, the current batch of transactions will be automatically saved to a temporary file after every change.

### **Tasks**

* Create a session package.  
* Implement saveBatch to serialize the Model.batch slice into a temporary JSON file (e.g., .ledger-helper-batch.tmp).  
* Implement loadBatch to read and deserialize this file back into a \[\]Transaction.  
* In the TUI's Update function, call saveBatch whenever a transaction is added to or edited in the batch.  
* In the main application entry point, before starting the TUI, check for the existence of the temp file and call loadBatch to restore the previous session.

### **Documentation**

* **Ledger Helper: Architectural Choices:** This document explicitly outlines the requirement for session persistence using a temporary file.  
* **Technical Requirements:** Section 4.4 details the requirements for saving, loading, and deleting the temporary session file.

### **Testing**

* Manually add several transactions to a batch.  
* Force-quit the application (e.g., using ctrl+c).  
* Restart the application and verify that the previously entered transactions are loaded and displayed correctly.  
* After a successful commit (Step 4.3), verify that the temporary file has been deleted.

## **Step 4.2: Inline Calculator & Transaction Formatting**

This step adds two high-value "quality of life" features. The inline calculator removes the need for the user to do manual arithmetic, and the transaction formatter ensures the application's output is always in the correct ledger-cli format.

### **Tasks**

* Select and integrate a lightweight third-party Go math expression evaluation library.  
* Implement the evaluateExpression wrapper function.  
* In the TUI's Update function, when an amount field loses focus, call evaluateExpression on its contents and update the field with the result if successful.  
* Implement the String() method on the core.Transaction type. This method will format the struct's data into a multi-line, correctly indented string that conforms to the ledger file syntax.

### **Documentation**

* **Project Brief:** This document lists "Inline Calculation for Splits" as a key feature of the MVP.  
* **Technical Requirements:** Section 3.1 details the required operators and the need for a decimal-safe math library.

### **Testing**

* Write unit tests for evaluateExpression with various inputs, including addition, multiplication, and order of operations (e.g., "10+5\*2").  
* Write unit tests for the Transaction.String() method to assert that its output perfectly matches the expected, correctly formatted ledger-cli text block.

## **Step 4.3: Committing to Ledger File**

This is the final and most important step, where the user's work is made permanent. This step implements the logic to take the verified batch of in-memory transactions, format them using the method from the previous step, and append them to the user's source-of-truth ledger file.

### **Tasks**

* In the Batch Review screen's Update function, add a case to handle the w key ("write").  
* When w is pressed, call a new commitBatch function.  
* Inside commitBatch, open the user's main ledger file in append mode.  
* Iterate through the Model.batch slice, call the .String() method on each transaction, and write the resulting string to the file.  
* After the file write is successfully completed, delete the temporary session file from Step 4.1.  
* Exit the application.

### **Documentation**

* **Ledger Helper: Architectural Choices:** This document defines the "Single Source of Truth" principle, emphasizing that the application reads from and appends to the user's file but does not modify it in place.

### **Testing**

* Perform a full, end-to-end manual test: start the app, add 2-3 transactions, press w to commit.  
* Open the actual ledger file and verify that the new entries have been appended at the end and are formatted correctly.  
* Confirm that the temporary session file (.ledger-helper-batch.tmp) no longer exists after the commit.

## **Step 4.4: Implement UI Error Handling**

The goal of this step is to implement a user-friendly error handling system within the TUI. This ensures that both recoverable, non-fatal errors (like invalid input) and fatal errors (like file access issues) are communicated clearly to the user instead of causing a silent failure or a crash.

### **Tasks**

* Add two fields to the main tui.Model: statusMessage string and statusExpiry time.Time.  
* For recoverable errors (e.g., an invalid expression in the inline calculator), set the statusMessage to the error text and statusExpiry to 5 seconds in the future.  
* Use a tea.Tick message (e.g., every second) in the Update loop to check if the statusExpiry has passed, and if so, clear the statusMessage.  
* Render the statusMessage in the status bar area of the View.  
* For fatal errors on startup (e.g., ParseFile fails), populate the main Model.err field.  
* In the View function, add a top-level check: if Model.err is not nil, render a full-screen, polite error message and halt rendering of any other UI components.

### **Testing**

* Manually test recoverable errors:  
  * In an amount field, type an invalid math expression like "10 \+\* 5" and tab away.  
  * Verify that an "Invalid expression" message appears in the status bar for a few seconds and then disappears.  
* Test fatal errors:  
  * Try to run the application on a non-existent or permission-denied ledger file.  
  * Verify that the application starts up and displays a clear, full-screen error message explaining that the file could not be read.