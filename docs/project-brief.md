# **Project Brief: A High-Velocity Data Entry Tool for Plain Text Accounting**

* **Document Version:** 1.0  
* **Last Updated:** September 11, 2025

# **1\. Project Overview**

This document outlines the requirements for a tool designed to solve the data entry problem for an established plain text accounting user. The user appreciates the ledger-cli system as a data storage and reporting format but finds the process of manually entering transactions into a text file to be tedious, error-prone, and a significant point of friction.

The goal of this project is to create a helper application that facilitates rapid, accurate, and intelligent data entry, which then outputs the standard plain text format required by ledger-cli.

# **2\. Target User Profile**

* **Operating System:** Arch Linux, with a heavily terminal-based environment.  
* **Core Tools:** ledger-cli for accounting, Helix (a terminal-based editor), git for version control and synchronization.  
* **Expertise:** A technical power user comfortable with command-line interfaces and plain text formats.  
* **Philosophy:** Values self-hosted, local-first software. Avoids cloud-based subscription services and proprietary data formats.

# **3\. Core Problem Statement**

The primary user finds direct text-file entry to be a source of high friction, leading to procrastination. Data entry is often delayed for a month or more, resulting in daunting, multi-hour "data entry marathons."

The key pain points with the current workflow are:

* **High Redundancy:** Constantly re-typing long, exact account names, payee names, and dates.  
* **High Cognitive Load:** The process requires significant context-switching and manual recall of past categorization decisions, payee-specific transaction structures, and complex arithmetic for split transactions.  
* **Error-Prone:** The manual process is susceptible to typos in account names (creating new, incorrect categories) and mathematical errors in transaction amounts, which require manual "hunts" to find and fix.

# **4\. Key Decisions & Constraints**

Throughout our discussion, we have established several core principles that must guide development:

* **Platform:** The tool must be a **Command-Line Interface (CLI) or Text-based User Interface (TUI)** to integrate seamlessly into the user's existing workflow.  
* **Output:** The tool's sole output must be valid, plain-text ledger formatted entries that are appended to the user's existing journal file. It is a data *entry* tool, not a data *format replacement*.  
* **Data Source:** The tool’s "intelligence" (for auto-completion, suggestions) must be derived by parsing the user's existing ledger file(s).  
* **Hosting:** The solution must be **entirely self-hosted and run locally**. No cloud components, external AI/ML services, or subscription models are to be used.

# **5\. Finalized User Stories**

The following user stories represent the complete set of desired features for the tool.

## **Core Experience & Data Entry**

1. **Story: Account & Payee Auto-completion—**As a user entering a transaction, I want the tool to **auto-complete** payee and account names based on my existing ledger file so that I can enter data faster and avoid typos.  
2. **Story: Intelligent Category Suggestion—**As a user entering a transaction, I want the tool to **suggest a posting account** based on the payee's history so that I don't have to manually look up how I categorized it last time.  
3. **Story: Inline Calculation for Splits—**As a user entering a complex split transaction, I want the tool to provide an **inline calculator** that lets me **sum multiple line-item calculations** (each with its own base price, discounts, and taxes) **into a single category total**, so that the tool handles all the complex receipt arithmetic for me.  
4. **Story: Automatic Template Suggestion—**As a user entering a payee, I want the tool to **automatically analyze that payee's history and suggest one or more common transaction structures**, so that I can **pre-fill the entire transaction** with a likely template and simply adjust the numbers.

## **Workflow & Environment**

5. **Story: Batch Entry Workflow—**As a user at my desk, I want an interface that allows me to **rapidly process a list of transactions from a single source** so that my workflow is efficient and focused.  
6. **Story: Terminal-First Interface—**As a power user, I want the tool to be a **CLI or TUI** so that it integrates seamlessly into my existing terminal environment.

## **Advanced Automation & Capture**

7. **Story: Two-Stage Transaction Capture—**As a user on the go, I want a way to **quickly capture partial transaction data** which creates a "draft," so that I can lower the barrier to capturing data in the moment and finalize it later.  
8. **Story: CSV Statement Importing—**As a user starting a data entry session, I want to **import a CSV file from my bank** so that the tool can pre-populate a list of transactions for me to review and categorize, massively reducing manual typing.

# **6\. Minimum Viable Product (MVP) Scope: The "Intelligent Entry" Core**

To deliver value as quickly as possible, the initial build will focus on the features that provide the most immediate relief from the core pain points of manual data entry.

## **In Scope for MVP:**

* **Foundation: Story \#6 (Terminal-First Interface)**: The application will be a TUI.  
* **Feature: Story \#5 (Batch Entry Workflow)**: The interface will be designed around processing batches of transactions from a single source.  
* **Feature: Story \#1 (Account & Payee Auto-completion)**: To speed up typing and eliminate typos.  
* **Feature: Story \#4 (Automatic Template Suggestion)**: To handle structural complexity and reduce cognitive load.  
* **Feature: Story \#3 (Inline Calculation for Splits)**: To eliminate the most complex manual arithmetic.

## **Out of Scope for MVP:**

The following features are recognized as valuable but will be deferred to a future version to ensure the initial product is focused and delivered effectively:

* Story \#2: Intelligent Category Suggestion (Distinct from template suggestion).  
* Story \#7: Two-Stage Transaction Capture.  
* Story \#8: CSV Statement Importing.

# **7\. Next Steps**

With the requirements and MVP scope defined, the project now moves from the **"what"** to the **"how"**. The immediate next step is to begin the **Design and Architecture Phase**, which should focus on answering the following questions:

1. **Interaction Design:** What is the optimal layout and user flow for the TUI, especially for a batch-oriented workflow? How are suggestions presented and selected? How is the inline calculator invoked and used?  
2. **System Architecture:** What is the strategy for parsing the user's ledger file efficiently to power the application's features? What data structures will be needed to store the learned payee and account information in memory?