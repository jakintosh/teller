// Package tui implements a terminal user interface for ledger transaction entry.
// The TUI provides an interactive form for entering double-entry accounting transactions
// with autocomplete support for accounts and payees, template-based transaction creation,
// and batch processing with session persistence.
package tui

import (
	"fmt"
	"time"

	"git.sr.ht/~jakintosh/teller/internal/intelligence"
	"git.sr.ht/~jakintosh/teller/internal/session"
	tea "github.com/charmbracelet/bubbletea"
)

// NewModel creates a new TUI model with the given intelligence database and ledger file path
func NewModel(db *intelligence.IntelligenceDB, ledgerFilePath string, report intelligence.BuildReport) *Model {
	m := &Model{
		db:             db,
		ledgerFilePath: ledgerFilePath,
		currentView:    viewBatch,
		editingIndex:   -1,
		buildReport:    report,
	}
	m.resetForm(time.Now())
	return m
}

// Init initializes the model and returns the initial command
func (m *Model) Init() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg { return statusTick{} })
}

// Update handles incoming messages and updates the model state
func (m *Model) Update(msg tea.Msg) (updated tea.Model, cmd tea.Cmd) {
	defer func() {
		if recovered := recover(); recovered != nil {
			recoveredErr := fmt.Errorf("unexpected internal error: %v", recovered)
			if len(m.batch) > 0 {
				if saveErr := session.SaveBatch(m.batch); saveErr != nil {
					recoveredErr = fmt.Errorf("%w (failed to persist batch session: %v)", recoveredErr, saveErr)
				} else {
					recoveredErr = fmt.Errorf("%w (pending batch session was saved)", recoveredErr)
				}
			}
			m.err = recoveredErr
			updated = m
			cmd = nil
		}
	}()

	if m.err != nil {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "ctrl+q", "ctrl+c":
				return m, tea.Quit
			}
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowHeight = msg.Height
		m.ensureBatchCursorVisible()
		return m, nil
	case statusTick:
		if !m.statusExpiry.IsZero() && time.Now().After(m.statusExpiry) {
			m.statusMessage = ""
			m.statusExpiry = time.Time{}
		}
		return m, tea.Tick(time.Second, func(time.Time) tea.Msg { return statusTick{} })
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

// View renders the current view based on the model state
func (m *Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress ctrl+q to quit.", m.err)
	}

	switch m.currentView {
	case viewBatch:
		return m.renderBatchView()
	case viewTransaction:
		return m.renderTransactionView()
	case viewTemplate:
		return m.renderTemplateView()
	case viewConfirm:
		return m.renderConfirmView()
	default:
		return "Unknown view"
	}
}
