package tui

import (
	"time"

	"git.sr.ht/~jakintosh/teller/core"
	"git.sr.ht/~jakintosh/teller/intelligence"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/shopspring/decimal"
)

// Constants define UI behavior and tolerances
const (
	statusDuration       = 5 * time.Second
	statusShortDuration  = 3 * time.Second
	maxSuggestionDisplay = 5
	balanceTolerance     = 0.01
	maxTemplateDisplay   = 5
)

// viewState represents the current screen being displayed
type viewState int

const (
	viewBatch viewState = iota
	viewTransaction
	viewTemplate
	viewConfirm
)

// confirmKind represents the type of confirmation being requested
type confirmKind int

const (
	confirmWrite confirmKind = iota
	confirmQuit
)

// focusedField represents which field currently has user focus
type focusedField int

const (
	focusDate focusedField = iota
	focusCleared
	focusPayee
	focusComment
	focusTemplateButton
	focusSectionAccount
	focusSectionAmount
	focusSectionComment
)

// sectionType distinguishes between debit and credit sections
type sectionType int

const (
	sectionDebit sectionType = iota
	sectionCredit
)

// dateSegment represents which part of the date is selected
type dateSegment int

const (
	dateSegmentYear dateSegment = iota
	dateSegmentMonth
	dateSegmentDay
)

// Model is the main application state container for the TUI
type Model struct {
	db             *intelligence.IntelligenceDB
	ledgerFilePath string
	loadSummary    core.LoadSummary

	batch       []core.Transaction
	cursor      int
	currentView viewState

	form            transactionForm
	templateOptions []intelligence.TemplateRecord
	templateCursor  int
	templateOffset  int
	templatePayee   string
	pendingConfirm  confirmKind
	editingIndex    int

	lastDate      time.Time
	statusMessage string
	statusExpiry  time.Time
	err           error
}

// transactionForm holds the state for the transaction entry form
type transactionForm struct {
	date           dateField
	cleared        bool
	payeeInput     textinput.Model
	commentInput   textinput.Model
	debitLines     []postingLine
	creditLines    []postingLine
	focusedField   focusedField
	focusedSection sectionType
	focusedIndex   int
	remaining      decimal.Decimal
	debitTotal     decimal.Decimal
	creditTotal    decimal.Decimal
}

// postingLine represents a single debit or credit line in the form
type postingLine struct {
	accountInput textinput.Model
	amountInput  textinput.Model
	commentInput textinput.Model
}

// dateField manages a date with segment-based navigation
type dateField struct {
	year    int
	month   int
	day     int
	segment dateSegment
	buffer  string
}

// statusTick is sent periodically to update status message expiry
type statusTick struct{}
