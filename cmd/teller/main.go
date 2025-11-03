package main

import (
	"fmt"
	"log"

	"git.sr.ht/~jakintosh/command-go/pkg/args"
	"git.sr.ht/~jakintosh/teller/internal/core"
	"git.sr.ht/~jakintosh/teller/internal/intelligence"
	"git.sr.ht/~jakintosh/teller/internal/parser"
	"git.sr.ht/~jakintosh/teller/internal/session"
	"git.sr.ht/~jakintosh/teller/internal/tui"
	"git.sr.ht/~jakintosh/teller/internal/version"
	tea "github.com/charmbracelet/bubbletea"
)

var versionInfo = version.Data()

var versionCmd = &args.Command{
	Name: "version",
	Help: "print detailed version information",
	Options: []args.Option{
		{
			Short: 'v',
			Long:  "verbose",
			Type:  args.OptionTypeFlag,
			Help:  "display verbose version information",
		},
	},
	Handler: func(i *args.Input) error {
		if i.GetFlag("verbose") {
			fmt.Printf("teller %s\n", versionInfo.Version)
			fmt.Printf("commit:\t%s\n", versionInfo.Commit)
			fmt.Printf("built:\t%s\n", versionInfo.BuildDate)
		} else {
			fmt.Println(versionInfo.Version)
		}
		return nil
	},
}

var root = &args.Command{
	Name:    "teller",
	Author:  "Jakintosh",
	Version: versionInfo.Version,
	Help:    "Categorize ledger transactions in a terminal UI.",
	Operands: []args.Operand{
		{
			Name: "ledger-file",
			Help: "path to the ledger file",
		},
	},
	Subcommands: []*args.Command{
		versionCmd,
	},
	Handler: func(i *args.Input) error {

		// read operands
		ledgerFile := i.GetOperand("ledger-file")

		// Parse the ledger file
		parseResult, err := parser.ParseFile(ledgerFile)
		if err != nil {
			log.Fatalf("Failed to parse ledger file '%s': %v", ledgerFile, err)
		}

		// Build intelligence database
		db, buildReport, err := intelligence.NewIntelligenceDB(parseResult)
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
		model := tui.NewModel(db, ledgerFile, buildReport)
		if len(previousBatch) > 0 {
			model.SetBatch(previousBatch)
		}

		program := tea.NewProgram(model, tea.WithAltScreen())

		if _, err := program.Run(); err != nil {
			log.Fatalf("TUI error: %v", err)
		}

		return nil
	},
}

func main() {
	root.Parse()
}
