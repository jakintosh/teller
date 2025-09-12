# **Ledger Helper: Architectural Choices**

This document provides a high-level summary of the architectural and technical decisions made for the ledger-helper project.

## **1\. Core Technology Stack**

* **Language:** Go  
* **UI Framework:** bubbletea

**Rationale:** The choice of Go was driven by the end-user's expertise in the language, which ensures development velocity and long-term maintainability. Go's ability to compile to a single, statically-linked binary is ideal for a command-line utility, providing simple, dependency-free distribution. The bubbletea framework provides a robust, stateful model (The Elm Architecture) for building the required Text-based User Interface (TUI).

## **2\. System Architecture**

The application is designed as a **stateless command-line utility**.

* **Single Source of Truth:** The user's existing ledger-cli text file is the sole data source. On startup, the application parses this file to build a temporary, in-memory model of payees, accounts, and transaction structures.  
* **No Database:** The application does not require an external database or persistent application state. Its "knowledge" is rebuilt from the ledger file at the beginning of each session.  
* **Session Persistence:** To prevent data loss from crashes or interruptions, the current working batch of transactions is saved to a temporary file (.ledger-helper-batch.tmp) after every modification. This file is deleted upon a successful write to the main ledger file.

## **3\. Interface Design**

* **Modality:** The application will be a terminal-based Text-based User Interface (TUI), designed to integrate seamlessly into the user's existing command-line workflow.  
* **Interaction Model:** The interface is designed around a "total-first" batch entry workflow, where the user first defines the total amount of a transaction and then allocates it across various splits.  
* **Key Feature:** A sophisticated, context-aware autocomplete system is a core component of the UI, supporting single suggestions, multi-option dropdowns, and segment-by-segment completion for hierarchical accounts.