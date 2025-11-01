package tui

import "strings"

// captureFormBaseline records the current form state for future dirty checks
func (m *Model) captureFormBaseline() {
	m.formBaseline = m.currentFormSnapshot()
}

// formIsDirty reports whether the current form differs from the captured baseline
func (m *Model) formIsDirty() bool {
	return !m.currentFormSnapshot().equals(m.formBaseline)
}

// currentFormSnapshot captures the essential fields of the form for comparison
func (m *Model) currentFormSnapshot() formSnapshot {
	snapshot := formSnapshot{
		date:    m.form.date.time(),
		cleared: m.form.cleared,
		payee:   strings.TrimSpace(m.form.payeeInput.Value()),
		comment: strings.TrimSpace(m.form.commentInput.Value()),
		debit:   make([]postingSnapshot, len(m.form.debitLines)),
		credit:  make([]postingSnapshot, len(m.form.creditLines)),
	}

	for i := range m.form.debitLines {
		line := &m.form.debitLines[i]
		snapshot.debit[i] = postingSnapshot{
			account: strings.TrimSpace(line.accountInput.Value()),
			amount:  strings.TrimSpace(line.amountInput.Value()),
			comment: strings.TrimSpace(line.commentInput.Value()),
		}
	}
	for i := range m.form.creditLines {
		line := &m.form.creditLines[i]
		snapshot.credit[i] = postingSnapshot{
			account: strings.TrimSpace(line.accountInput.Value()),
			amount:  strings.TrimSpace(line.amountInput.Value()),
			comment: strings.TrimSpace(line.commentInput.Value()),
		}
	}
	return snapshot
}

// equals compares two form snapshots for equality
func (a formSnapshot) equals(b formSnapshot) bool {
	if !a.date.Equal(b.date) {
		return false
	}
	if a.cleared != b.cleared {
		return false
	}
	if a.payee != b.payee || a.comment != b.comment {
		return false
	}
	if len(a.debit) != len(b.debit) || len(a.credit) != len(b.credit) {
		return false
	}
	for i := range a.debit {
		if a.debit[i] != b.debit[i] {
			return false
		}
	}
	for i := range a.credit {
		if a.credit[i] != b.credit[i] {
			return false
		}
	}
	return true
}
