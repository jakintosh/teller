package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"git.sr.ht/~jakintosh/teller/internal/core"
	"git.sr.ht/~jakintosh/teller/internal/intelligence"
	"git.sr.ht/~jakintosh/teller/internal/parser"
	"git.sr.ht/~jakintosh/teller/internal/session"
	"git.sr.ht/~jakintosh/teller/internal/tui"
	"git.sr.ht/~jakintosh/teller/internal/version"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	flagSet := flag.NewFlagSet("teller", flag.ExitOnError)
	flagSet.Usage = func() {
		fmt.Fprintln(flagSet.Output(), "Usage: teller [--version] <ledger-file>")
		fmt.Fprintln(flagSet.Output(), "       teller version")
	}
	showVersion := flagSet.Bool("version", false, "print version information and exit")
	shortVersion := flagSet.Bool("v", false, "print version information and exit")

	if err := flagSet.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}

	if *showVersion || *shortVersion {
		printShortVersion()
		return
	}

	if flagSet.NArg() > 0 && flagSet.Arg(0) == "version" {
		printDetailedVersion()
		return
	}

	if flagSet.NArg() < 1 {
		flagSet.Usage()
		os.Exit(1)
	}

	ledgerFile := flagSet.Arg(0)

	// Parse the ledger file
	parseResult, err := parser.ParseFile(ledgerFile)
	if err != nil {
		log.Fatalf("Failed to parse ledger file '%s': %v", ledgerFile, err)
	}

	transactions := parseResult.Transactions

	// Build intelligence database
	db, buildReport, err := intelligence.NewIntelligenceDB(transactions)
	if err != nil {
		log.Fatalf("Failed to create intelligence database: %v", err)
	}

	loadSummary := core.LoadSummary{
		Transactions:    len(transactions),
		UniquePayees:    buildReport.UniquePayees,
		UniqueTemplates: buildReport.UniqueTemplates,
	}

	for _, issue := range parseResult.Issues {
		loadSummary.Issues = append(loadSummary.Issues, core.LoadIssue{
			Stage:   "parser",
			Message: fmt.Sprintf("line %d: %s", issue.Line, issue.Message),
		})
	}

	for _, msg := range buildReport.Issues {
		loadSummary.Issues = append(loadSummary.Issues, core.LoadIssue{
			Stage:   "intelligence",
			Message: msg,
		})
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
	model := tui.NewModel(db, ledgerFile, loadSummary)
	if len(previousBatch) > 0 {
		model.SetBatch(previousBatch)
	}

	program := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := program.Run(); err != nil {
		log.Fatalf("TUI error: %v", err)
	}
}

func printShortVersion() {
	info := version.Data()
	fmt.Println(info.Version)
}

func printDetailedVersion() {
	info := version.Data()
	fmt.Printf("teller %s\n", info.Version)
	fmt.Printf("commit:\t%s\n", info.Commit)
	fmt.Printf("built:\t%s\n", info.BuildDate)
}
