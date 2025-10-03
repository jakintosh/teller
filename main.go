package main

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"git.sr.ht/~jakintosh/teller/core"
	"git.sr.ht/~jakintosh/teller/intelligence"
	"git.sr.ht/~jakintosh/teller/parser"
	"git.sr.ht/~jakintosh/teller/session"
	"git.sr.ht/~jakintosh/teller/tui"
)

func main() {
	// Check for ledger file argument
	if len(os.Args) < 2 {
		fmt.Println("Usage: teller <ledger-file>")
		os.Exit(1)
	}

	ledgerFile := os.Args[1]

	// Parse the ledger file
	transactions, err := parser.ParseFile(ledgerFile)
	if err != nil {
		log.Fatalf("Failed to parse ledger file '%s': %v", ledgerFile, err)
	}

	// Build intelligence database
	db, err := intelligence.NewIntelligenceDB(transactions)
	if err != nil {
		log.Fatalf("Failed to create intelligence database: %v", err)
	}

	// Check for existing session and restore if available
	var previousBatch []core.Transaction
	if session.HasSession() {
		fmt.Print("Previous session found. Restore it? [y/N]: ")
		var response string
		fmt.Scanln(&response)
		if response == "y" || response == "Y" {
			previousBatch, err = session.LoadBatch()
			if err != nil {
				log.Printf("Warning: failed to load previous session: %v", err)
				previousBatch = []core.Transaction{}
			} else {
				fmt.Printf("Restored %d transactions from previous session.\n", len(previousBatch))
			}
		} else {
			// User declined to restore, delete the session file
			session.DeleteSession()
		}
	}

	// Create and start the TUI
	model := tui.NewModel(db, ledgerFile)
	if len(previousBatch) > 0 {
		model.SetBatch(previousBatch)
	}

	program := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := program.Run(); err != nil {
		log.Fatalf("TUI error: %v", err)
	}
}