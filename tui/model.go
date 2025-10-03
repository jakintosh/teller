package tui

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"git.sr.ht/~jakintosh/teller/core"
	"git.sr.ht/~jakintosh/teller/intelligence"
	"git.sr.ht/~jakintosh/teller/session"
	"git.sr.ht/~jakintosh/teller/util"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/shopspring/decimal"
)

const (
	statusDuration       = 5 * time.Second
	statusShortDuration  = 3 * time.Second
	maxSuggestionDisplay = 5
	balanceTolerance     = 0.01
	maxTemplateDisplay   = 5
)

type viewState int

type confirmKind int

type focusedField int

type sectionType int

type dateSegment int

const (
	viewBatch viewState = iota
	viewTransaction
	viewTemplate
	viewConfirm
)

const (
	confirmWrite confirmKind = iota
	confirmQuit
)

const (
	focusDate focusedField = iota
	focusPayee
	focusTemplateButton
	focusSectionAccount
	focusSectionAmount
)

const (
	sectionDebit sectionType = iota
	sectionCredit
)

const (
	dateSegmentYear dateSegment = iota
	dateSegmentMonth
	dateSegmentDay
)

type Model struct {
	db             *intelligence.IntelligenceDB
	ledgerFilePath string

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

type transactionForm struct {
	date           dateField
	payeeInput     textinput.Model
	debitLines     []postingLine
	creditLines    []postingLine
	focusedField   focusedField
	focusedSection sectionType
	focusedIndex   int
	remaining      decimal.Decimal
	debitTotal     decimal.Decimal
	creditTotal    decimal.Decimal
}

type postingLine struct {
	accountInput textinput.Model
	amountInput  textinput.Model
}

type dateField struct {
	year    int
	month   int
	day     int
	segment dateSegment
	buffer  string
}

func NewModel(db *intelligence.IntelligenceDB, ledgerFilePath string) *Model {
	m := &Model{
		db:             db,
		ledgerFilePath: ledgerFilePath,
		currentView:    viewBatch,
		editingIndex:   -1,
	}
	m.resetForm(time.Now())
	return m
}

func (m *Model) SetBatch(batch []core.Transaction) {
	m.batch = append([]core.Transaction(nil), batch...)
	sort.Slice(m.batch, func(i, j int) bool { return m.batch[i].Date.Before(m.batch[j].Date) })
	if len(m.batch) == 0 {
		m.cursor = 0
		m.lastDate = time.Time{}
		return
	}
	m.cursor = len(m.batch) - 1
	m.lastDate = m.batch[m.cursor].Date
}

func (m *Model) resetForm(baseDate time.Time) {
	m.form = newTransactionForm(baseDate)
	m.templateOptions = nil
	m.templateCursor = 0
	m.templateOffset = 0
	m.templatePayee = ""
	m.editingIndex = -1
}

func newTransactionForm(baseDate time.Time) transactionForm {
	if baseDate.IsZero() {
		baseDate = time.Now()
	}
	date := dateField{}
	date.setTime(baseDate)
	date.segment = dateSegmentDay

	payee := newTextInput("Payee")

	debit := []postingLine{newPostingLine()}
	credit := []postingLine{newPostingLine()}

	return transactionForm{
		date:           date,
		payeeInput:     payee,
		debitLines:     debit,
		creditLines:    credit,
		focusedField:   focusDate,
		focusedSection: sectionDebit,
		focusedIndex:   0,
		remaining:      decimal.Zero,
		debitTotal:     decimal.Zero,
		creditTotal:    decimal.Zero,
	}
}

func newTextInput(placeholder string) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Prompt = ""
	ti.CharLimit = 256
	ti.Width = 40
	ti.ShowSuggestions = true
	return ti
}

func newPostingLine() postingLine {
	account := newTextInput("Account")
	amount := newTextInput("Amount")
	amount.ShowSuggestions = false
	return postingLine{accountInput: account, amountInput: amount}
}

func (d *dateField) setTime(t time.Time) {
	d.year = t.Year()
	d.month = int(t.Month())
	d.day = t.Day()
	d.segment = dateSegmentYear
	d.buffer = ""
}

func (d dateField) time() time.Time {
	if d.year == 0 || d.month == 0 || d.day == 0 {
		return time.Time{}
	}
	return time.Date(d.year, time.Month(d.month), d.day, 0, 0, 0, 0, time.Local)
}

func (d dateField) display(focused bool) string {
	parts := []string{
		fmt.Sprintf("%04d", d.year),
		fmt.Sprintf("%02d", d.month),
		fmt.Sprintf("%02d", d.day),
	}
	if focused {
		parts[d.segment] = "[" + parts[d.segment] + "]"
	}
	return strings.Join(parts, "-")
}

func (d *dateField) segmentLeft() {
	d.buffer = ""
	if d.segment > dateSegmentYear {
		d.segment--
	}
}

func (d *dateField) segmentRight() {
	d.buffer = ""
	if d.segment < dateSegmentDay {
		d.segment++
	}
}

func (d *dateField) increment(delta int) {
	switch d.segment {
	case dateSegmentYear:
		d.year += delta
	case dateSegmentMonth:
		d.month += delta
		if d.month < 1 {
			d.month = 12
			d.year--
		} else if d.month > 12 {
			d.month = 1
			d.year++
		}
	case dateSegmentDay:
		t := d.time()
		if t.IsZero() {
			t = time.Now()
		}
		t = t.AddDate(0, 0, delta)
		d.year = t.Year()
		d.month = int(t.Month())
		d.day = t.Day()
	}
	d.ensureDayInMonth()
}

func (d *dateField) handleDigit(r rune) {
	if r < '0' || r > '9' {
		return
	}
	d.buffer += string(r)
	switch d.segment {
	case dateSegmentYear:
		if len(d.buffer) > 4 {
			d.buffer = d.buffer[len(d.buffer)-4:]
		}
		if val, err := strconv.Atoi(d.buffer); err == nil {
			d.year = val
		}
	case dateSegmentMonth:
		if len(d.buffer) > 2 {
			d.buffer = d.buffer[len(d.buffer)-2:]
		}
		if val, err := strconv.Atoi(d.buffer); err == nil {
			if val < 1 {
				val = 1
			}
			if val > 12 {
				val = 12
			}
			d.month = val
		}
		if len(d.buffer) >= 2 {
			d.segmentRight()
		}
	case dateSegmentDay:
		if len(d.buffer) > 2 {
			d.buffer = d.buffer[len(d.buffer)-2:]
		}
		if val, err := strconv.Atoi(d.buffer); err == nil {
			maxDay := daysInMonth(d.year, d.month)
			if val < 1 {
				val = 1
			}
			if val > maxDay {
				val = maxDay
			}
			d.day = val
		}
		if len(d.buffer) >= 2 {
			d.segmentRight()
		}
	}
	d.ensureDayInMonth()
}

func (d *dateField) ensureDayInMonth() {
	maxDay := daysInMonth(d.year, d.month)
	if d.day > maxDay {
		d.day = maxDay
	}
	if d.day < 1 {
		d.day = 1
	}
}

func daysInMonth(year, month int) int {
	if month < 1 || month > 12 {
		return 31
	}
	t := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	return t.AddDate(0, 1, -1).Day()
}

type statusTick struct{}

func (m *Model) Init() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg { return statusTick{} })
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
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

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.currentView {
	case viewBatch:
		return m, m.updateBatchView(msg)
	case viewTransaction:
		return m, m.updateTransactionView(msg)
	case viewTemplate:
		return m, m.updateTemplateView(msg)
	case viewConfirm:
		return m, m.updateConfirmView(msg)
	default:
		return m, nil
	}
}

func (m *Model) updateBatchView(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "ctrl+q", "ctrl+c":
		return tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.batch)-1 {
			m.cursor++
		}
	case "n":
		m.startNewTransaction()
	case "e", "enter":
		if len(m.batch) > 0 {
			m.startEditingTransaction(m.cursor)
		}
	case "w":
		if len(m.batch) == 0 {
			m.setStatus("No transactions to write", statusShortDuration)
		} else {
			m.openConfirm(confirmWrite)
		}
	case "q":
		m.openConfirm(confirmQuit)
	}
	return nil
}

func (m *Model) updateTransactionView(msg tea.KeyMsg) tea.Cmd {
	if m.form.focusedField == focusDate {
		if m.handleDateKey(msg) {
			m.recalculateTotals()
			return nil
		}
	}

	switch msg.String() {
	case "ctrl+q":
		return tea.Quit
	case "ctrl+c":
		m.confirmTransaction()
		return nil
	case "esc":
		m.cancelTransaction()
		return nil
	case "shift+tab":
		m.evaluateAmountField()
		m.retreatFocus()
		return nil
	case "tab":
		if !m.tryAcceptSuggestion() {
			m.evaluateAmountField()
			m.advanceFocus()
		}
		return nil
	case "enter":
		if m.handleEnterKey() {
			return nil
		}
	case "ctrl+n":
		if m.hasActiveLine() {
			m.addLine(m.form.focusedSection, true)
		}
		return nil
	case "ctrl+d":
		if m.hasActiveLine() {
			m.deleteLine(m.form.focusedSection)
		}
		return nil
	case "b":
		if m.canBalanceCurrentLine() && m.balanceCurrentLine() {
			m.recalculateTotals()
		}
		return nil
	}

	cmd := m.updateFocusedInput(msg)
	m.refreshSuggestions()
	m.refreshTemplateOptions()
	m.recalculateTotals()
	return cmd
}

func (m *Model) handleDateKey(msg tea.KeyMsg) bool {
	switch msg.String() {
	case "left":
		m.form.date.segmentLeft()
		return true
	case "right":
		m.form.date.segmentRight()
		return true
	case "up":
		m.form.date.increment(1)
		return true
	case "down":
		m.form.date.increment(-1)
		return true
	}
	if len(msg.Runes) == 1 {
		r := msg.Runes[0]
		if r >= '0' && r <= '9' {
			m.form.date.handleDigit(r)
			return true
		}
	}
	return false
}

func (m *Model) cancelTransaction() {
	m.resetForm(m.defaultDate())
	m.currentView = viewBatch
}

func (m *Model) defaultDate() time.Time {
	if !m.lastDate.IsZero() {
		return m.lastDate
	}
	if len(m.batch) > 0 {
		return m.batch[len(m.batch)-1].Date
	}
	return time.Now()
}

func (m *Model) evaluateAmountField() {
	if m.form.focusedField != focusSectionAmount {
		return
	}
	if line := m.currentLine(); line != nil {
		m.evaluateInput(&line.amountInput)
	}
}

func (m *Model) evaluateInput(input *textinput.Model) bool {
	value := strings.TrimSpace(input.Value())
	if value == "" {
		return true
	}
	evaluated, err := util.EvaluateExpression(value)
	if err != nil {
		m.setStatus(fmt.Sprintf("Invalid expression: %v", err), statusDuration)
		return false
	}
	input.SetValue(evaluated)
	input.CursorEnd()
	return true
}

func (m *Model) advanceFocus() {
	switch m.form.focusedField {
	case focusDate:
		m.form.focusedField = focusPayee
		m.form.payeeInput.Focus()
	case focusPayee:
		m.focusTemplateButton()
		m.refreshTemplateOptions()
	case focusTemplateButton:
		m.focusSection(sectionDebit, 0, focusSectionAccount)
	case focusSectionAccount:
		if line := m.currentLine(); line != nil {
			line.accountInput.Blur()
			line.amountInput.Focus()
			m.form.focusedField = focusSectionAmount
		}
	case focusSectionAmount:
		if m.form.focusedSection == sectionDebit {
			if m.form.focusedIndex < len(m.form.debitLines)-1 {
				m.focusSection(sectionDebit, m.form.focusedIndex+1, focusSectionAccount)
			} else {
				m.focusSection(sectionCredit, 0, focusSectionAccount)
			}
		} else {
			if m.form.focusedIndex < len(m.form.creditLines)-1 {
				m.focusSection(sectionCredit, m.form.focusedIndex+1, focusSectionAccount)
			} else {
				m.addLine(sectionCredit, false)
				m.focusSection(sectionCredit, len(m.form.creditLines)-1, focusSectionAccount)
			}
		}
	}
	m.refreshSuggestions()
}

func (m *Model) retreatFocus() {
	switch m.form.focusedField {
	case focusPayee:
		m.form.payeeInput.Blur()
		m.form.focusedField = focusDate
	case focusTemplateButton:
		m.form.focusedField = focusPayee
		m.form.payeeInput.Focus()
	case focusSectionAccount:
		if m.form.focusedSection == sectionDebit {
			if m.form.focusedIndex == 0 {
				m.focusTemplateButton()
			} else {
				m.focusSection(sectionDebit, m.form.focusedIndex-1, focusSectionAmount)
			}
		} else {
			if m.form.focusedIndex == 0 {
				if len(m.form.debitLines) > 0 {
					m.focusSection(sectionDebit, len(m.form.debitLines)-1, focusSectionAmount)
				} else {
					m.form.focusedField = focusPayee
					m.form.payeeInput.Focus()
				}
			} else {
				m.focusSection(sectionCredit, m.form.focusedIndex-1, focusSectionAmount)
			}
		}
	case focusSectionAmount:
		m.focusSection(m.form.focusedSection, m.form.focusedIndex, focusSectionAccount)
	default:
		m.form.focusedField = focusDate
	}
	m.refreshSuggestions()
}

func (m *Model) focusSection(section sectionType, index int, field focusedField) {
	if section == sectionDebit {
		if index >= len(m.form.debitLines) {
			index = len(m.form.debitLines) - 1
		}
	} else {
		if index >= len(m.form.creditLines) {
			index = len(m.form.creditLines) - 1
		}
	}
	if index < 0 {
		index = 0
	}
	m.blurCurrent()
	m.form.focusedSection = section
	m.form.focusedIndex = index
	m.form.focusedField = field
	if line := m.currentLine(); line != nil {
		if field == focusSectionAccount {
			line.accountInput.Focus()
		} else {
			line.amountInput.Focus()
		}
	}
}

func (m *Model) focusTemplateButton() {
	m.blurCurrent()
	m.form.focusedField = focusTemplateButton
}

func (m *Model) blurCurrent() {
	if line := m.currentLine(); line != nil {
		line.accountInput.Blur()
		line.amountInput.Blur()
	}
	if m.form.focusedField == focusPayee {
		m.form.payeeInput.Blur()
	}
}

func (m *Model) currentLine() *postingLine {
	switch m.form.focusedSection {
	case sectionDebit:
		if m.form.focusedIndex >= 0 && m.form.focusedIndex < len(m.form.debitLines) {
			return &m.form.debitLines[m.form.focusedIndex]
		}
	case sectionCredit:
		if m.form.focusedIndex >= 0 && m.form.focusedIndex < len(m.form.creditLines) {
			return &m.form.creditLines[m.form.focusedIndex]
		}
	}
	return nil
}

func (m *Model) lineHasFocus(section sectionType, index int) bool {
	if m.form.focusedField != focusSectionAccount && m.form.focusedField != focusSectionAmount {
		return false
	}
	return m.form.focusedSection == section && m.form.focusedIndex == index
}

func (m *Model) hasActiveLine() bool {
	if m.form.focusedField != focusSectionAccount && m.form.focusedField != focusSectionAmount {
		return false
	}
	return m.currentLine() != nil
}

func (m *Model) canBalanceCurrentLine() bool {
	if m.form.focusedField != focusSectionAmount || m.form.focusedSection != sectionCredit {
		return false
	}
	line := m.currentLine()
	if line == nil {
		return false
	}
	if strings.TrimSpace(line.amountInput.Value()) != "" {
		return false
	}
	if len(m.form.debitLines)+len(m.form.creditLines) < 2 {
		return false
	}
	unfilled := 0
	for i := range m.form.debitLines {
		if strings.TrimSpace(m.form.debitLines[i].amountInput.Value()) == "" {
			unfilled++
		}
	}
	for i := range m.form.creditLines {
		if strings.TrimSpace(m.form.creditLines[i].amountInput.Value()) == "" {
			unfilled++
		}
	}
	return unfilled == 1
}

func (m *Model) handleEnterKey() bool {
	switch m.form.focusedField {
	case focusDate:
		m.advanceFocus()
		return true
	case focusPayee:
		m.advanceFocus()
		return true
	case focusTemplateButton:
		m.openTemplateSelection()
		return true
	case focusSectionAccount:
		m.advanceFocus()
		return true
	case focusSectionAmount:
		if line := m.currentLine(); line != nil {
			if m.evaluateInput(&line.amountInput) {
				m.advanceFocus()
			}
		}
		return true
	}
	return false
}

func (m *Model) updateFocusedInput(msg tea.KeyMsg) tea.Cmd {
	switch m.form.focusedField {
	case focusPayee:
		var cmd tea.Cmd
		m.form.payeeInput, cmd = m.form.payeeInput.Update(msg)
		return cmd
	case focusSectionAccount:
		if line := m.currentLine(); line != nil {
			var cmd tea.Cmd
			line.accountInput, cmd = line.accountInput.Update(msg)
			return cmd
		}
	case focusSectionAmount:
		if line := m.currentLine(); line != nil {
			var cmd tea.Cmd
			line.amountInput, cmd = line.amountInput.Update(msg)
			return cmd
		}
	}
	return nil
}

func (m *Model) tryAcceptSuggestion() bool {
	input := m.currentTextInput()
	if input == nil {
		return false
	}
	suggestion := input.CurrentSuggestion()
	if suggestion == "" {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(input.Value()), suggestion) {
		return false
	}
	input.SetValue(suggestion)
	input.CursorEnd()
	m.refreshSuggestions()
	if input == &m.form.payeeInput {
		m.templatePayee = ""
		m.refreshTemplateOptions()
	}
	return true
}

func (m *Model) currentTextInput() *textinput.Model {
	switch m.form.focusedField {
	case focusPayee:
		return &m.form.payeeInput
	case focusSectionAccount:
		if line := m.currentLine(); line != nil {
			return &line.accountInput
		}
	case focusSectionAmount:
		if line := m.currentLine(); line != nil {
			return &line.amountInput
		}
	}
	return nil
}

func (m *Model) addLine(section sectionType, cloneCategory bool) {
	var lines *[]postingLine
	switch section {
	case sectionDebit:
		lines = &m.form.debitLines
	case sectionCredit:
		lines = &m.form.creditLines
	}
	seed := ""
	if cloneCategory {
		if line := m.currentLine(); line != nil {
			seed = categorySeed(line.accountInput.Value())
		}
	}
	newLine := newPostingLine()
	if seed != "" {
		newLine.accountInput.SetValue(seed)
	}
	*lines = append(*lines, newLine)
}

func (m *Model) deleteLine(section sectionType) {
	var lines *[]postingLine
	switch section {
	case sectionDebit:
		lines = &m.form.debitLines
	case sectionCredit:
		lines = &m.form.creditLines
	}
	if lines == nil || len(*lines) <= 1 {
		m.setStatus("At least one line required", statusShortDuration)
		return
	}
	idx := m.form.focusedIndex
	if idx < 0 || idx >= len(*lines) {
		return
	}
	*lines = append((*lines)[:idx], (*lines)[idx+1:]...)
	if idx >= len(*lines) {
		idx = len(*lines) - 1
	}
	m.focusSection(section, idx, focusSectionAccount)
	m.recalculateTotals()
}

func categorySeed(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if !strings.Contains(value, ":") {
		return value + ":"
	}
	idx := strings.LastIndex(value, ":")
	if idx == len(value)-1 {
		return value
	}
	return value[:idx+1]
}

func (m *Model) balanceCurrentLine() bool {
	if m.form.focusedField != focusSectionAmount || m.form.focusedSection != sectionCredit {
		return false
	}
	if line := m.currentLine(); line != nil {
		difference := m.form.debitTotal.Sub(m.form.creditTotal.Sub(lineAmount(line)))
		if difference.IsZero() {
			return false
		}
		line.amountInput.SetValue(difference.StringFixed(2))
		line.amountInput.CursorEnd()
		return true
	}
	return false
}

func lineAmount(line *postingLine) decimal.Decimal {
	value := strings.TrimSpace(line.amountInput.Value())
	if value == "" {
		return decimal.Zero
	}
	amount, err := decimal.NewFromString(value)
	if err != nil {
		return decimal.Zero
	}
	return amount
}

func (m *Model) recalculateTotals() {
	debit := decimal.Zero
	for i := range m.form.debitLines {
		debit = debit.Add(lineAmount(&m.form.debitLines[i]))
	}
	credit := decimal.Zero
	for i := range m.form.creditLines {
		credit = credit.Add(lineAmount(&m.form.creditLines[i]))
	}
	m.form.debitTotal = debit
	m.form.creditTotal = credit
	m.form.remaining = debit.Sub(credit)
}

func (m *Model) setStatus(message string, duration time.Duration) {
	m.statusMessage = message
	m.statusExpiry = time.Now().Add(duration)
}

func (m *Model) startNewTransaction() {
	m.resetForm(m.defaultDate())
	m.currentView = viewTransaction
}

func (m *Model) startEditingTransaction(index int) {
	if index < 0 || index >= len(m.batch) {
		return
	}
	tx := m.batch[index]
	m.resetForm(tx.Date)
	m.editingIndex = index
	m.form.payeeInput.SetValue(tx.Payee)
	m.form.payeeInput.CursorEnd()
	m.refreshTemplateOptions()

	m.form.debitLines = nil
	m.form.creditLines = nil
	for _, posting := range tx.Postings {
		amount, err := decimal.NewFromString(posting.Amount)
		if err != nil {
			continue
		}
		line := newPostingLine()
		line.accountInput.SetValue(posting.Account)
		line.accountInput.CursorEnd()
		line.amountInput.SetValue(amount.Abs().StringFixed(2))
		line.amountInput.CursorEnd()
		if amount.Sign() >= 0 {
			m.form.debitLines = append(m.form.debitLines, line)
		} else {
			m.form.creditLines = append(m.form.creditLines, line)
		}
	}
	if len(m.form.debitLines) == 0 {
		m.form.debitLines = []postingLine{newPostingLine()}
	}
	if len(m.form.creditLines) == 0 {
		m.form.creditLines = []postingLine{newPostingLine()}
	}
	m.recalculateTotals()
	m.focusSection(sectionDebit, 0, focusSectionAccount)
	m.currentView = viewTransaction
}

func (m *Model) openConfirm(kind confirmKind) {
	m.pendingConfirm = kind
	m.currentView = viewConfirm
}

func (m *Model) openTemplateSelection() {
	m.refreshTemplateOptions()
	if m.templatePayee == "" {
		m.setStatus("Enter a payee to view templates", statusShortDuration)
		return
	}
	if len(m.templateOptions) == 0 {
		m.setStatus("No templates available for this payee", statusShortDuration)
		return
	}
	if m.templateCursor >= len(m.templateOptions) {
		m.templateCursor = 0
	}
	m.ensureTemplateCursorVisible()
	m.currentView = viewTemplate
}

func (m *Model) updateTemplateView(msg tea.KeyMsg) tea.Cmd {
	if len(m.templateOptions) == 0 {
		m.currentView = viewTransaction
		return nil
	}
	switch msg.String() {
	case "up", "k":
		if m.templateCursor > 0 {
			m.templateCursor--
			m.ensureTemplateCursorVisible()
		}
	case "down", "j":
		if m.templateCursor < len(m.templateOptions)-1 {
			m.templateCursor++
			m.ensureTemplateCursorVisible()
		}
	case "enter":
		m.applyTemplate(m.templateOptions[m.templateCursor])
	case "esc":
		m.skipTemplate()
	}
	return nil
}

func (m *Model) applyTemplate(record intelligence.TemplateRecord) {
	m.form.debitLines = nil
	for _, account := range record.DebitAccounts {
		line := newPostingLine()
		line.accountInput.SetValue(account)
		line.accountInput.CursorEnd()
		m.form.debitLines = append(m.form.debitLines, line)
	}
	if len(m.form.debitLines) == 0 {
		m.form.debitLines = []postingLine{newPostingLine()}
	}

	m.form.creditLines = nil
	for _, account := range record.CreditAccounts {
		line := newPostingLine()
		line.accountInput.SetValue(account)
		line.accountInput.CursorEnd()
		m.form.creditLines = append(m.form.creditLines, line)
	}
	if len(m.form.creditLines) == 0 {
		m.form.creditLines = []postingLine{newPostingLine()}
	}

	m.currentView = viewTransaction
	m.focusSection(sectionDebit, 0, focusSectionAccount)
	m.recalculateTotals()
}

func (m *Model) skipTemplate() {
	m.currentView = viewTransaction
	m.focusSection(sectionDebit, 0, focusSectionAccount)
	m.recalculateTotals()
}

func (m *Model) updateConfirmView(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "ctrl+q":
		return tea.Quit
	case "enter":
		switch m.pendingConfirm {
		case confirmWrite:
			if err := m.writeTransactionsToLedger(); err != nil {
				m.setStatus(fmt.Sprintf("Failed to write: %v", err), statusDuration)
			} else {
				count := len(m.batch)
				m.setStatus(fmt.Sprintf("Wrote %d transaction(s) to %s", count, m.ledgerFilePath), statusShortDuration)
				m.batch = nil
				m.cursor = 0
				if err := session.DeleteSession(); err != nil {
					m.setStatus(fmt.Sprintf("Ledger written but session cleanup failed: %v", err), statusDuration)
				}
			}
			m.currentView = viewBatch
		case confirmQuit:
			if err := session.DeleteSession(); err != nil {
				m.setStatus(fmt.Sprintf("Failed to clear session: %v", err), statusDuration)
			}
			return tea.Quit
		}
	case "esc":
		m.currentView = viewBatch
	}
	return nil
}

func (m *Model) confirmTransaction() bool {
	date := m.form.date.time()
	if date.IsZero() {
		m.setStatus("Invalid date", statusShortDuration)
		m.form.focusedField = focusDate
		return false
	}
	if strings.TrimSpace(m.form.payeeInput.Value()) == "" {
		m.setStatus("Payee is required", statusShortDuration)
		m.form.focusedField = focusPayee
		m.form.payeeInput.Focus()
		return false
	}

	for i := range m.form.debitLines {
		_ = m.evaluateInput(&m.form.debitLines[i].amountInput)
	}
	for i := range m.form.creditLines {
		_ = m.evaluateInput(&m.form.creditLines[i].amountInput)
	}
	m.recalculateTotals()

	if len(m.form.debitLines) == 0 || len(m.form.creditLines) == 0 {
		m.setStatus("At least one debit and credit leg required", statusShortDuration)
		return false
	}
	if m.form.debitTotal.IsZero() || m.form.creditTotal.IsZero() {
		m.setStatus("Amounts required in both sections", statusShortDuration)
		return false
	}

	difference := m.form.debitTotal.Sub(m.form.creditTotal).Abs()
	if difference.GreaterThan(decimal.NewFromFloat(balanceTolerance)) {
		m.setStatus("Debits and credits must balance", statusDuration)
		return false
	}

	postings := make([]core.Posting, 0, len(m.form.debitLines)+len(m.form.creditLines))
	for i := range m.form.debitLines {
		line := &m.form.debitLines[i]
		account := strings.TrimSpace(line.accountInput.Value())
		amount := lineAmount(line)
		if account == "" || amount.IsZero() {
			continue
		}
		postings = append(postings, core.Posting{
			Account: account,
			Amount:  amount.StringFixed(2),
		})
	}
	for i := range m.form.creditLines {
		line := &m.form.creditLines[i]
		account := strings.TrimSpace(line.accountInput.Value())
		amount := lineAmount(line)
		if account == "" || amount.IsZero() {
			continue
		}
		postings = append(postings, core.Posting{
			Account: account,
			Amount:  amount.Neg().StringFixed(2),
		})
	}

	if len(postings) < 2 {
		m.setStatus("Incomplete transaction", statusShortDuration)
		return false
	}

	tx := core.Transaction{
		Date:     date,
		Payee:    m.form.payeeInput.Value(),
		Postings: postings,
	}

	wasEdit := m.editingIndex >= 0 && m.editingIndex < len(m.batch)
	if wasEdit {
		m.batch[m.editingIndex] = tx
	} else {
		m.batch = append(m.batch, tx)
	}

	sort.SliceStable(m.batch, func(i, j int) bool {
		if m.batch[i].Date.Equal(m.batch[j].Date) {
			return m.batch[i].Payee < m.batch[j].Payee
		}
		return m.batch[i].Date.Before(m.batch[j].Date)
	})

	m.cursor = m.findTransactionIndex(tx)
	if err := session.SaveBatch(m.batch); err != nil {
		m.setStatus(fmt.Sprintf("Saved but session write failed: %v", err), statusDuration)
	} else {
		action := "added"
		if wasEdit {
			action = "updated"
		}
		m.setStatus(fmt.Sprintf("Transaction %s (%d total)", action, len(m.batch)), statusShortDuration)
	}

	m.lastDate = date
	m.resetForm(date)
	m.currentView = viewBatch
	return true
}

func (m *Model) findTransactionIndex(tx core.Transaction) int {
	for i, candidate := range m.batch {
		if candidate.Date.Equal(tx.Date) && candidate.Payee == tx.Payee && len(candidate.Postings) == len(tx.Postings) {
			match := true
			for j := range candidate.Postings {
				if candidate.Postings[j] != tx.Postings[j] {
					match = false
					break
				}
			}
			if match {
				return i
			}
		}
	}
	return len(m.batch) - 1
}

func (m *Model) renderBatchView() string {
	var b strings.Builder
	fmt.Fprintf(&b, "-- Batch Summary (%d transactions) --\n\n", len(m.batch))
	if len(m.batch) == 0 {
		b.WriteString("No transactions in current batch.\n\n")
	} else {
		for i, tx := range m.batch {
			cursor := " "
			if i == m.cursor {
				cursor = ">"
			}
			payee := tx.Payee
			if len(payee) > 28 {
				payee = payee[:25] + "..."
			}
			primary := ""
			if len(tx.Postings) > 0 {
				primary = tx.Postings[0].Account
				parts := strings.Split(primary, ":")
				primary = parts[len(parts)-1]
			}
			fmt.Fprintf(&b, "%s %s %-28s (%s)\n", cursor, tx.Date.Format("2006-01-02"), payee, primary)
		}
		b.WriteString("\n")
	}
	if msg := m.statusLine(); msg != "" {
		fmt.Fprintf(&b, "%s\n\n", msg)
	}
	b.WriteString("[n]ew  [e]dit  [w]rite  [q]uit  [enter]edit selected")
	return b.String()
}

func (m *Model) renderTransactionView() string {
	var b strings.Builder
	fmt.Fprintf(&b, "-- Transaction Entry -- Remaining: $%s --\n\n", m.form.remaining.StringFixed(2))

	fmt.Fprintf(&b, "Date    %s\n", m.form.date.display(m.form.focusedField == focusDate))
	fmt.Fprintf(&b, "Payee   %s", m.form.payeeInput.View())
	if m.form.focusedField == focusPayee {
		b.WriteString(renderSuggestionList(m.form.payeeInput))
	}
	b.WriteString("\n")
	buttonCursor := " "
	if m.form.focusedField == focusTemplateButton {
		buttonCursor = ">"
	}
	fmt.Fprintf(&b, "        %s[%s]\n\n", buttonCursor, templateAvailabilityLabel(len(m.templateOptions)))

	fmt.Fprintf(&b, "Debits   (total %s)\n", m.form.debitTotal.StringFixed(2))
	for i, line := range m.form.debitLines {
		cursor := " "
		if m.lineHasFocus(sectionDebit, i) {
			cursor = ">"
		}
		fmt.Fprintf(&b, "%s [%s] [%s]", cursor, line.accountInput.View(), line.amountInput.View())
		if m.lineHasFocus(sectionDebit, i) && m.form.focusedField == focusSectionAccount {
			b.WriteString(renderSuggestionList(line.accountInput))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")

	fmt.Fprintf(&b, "Credits  (total %s)\n", m.form.creditTotal.StringFixed(2))
	for i, line := range m.form.creditLines {
		cursor := " "
		if m.lineHasFocus(sectionCredit, i) {
			cursor = ">"
		}
		fmt.Fprintf(&b, "%s [%s] [%s]", cursor, line.accountInput.View(), line.amountInput.View())
		if m.lineHasFocus(sectionCredit, i) && m.form.focusedField == focusSectionAccount {
			b.WriteString(renderSuggestionList(line.accountInput))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")

	if msg := m.statusLine(); msg != "" {
		fmt.Fprintf(&b, "%s\n\n", msg)
	}

	commands := []string{"[tab]next", "[shift+tab]prev"}
	if m.hasActiveLine() {
		commands = append(commands, "[ctrl+n]add line", "[ctrl+d]delete line")
	}
	if m.canBalanceCurrentLine() {
		commands = append(commands, "[b]alance")
	}
	commands = append(commands, "[ctrl+c]confirm", "[esc]cancel", "[ctrl+q]quit")
	b.WriteString(strings.Join(commands, "\n"))
	return b.String()
}

func templateAvailabilityLabel(count int) string {
	if count == 1 {
		return "1 template available"
	}
	return fmt.Sprintf("%d templates available", count)
}

func (m *Model) renderTemplateView() string {
	var b strings.Builder
	fmt.Fprintf(&b, "-- Templates for %s --\n\n", m.form.payeeInput.Value())
	if len(m.templateOptions) == 0 {
		b.WriteString("No templates available\n\n[esc]skip")
		return b.String()
	}
	start := m.templateOffset
	if start < 0 {
		start = 0
	}
	if start >= len(m.templateOptions) {
		start = len(m.templateOptions) - 1
	}
	end := start + maxTemplateDisplay
	if end > len(m.templateOptions) {
		end = len(m.templateOptions)
	}
	for i := start; i < end; i++ {
		tpl := m.templateOptions[i]
		cursor := " "
		if i == m.templateCursor {
			cursor = ">"
		}
		usageLabel := "times"
		if tpl.Frequency == 1 {
			usageLabel = "time"
		}
		fmt.Fprintf(&b, "%s %d. Used %d %s\n", cursor, i+1, tpl.Frequency, usageLabel)
		b.WriteString("    Debit Accounts:\n")
		if len(tpl.DebitAccounts) == 0 {
			b.WriteString("      (none)\n")
		} else {
			for _, account := range tpl.DebitAccounts {
				fmt.Fprintf(&b, "      %s\n", account)
			}
		}
		b.WriteString("    Credit Accounts:\n")
		if len(tpl.CreditAccounts) == 0 {
			b.WriteString("      (none)\n")
		} else {
			for _, account := range tpl.CreditAccounts {
				fmt.Fprintf(&b, "      %s\n", account)
			}
		}
		if i < end-1 {
			b.WriteString("\n")
		}
	}
	b.WriteString("\n[enter]apply\n[esc]skip")
	return b.String()
}

func (m *Model) renderConfirmView() string {
	var b strings.Builder
	switch m.pendingConfirm {
	case confirmWrite:
		fmt.Fprintf(&b, "Write %d transaction(s) to %s?\n\n", len(m.batch), m.ledgerFilePath)
	case confirmQuit:
		if len(m.batch) > 0 {
			fmt.Fprintf(&b, "Quit without writing %d pending transaction(s)?\n\n", len(m.batch))
		} else {
			b.WriteString("Quit the application?\n\n")
		}
	}
	b.WriteString("[enter]confirm  [esc]cancel  [ctrl+q]quit immediately")
	return b.String()
}

func (m *Model) statusLine() string {
	if m.statusMessage == "" {
		return ""
	}
	if !m.statusExpiry.IsZero() && time.Now().After(m.statusExpiry) {
		return ""
	}
	return m.statusMessage
}

func renderSuggestionList(input textinput.Model) string {
	matches := input.MatchedSuggestions()
	if len(matches) <= 1 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n")
	display := len(matches)
	if display > maxSuggestionDisplay {
		display = maxSuggestionDisplay
	}
	for i := 0; i < display; i++ {
		cursor := " "
		if i == input.CurrentSuggestionIndex() {
			cursor = ">"
		}
		fmt.Fprintf(&b, "      %s %s\n", cursor, matches[i])
	}
	if len(matches) > display {
		fmt.Fprintf(&b, "      ... and %d more\n", len(matches)-display)
	}
	return b.String()
}

func (m *Model) refreshSuggestions() {
	switch m.form.focusedField {
	case focusPayee:
		m.form.payeeInput.SetSuggestions(m.db.FindPayees(m.form.payeeInput.Value()))
	case focusSectionAccount:
		if line := m.currentLine(); line != nil {
			line.accountInput.SetSuggestions(m.accountSuggestions(line.accountInput.Value()))
		}
	case focusSectionAmount:
		if line := m.currentLine(); line != nil {
			line.amountInput.SetSuggestions(nil)
		}
	}
}

func (m *Model) refreshTemplateOptions() {
	payee := strings.TrimSpace(m.form.payeeInput.Value())
	if payee == m.templatePayee {
		return
	}
	m.templatePayee = payee
	if payee == "" {
		m.templateOptions = nil
		m.templateCursor = 0
		m.templateOffset = 0
		return
	}
	m.templateOptions = m.db.FindTemplates(payee)
	m.templateCursor = 0
	m.templateOffset = 0
}

func (m *Model) ensureTemplateCursorVisible() {
	if len(m.templateOptions) == 0 {
		m.templateOffset = 0
		return
	}
	if m.templateCursor < 0 {
		m.templateCursor = 0
	}
	if m.templateCursor >= len(m.templateOptions) {
		m.templateCursor = len(m.templateOptions) - 1
	}
	if m.templateCursor < m.templateOffset {
		m.templateOffset = m.templateCursor
	}
	visible := maxTemplateDisplay
	if visible <= 0 {
		visible = 1
	}
	if m.templateCursor >= m.templateOffset+visible {
		m.templateOffset = m.templateCursor - visible + 1
	}
	maxOffset := len(m.templateOptions) - visible
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.templateOffset > maxOffset {
		m.templateOffset = maxOffset
	}
	if m.templateOffset < 0 {
		m.templateOffset = 0
	}
}

func (m *Model) accountSuggestions(prefix string) []string {
	raw := m.db.FindAccounts(prefix)
	if len(raw) == 0 {
		return nil
	}
	seen := make(map[string]struct{})
	suggestions := make([]string, 0, len(raw))
	for _, account := range raw {
		suggestion := nextHierarchicalSuggestion(prefix, account)
		if suggestion == "" {
			continue
		}
		key := strings.ToLower(suggestion)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		suggestions = append(suggestions, suggestion)
	}
	sort.Strings(suggestions)
	return suggestions
}

func nextHierarchicalSuggestion(prefix, account string) string {
	if prefix == "" {
		parts := strings.Split(account, ":")
		if len(parts) > 0 {
			return parts[0]
		}
		return account
	}
	lowerPrefix := strings.ToLower(prefix)
	lowerAccount := strings.ToLower(account)
	if !strings.HasPrefix(lowerAccount, lowerPrefix) {
		return ""
	}
	if len(account) == len(prefix) {
		return account
	}
	if strings.HasSuffix(prefix, ":") {
		remainder := account[len(prefix):]
		idx := strings.IndexRune(remainder, ':')
		if idx == -1 {
			return account
		}
		return account[:len(prefix)+idx]
	}
	prefixSegments := strings.Split(prefix, ":")
	accountSegments := strings.Split(account, ":")
	if len(prefixSegments) > len(accountSegments) {
		return account
	}
	for i := 0; i < len(prefixSegments)-1; i++ {
		if !strings.EqualFold(prefixSegments[i], accountSegments[i]) {
			return ""
		}
	}
	lastPrefix := prefixSegments[len(prefixSegments)-1]
	accountSegment := accountSegments[len(prefixSegments)-1]
	if !strings.HasPrefix(strings.ToLower(accountSegment), strings.ToLower(lastPrefix)) {
		return ""
	}
	return strings.Join(accountSegments[:len(prefixSegments)], ":")
}

func (m *Model) writeTransactionsToLedger() error {
	if len(m.batch) == 0 {
		return fmt.Errorf("no transactions to write")
	}
	file, err := os.OpenFile(m.ledgerFilePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open ledger: %w", err)
	}
	defer file.Close()
	for _, tx := range m.batch {
		if _, err := file.WriteString("\n"); err != nil {
			return fmt.Errorf("write separator: %w", err)
		}
		if _, err := file.WriteString(tx.String()); err != nil {
			return fmt.Errorf("write transaction: %w", err)
		}
	}
	return nil
}

func (m *Model) textInputFocused() bool {
	switch m.form.focusedField {
	case focusPayee:
		return m.form.payeeInput.Focused()
	case focusSectionAccount:
		if line := m.currentLine(); line != nil {
			return line.accountInput.Focused()
		}
	case focusSectionAmount:
		if line := m.currentLine(); line != nil {
			return line.amountInput.Focused()
		}
	}
	return false
}

func (m *Model) refreshAfterLoad() {
	m.recalculateTotals()
	if m.currentView == viewTransaction {
		m.refreshSuggestions()
	}
}
