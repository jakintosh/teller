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
)

type viewState int

type confirmKind int

type focusedField int

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
	focusTotal
	focusPrimaryAccount
	focusPostingAccount
	focusPostingAmount
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
	pendingConfirm  confirmKind
	editingIndex    int

	lastDate      time.Time
	lastPrimary   string
	statusMessage string
	statusExpiry  time.Time
	err           error
}

type transactionForm struct {
	date            dateField
	payeeInput      textinput.Model
	totalInput      textinput.Model
	primaryInput    textinput.Model
	postings        []postingLine
	focusedField    focusedField
	focusedPosting  int
	remaining       decimal.Decimal
	headerConfirmed bool
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
	m.resetForm(time.Now(), "")
	return m
}

func (m *Model) SetBatch(batch []core.Transaction) {
	m.batch = append([]core.Transaction(nil), batch...)
	sort.Slice(m.batch, func(i, j int) bool {
		return m.batch[i].Date.Before(m.batch[j].Date)
	})
	if len(m.batch) == 0 {
		m.cursor = 0
		m.lastDate = time.Time{}
		m.lastPrimary = ""
		return
	}
	m.cursor = len(m.batch) - 1
	last := m.batch[m.cursor]
	m.lastDate = last.Date
	if len(last.Postings) > 0 {
		m.lastPrimary = last.Postings[0].Account
	}
}

func (m *Model) resetForm(baseDate time.Time, primaryDefault string) {
	m.form = newTransactionForm(baseDate, primaryDefault)
	m.templateOptions = nil
	m.templateCursor = 0
	m.editingIndex = -1
}

func newTransactionForm(baseDate time.Time, primaryDefault string) transactionForm {
	if baseDate.IsZero() {
		baseDate = time.Now()
	}

	date := dateField{}
	date.setTime(baseDate)

	payee := newTextInput("Payee")
	total := newTextInput("Total")
	primary := newTextInput("Primary Account")
	if primaryDefault != "" {
		primary.SetValue(primaryDefault)
	}

	postings := []postingLine{newPostingLine("")}

	return transactionForm{
		date:           date,
		payeeInput:     payee,
		totalInput:     total,
		primaryInput:   primary,
		postings:       postings,
		focusedField:   focusDate,
		focusedPosting: 0,
		remaining:      decimal.Zero,
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

func newPostingLine(seed string) postingLine {
	account := newTextInput("Account")
	if seed != "" {
		account.SetValue(seed)
	}
	amount := newTextInput("Amount")
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
		return fmt.Sprintf("Error: %v\n\nPress ctrl+c to quit.", m.err)
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
	case "ctrl+c":
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
	// Date-specific handling before global shortcuts
	if m.form.focusedField == focusDate {
		if m.handleDateKey(msg) {
			m.recalculateRemaining()
			return nil
		}
	}

	switch msg.String() {
	case "ctrl+c":
		return tea.Quit
	case "esc":
		m.cancelTransaction()
		return nil
	case "c":
		if !m.textInputFocused() {
			m.confirmTransaction()
			return nil
		}
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
	case "ctrl+n":
		m.addPostingLine(true)
		return nil
	case "b":
		if m.balanceCurrentPosting() {
			m.recalculateRemaining()
		}
		return nil
	case "enter":
		if m.handleEnterKey() {
			return nil
		}
	}

	cmd := m.updateFocusedInput(msg)
	m.refreshSuggestions()
	m.recalculateRemaining()
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
	m.resetForm(m.defaultDate(), m.lastPrimary)
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
	switch m.form.focusedField {
	case focusTotal:
		m.evaluateInput(&m.form.totalInput)
	case focusPostingAmount:
		if m.form.focusedPosting >= 0 && m.form.focusedPosting < len(m.form.postings) {
			posting := &m.form.postings[m.form.focusedPosting]
			m.evaluateInput(&posting.amountInput)
		}
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
		m.form.payeeInput.Blur()
		m.form.focusedField = focusTotal
		m.form.totalInput.Focus()
	case focusTotal:
		m.form.totalInput.Blur()
		m.form.focusedField = focusPrimaryAccount
		m.form.primaryInput.Focus()
	case focusPrimaryAccount:
		if m.finalizeHeader(false) {
			return
		}
	case focusPostingAccount:
		posting := &m.form.postings[m.form.focusedPosting]
		posting.accountInput.Blur()
		m.form.focusedField = focusPostingAmount
		posting.amountInput.Focus()
	case focusPostingAmount:
		posting := &m.form.postings[m.form.focusedPosting]
		posting.amountInput.Blur()
		if m.form.focusedPosting < len(m.form.postings)-1 {
			m.form.focusedPosting++
		} else {
			m.addPostingLine(false)
			m.form.focusedPosting = len(m.form.postings) - 1
		}
		m.form.focusedField = focusPostingAccount
		m.form.postings[m.form.focusedPosting].accountInput.Focus()
	}
	m.refreshSuggestions()
}

func (m *Model) retreatFocus() {
	switch m.form.focusedField {
	case focusPayee:
		m.form.payeeInput.Blur()
		m.form.focusedField = focusDate
	case focusTotal:
		m.form.totalInput.Blur()
		m.form.focusedField = focusPayee
		m.form.payeeInput.Focus()
	case focusPrimaryAccount:
		m.form.primaryInput.Blur()
		m.form.focusedField = focusTotal
		m.form.totalInput.Focus()
	case focusPostingAccount:
		if m.form.focusedPosting == 0 {
			m.form.focusedField = focusPrimaryAccount
			m.form.primaryInput.Focus()
		} else {
			current := &m.form.postings[m.form.focusedPosting]
			current.accountInput.Blur()
			m.form.focusedPosting--
			m.form.focusedField = focusPostingAmount
			m.form.postings[m.form.focusedPosting].amountInput.Focus()
		}
	case focusPostingAmount:
		posting := &m.form.postings[m.form.focusedPosting]
		posting.amountInput.Blur()
		m.form.focusedField = focusPostingAccount
		posting.accountInput.Focus()
	default:
		m.form.focusedField = focusDate
	}
	m.refreshSuggestions()
}

func (m *Model) finalizeHeader(fromEnter bool) bool {
	if m.form.payeeInput.Value() == "" {
		m.setStatus("Payee is required", statusShortDuration)
		m.form.payeeInput.Focus()
		m.form.focusedField = focusPayee
		return false
	}

	if !m.evaluateInput(&m.form.totalInput) {
		m.form.focusedField = focusTotal
		m.form.totalInput.Focus()
		return false
	}
	if strings.TrimSpace(m.form.totalInput.Value()) == "" {
		m.setStatus("Total amount is required", statusShortDuration)
		m.form.focusedField = focusTotal
		m.form.totalInput.Focus()
		return false
	}

	if strings.TrimSpace(m.form.primaryInput.Value()) == "" {
		m.setStatus("Primary account is required", statusShortDuration)
		m.form.focusedField = focusPrimaryAccount
		m.form.primaryInput.Focus()
		return false
	}

	m.form.primaryInput.Blur()
	m.form.headerConfirmed = true

	payee := m.form.payeeInput.Value()
	m.templateOptions = m.db.FindTemplates(payee)
	if len(m.templateOptions) > 0 {
		m.templateCursor = 0
		m.currentView = viewTemplate
		return true
	}

	if len(m.form.postings) == 0 {
		m.addPostingLine(false)
	}
	m.form.focusedField = focusPostingAccount
	m.form.focusedPosting = 0
	m.form.postings[0].accountInput.Focus()
	m.refreshSuggestions()
	m.recalculateRemaining()
	return true
}

func (m *Model) currentTextInput() *textinput.Model {
	switch m.form.focusedField {
	case focusPayee:
		return &m.form.payeeInput
	case focusTotal:
		return &m.form.totalInput
	case focusPrimaryAccount:
		return &m.form.primaryInput
	case focusPostingAccount:
		if m.form.focusedPosting >= 0 && m.form.focusedPosting < len(m.form.postings) {
			return &m.form.postings[m.form.focusedPosting].accountInput
		}
	case focusPostingAmount:
		if m.form.focusedPosting >= 0 && m.form.focusedPosting < len(m.form.postings) {
			return &m.form.postings[m.form.focusedPosting].amountInput
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
	if m.form.focusedField == focusPayee {
		// Refresh template candidates immediately when payee chosen
		m.templateOptions = m.db.FindTemplates(suggestion)
	}
	return true
}

func (m *Model) addPostingLine(cloneCategory bool) {
	seed := ""
	if cloneCategory && m.form.focusedPosting >= 0 && m.form.focusedPosting < len(m.form.postings) {
		seed = categorySeed(m.form.postings[m.form.focusedPosting].accountInput.Value())
	}
	m.form.postings = append(m.form.postings, newPostingLine(seed))
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

func (m *Model) balanceCurrentPosting() bool {
	if m.form.focusedField != focusPostingAmount {
		return false
	}
	emptyIndex := -1
	for i, posting := range m.form.postings {
		if strings.TrimSpace(posting.amountInput.Value()) == "" {
			if emptyIndex != -1 {
				return false
			}
			emptyIndex = i
		}
	}
	if emptyIndex == -1 || emptyIndex != m.form.focusedPosting {
		return false
	}
	if m.form.remaining.IsZero() {
		return false
	}
	posting := &m.form.postings[m.form.focusedPosting]
	posting.amountInput.SetValue(m.form.remaining.StringFixed(2))
	posting.amountInput.CursorEnd()
	return true
}

func (m *Model) handleEnterKey() bool {
	switch m.form.focusedField {
	case focusDate:
		m.advanceFocus()
		return true
	case focusPayee:
		m.advanceFocus()
		return true
	case focusTotal:
		if m.evaluateInput(&m.form.totalInput) {
			m.advanceFocus()
		}
		return true
	case focusPrimaryAccount:
		m.finalizeHeader(true)
		return true
	case focusPostingAccount:
		m.advanceFocus()
		return true
	case focusPostingAmount:
		if m.evaluateInput(&m.form.postings[m.form.focusedPosting].amountInput) {
			m.advanceFocus()
		}
		return true
	}
	return false
}

func (m *Model) updateFocusedInput(msg tea.KeyMsg) tea.Cmd {
	input := m.currentTextInput()
	if input == nil {
		return nil
	}
	updated, cmd := input.Update(msg)
	*input = updated
	return cmd
}

func (m *Model) refreshSuggestions() {
	switch m.form.focusedField {
	case focusPayee:
		m.form.payeeInput.SetSuggestions(m.db.FindPayees(m.form.payeeInput.Value()))
	case focusPrimaryAccount:
		m.form.primaryInput.SetSuggestions(m.accountSuggestions(m.form.primaryInput.Value()))
	case focusPostingAccount:
		if m.form.focusedPosting >= 0 && m.form.focusedPosting < len(m.form.postings) {
			posting := &m.form.postings[m.form.focusedPosting]
			posting.accountInput.SetSuggestions(m.accountSuggestions(posting.accountInput.Value()))
		}
	case focusPostingAmount:
		if m.form.focusedPosting >= 0 && m.form.focusedPosting < len(m.form.postings) {
			posting := &m.form.postings[m.form.focusedPosting]
			posting.amountInput.SetSuggestions(nil)
		}
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
		segments := strings.Split(account, ":")
		if len(segments) > 0 {
			return segments[0]
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

func (m *Model) recalculateRemaining() {
	total := decimal.Zero
	totalStr := strings.TrimSpace(m.form.totalInput.Value())
	if totalStr != "" {
		if value, err := decimal.NewFromString(totalStr); err == nil {
			total = value
		}
	}

	allocated := decimal.Zero
	for _, posting := range m.form.postings {
		amountStr := strings.TrimSpace(posting.amountInput.Value())
		if amountStr == "" {
			continue
		}
		if value, err := decimal.NewFromString(amountStr); err == nil {
			allocated = allocated.Add(value)
		}
	}

	remaining := total.Sub(allocated)

	if m.form.headerConfirmed && len(m.form.postings) > 0 {
		first := &m.form.postings[0]
		if strings.TrimSpace(first.amountInput.Value()) == "" && !total.IsZero() {
			first.amountInput.SetValue(remaining.StringFixed(2))
			first.amountInput.CursorEnd()
			allocated = allocated.Add(remaining)
			remaining = total.Sub(allocated)
		}
	}

	m.form.remaining = remaining
}
func (m *Model) setStatus(message string, duration time.Duration) {
	m.statusMessage = message
	m.statusExpiry = time.Now().Add(duration)
}

func (m *Model) startNewTransaction() {
	m.resetForm(m.defaultDate(), m.lastPrimary)
	m.currentView = viewTransaction
}

func (m *Model) startEditingTransaction(index int) {
	if index < 0 || index >= len(m.batch) {
		return
	}
	tx := m.batch[index]
	primary := ""
	if len(tx.Postings) > 0 {
		primary = tx.Postings[0].Account
	}
	m.resetForm(tx.Date, primary)
	m.editingIndex = index
	m.form.headerConfirmed = true
	m.form.payeeInput.SetValue(tx.Payee)
	m.form.payeeInput.CursorEnd()

	if len(tx.Postings) > 0 {
		total, err := decimal.NewFromString(tx.Postings[0].Amount)
		if err == nil {
			m.form.totalInput.SetValue(total.Abs().StringFixed(2))
		}
	}
	m.form.totalInput.CursorEnd()
	m.form.primaryInput.CursorEnd()

	// Build postings excluding primary
	m.form.postings = nil
	for i, posting := range tx.Postings {
		if i == 0 {
			continue
		}
		line := newPostingLine("")
		line.accountInput.SetValue(posting.Account)
		line.accountInput.CursorEnd()
		line.amountInput.SetValue(posting.Amount)
		line.amountInput.CursorEnd()
		m.form.postings = append(m.form.postings, line)
	}
	if len(m.form.postings) == 0 {
		m.form.postings = []postingLine{newPostingLine("")}
	}

	m.recalculateRemaining()
	m.form.focusedField = focusDate
	m.form.focusedPosting = 0
	m.currentView = viewTransaction
	m.refreshSuggestions()
}

func (m *Model) openConfirm(kind confirmKind) {
	m.pendingConfirm = kind
	m.currentView = viewConfirm
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
		}
	case "down", "j":
		if m.templateCursor < len(m.templateOptions)-1 {
			m.templateCursor++
		}
	case "enter":
		m.applyTemplate(m.templateOptions[m.templateCursor])
		return nil
	case "esc":
		m.skipTemplate()
	}
	return nil
}

func (m *Model) applyTemplate(record intelligence.TemplateRecord) {
	primary := strings.TrimSpace(m.form.primaryInput.Value())
	postings := make([]postingLine, 0, len(record.Accounts))
	for _, account := range record.Accounts {
		if strings.EqualFold(strings.TrimSpace(account), primary) {
			continue
		}
		line := newPostingLine("")
		line.accountInput.SetValue(account)
		line.accountInput.CursorEnd()
		postings = append(postings, line)
	}
	if len(postings) == 0 {
		postings = []postingLine{newPostingLine("")}
	}
	m.form.postings = postings
	m.templateOptions = nil
	m.currentView = viewTransaction
	m.form.focusedField = focusPostingAccount
	m.form.focusedPosting = 0
	m.form.postings[0].accountInput.Focus()
	m.refreshSuggestions()
	m.recalculateRemaining()
}

func (m *Model) skipTemplate() {
	m.templateOptions = nil
	if len(m.form.postings) == 0 {
		m.addPostingLine(false)
	}
	m.currentView = viewTransaction
	m.form.focusedField = focusPostingAccount
	m.form.focusedPosting = 0
	m.form.postings[0].accountInput.Focus()
	m.refreshSuggestions()
	m.recalculateRemaining()
}

func (m *Model) updateConfirmView(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "ctrl+c":
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
					m.setStatus(fmt.Sprintf("Ledger written but failed to clear session: %v", err), statusDuration)
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
	if !m.form.headerConfirmed {
		if !m.finalizeHeader(false) {
			return false
		}
	}

	date := m.form.date.time()
	if date.IsZero() {
		m.setStatus("Invalid date", statusShortDuration)
		m.form.focusedField = focusDate
		return false
	}

	if !m.evaluateInput(&m.form.totalInput) {
		m.form.focusedField = focusTotal
		return false
	}

	totalValue, err := decimal.NewFromString(strings.TrimSpace(m.form.totalInput.Value()))
	if err != nil {
		m.setStatus("Total amount is invalid", statusShortDuration)
		m.form.focusedField = focusTotal
		return false
	}

	primaryAccount := strings.TrimSpace(m.form.primaryInput.Value())
	if primaryAccount == "" {
		m.setStatus("Primary account is required", statusShortDuration)
		m.form.focusedField = focusPrimaryAccount
		return false
	}

	// Evaluate all posting amount inputs before validation
	for i := range m.form.postings {
		if !m.evaluateInput(&m.form.postings[i].amountInput) {
			m.form.focusedField = focusPostingAmount
			m.form.focusedPosting = i
			return false
		}
	}

	// Build postings
	postings := make([]core.Posting, 0, len(m.form.postings)+1)
	postings = append(postings, core.Posting{
		Account: primaryAccount,
		Amount:  totalValue.Neg().StringFixed(2),
	})

	var allocationCount int
	allocated := decimal.Zero

	for i, line := range m.form.postings {
		account := strings.TrimSpace(line.accountInput.Value())
		amountStr := strings.TrimSpace(line.amountInput.Value())
		if account == "" && amountStr == "" {
			continue
		}
		if account == "" {
			m.setStatus(fmt.Sprintf("Posting %d is missing an account", i+1), statusShortDuration)
			m.form.focusedField = focusPostingAccount
			m.form.focusedPosting = i
			m.form.postings[i].accountInput.Focus()
			return false
		}
		if amountStr == "" {
			m.setStatus(fmt.Sprintf("Posting %d is missing an amount", i+1), statusShortDuration)
			m.form.focusedField = focusPostingAmount
			m.form.focusedPosting = i
			m.form.postings[i].amountInput.Focus()
			return false
		}
		amountValue, err := decimal.NewFromString(amountStr)
		if err != nil {
			m.setStatus(fmt.Sprintf("Posting %d amount is invalid", i+1), statusShortDuration)
			m.form.focusedField = focusPostingAmount
			m.form.focusedPosting = i
			return false
		}
		allocated = allocated.Add(amountValue)
		postings = append(postings, core.Posting{Account: account, Amount: amountValue.StringFixed(2)})
		allocationCount++
	}

	if allocationCount == 0 {
		m.setStatus("At least one allocation is required", statusShortDuration)
		return false
	}

	remaining := totalValue.Sub(allocated)
	if remaining.Abs().GreaterThan(decimal.NewFromFloat(0.01)) {
		m.setStatus("Allocations must balance with total", statusDuration)
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
		m.editingIndex = len(m.batch) - 1
	}

	sort.SliceStable(m.batch, func(i, j int) bool {
		if m.batch[i].Date.Equal(m.batch[j].Date) {
			return m.batch[i].Payee < m.batch[j].Payee
		}
		return m.batch[i].Date.Before(m.batch[j].Date)
	})

	// Update cursor to edited/added transaction
	m.cursor = m.findTransactionIndex(tx)

	if err := session.SaveBatch(m.batch); err != nil {
		m.setStatus(fmt.Sprintf("Saved transaction but failed to persist session: %v", err), statusDuration)
	} else {
		action := "added"
		if wasEdit {
			action = "updated"
		}
		m.setStatus(fmt.Sprintf("Transaction %s (%d total)", action, len(m.batch)), statusShortDuration)
	}

	m.lastDate = date
	m.lastPrimary = primaryAccount
	m.currentView = viewBatch
	m.resetForm(date, primaryAccount)
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
func (m *Model) textInputFocused() bool {
	input := m.currentTextInput()
	if input == nil {
		return false
	}
	return input.Focused()
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

	fmt.Fprintf(&b, "Total   %s\n", m.form.totalInput.View())

	fmt.Fprintf(&b, "From    %s", m.form.primaryInput.View())
	if m.form.focusedField == focusPrimaryAccount {
		b.WriteString(renderSuggestionList(m.form.primaryInput))
	}
	b.WriteString("\n\n")

	if len(m.form.postings) > 0 {
		b.WriteString("Allocations:\n")
		for i, posting := range m.form.postings {
			cursor := " "
			if m.form.focusedPosting == i && (m.form.focusedField == focusPostingAccount || m.form.focusedField == focusPostingAmount) {
				cursor = ">"
			}
			fmt.Fprintf(&b, "%s [%s] [%s]", cursor, posting.accountInput.View(), posting.amountInput.View())
			if m.form.focusedPosting == i && m.form.focusedField == focusPostingAccount {
				b.WriteString(renderSuggestionList(posting.accountInput))
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	if msg := m.statusLine(); msg != "" {
		fmt.Fprintf(&b, "%s\n\n", msg)
	}

	help := "[tab]next [shift+tab]prev [ctrl+n]add leg [b]alance [esc]cancel"
	if !m.textInputFocused() {
		help = help + " [c]onfirm"
	}
	b.WriteString(help)
	return b.String()
}

func (m *Model) renderTemplateView() string {
	var b strings.Builder
	payee := m.form.payeeInput.Value()
	fmt.Fprintf(&b, "-- Templates for %s --\n\n", payee)

	for i, template := range m.templateOptions {
		cursor := " "
		if i == m.templateCursor {
			cursor = ">"
		}
		fmt.Fprintf(&b, "%s %d. %s (used %d times)\n", cursor, i+1, strings.Join(template.Accounts, ", "), template.Frequency)
	}

	b.WriteString("\n[enter]apply  [esc]skip")
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
	b.WriteString("[enter]confirm  [esc]cancel")
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
	if len(matches) == 0 {
		return ""
	}
	if len(matches) == 1 {
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
