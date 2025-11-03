package tui

import (
	"time"

	"git.sr.ht/~jakintosh/teller/internal/core"
	"git.sr.ht/~jakintosh/teller/internal/intelligence"
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
	confirmNone confirmKind = iota
	confirmWrite
	confirmQuit
	confirmDiscard
)

// statusKind represents the type of status message being displayed
type statusKind int

const (
	statusInfo statusKind = iota
	statusSuccess
	statusError
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

// focusPosition uniquely identifies a focusable element in the form
// This allows us to build a spatial focus path that matches the visual layout
type focusPosition struct {
	field   focusedField
	section sectionType
	index   int
}

// Model is the main application state container for the TUI
type Model struct {
	db             *intelligence.IntelligenceDB
	ledgerFilePath string
	buildReport    intelligence.BuildReport

	batch       []core.Transaction
	cursor      int
	currentView viewState
	batchOffset int

	form              transactionForm
	formBaseline      formSnapshot
	templateOptions   []intelligence.TemplateRecord
	templateCursor    int
	templateOffset    int
	templatePayee     string
	pendingConfirm    confirmKind
	confirmReturnView viewState
	editingIndex      int

	windowHeight  int
	lastDate      time.Time
	statusMessage string
	statusKind    statusKind
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

type formSnapshot struct {
	date    time.Time
	cleared bool
	payee   string
	comment string
	debit   []postingSnapshot
	credit  []postingSnapshot
}

type postingSnapshot struct {
	account string
	amount  string
	comment string
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
