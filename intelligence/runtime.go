package intelligence

import (
	"sort"
	"strings"

	"git.sr.ht/~jakintosh/teller/core"
	"github.com/shopspring/decimal"
)

// RuntimeIntelligence stores intelligence extracted from transactions in the current batch.
// This is separate from the base IntelligenceDB to clearly distinguish between
// ledger-based and runtime-added data.
type RuntimeIntelligence struct {
	Payees    []string
	Accounts  *Trie
	Templates map[string][]TemplateRecord
}

// NewRuntimeIntelligence creates an empty runtime intelligence database.
func NewRuntimeIntelligence() *RuntimeIntelligence {
	return &RuntimeIntelligence{
		Payees:    []string{},
		Accounts:  NewTrie(),
		Templates: make(map[string][]TemplateRecord),
	}
}

// BuildFromBatch creates a new runtime intelligence database from the given batch of transactions.
// This rebuilds the entire runtime database from scratch, ensuring it only contains data
// that is actually present in the current batch. This approach is simple and guarantees correctness.
func (r *RuntimeIntelligence) BuildFromBatch(transactions []core.Transaction) {
	// Create a fresh instance
	*r = RuntimeIntelligence{
		Accounts:  NewTrie(),
		Templates: make(map[string][]TemplateRecord),
	}

	// Extract unique payees
	payeeSet := make(map[string]bool)
	// Extract unique accounts
	accountSet := make(map[string]bool)

	for _, tx := range transactions {
		if tx.Payee != "" {
			payeeSet[tx.Payee] = true
		}

		// Process all postings to extract account names
		for _, posting := range tx.Postings {
			if posting.Account != "" {
				accountSet[posting.Account] = true
			}
		}
	}

	// Convert payees to sorted slice
	r.Payees = make([]string, 0, len(payeeSet))
	for payee := range payeeSet {
		r.Payees = append(r.Payees, payee)
	}
	sort.Strings(r.Payees)

	// Insert all accounts into the Trie
	for account := range accountSet {
		r.Accounts.Insert(account)
	}

	// Analyze transaction templates (same logic as in NewIntelligenceDB)
	templateFreq := make(map[string]map[string]templateBucket) // payee -> key -> bucket

	for _, tx := range transactions {
		if tx.Payee == "" || len(tx.Postings) == 0 {
			continue
		}

		var debitAccounts []string
		var creditAccounts []string

		type templatePosting struct {
			account   string
			amount    decimal.Decimal
			hasAmount bool
		}

		var postings []templatePosting
		var balance decimal.Decimal
		var missing []int

		for _, posting := range tx.Postings {
			account := strings.TrimSpace(posting.Account)
			if account == "" {
				continue
			}

			rawAmount := strings.TrimSpace(posting.Amount)
			entry := templatePosting{account: account}
			if rawAmount == "" {
				missing = append(missing, len(postings))
				postings = append(postings, entry)
				continue
			}

			amount, err := decimal.NewFromString(rawAmount)
			if err != nil {
				postings = append(postings, entry)
				continue
			}

			entry.amount = amount
			entry.hasAmount = true
			postings = append(postings, entry)
			balance = balance.Add(amount)
		}

		if len(postings) == 0 {
			continue
		}

		if len(missing) == 1 {
			remainder := balance.Neg()
			postings[missing[0]].amount = remainder
			postings[missing[0]].hasAmount = true
		} else if len(missing) > 1 {
			continue
		}

		for _, entry := range postings {
			if !entry.hasAmount {
				continue
			}
			if entry.amount.Sign() >= 0 {
				debitAccounts = append(debitAccounts, entry.account)
			} else {
				creditAccounts = append(creditAccounts, entry.account)
			}
		}

		if len(debitAccounts) == 0 && len(creditAccounts) == 0 {
			continue
		}

		sortedDebit := append([]string(nil), debitAccounts...)
		sortedCredit := append([]string(nil), creditAccounts...)
		sort.Strings(sortedDebit)
		sort.Strings(sortedCredit)
		templateKey := strings.Join(sortedDebit, "|") + "->" + strings.Join(sortedCredit, "|")

		if templateFreq[tx.Payee] == nil {
			templateFreq[tx.Payee] = make(map[string]templateBucket)
		}

		bucket := templateFreq[tx.Payee][templateKey]
		bucket.frequency++
		bucket.debit = sortedDebit
		bucket.credit = sortedCredit
		templateFreq[tx.Payee][templateKey] = bucket
	}

	// Convert to TemplateRecord slices and sort by frequency
	for payee, templates := range templateFreq {
		var records []TemplateRecord
		for _, bucket := range templates {
			records = append(records, TemplateRecord{
				DebitAccounts:  append([]string(nil), bucket.debit...),
				CreditAccounts: append([]string(nil), bucket.credit...),
				Frequency:      bucket.frequency,
			})
		}

		// Sort by frequency (descending)
		sort.Slice(records, func(i, j int) bool {
			if records[i].Frequency == records[j].Frequency {
				if len(records[i].DebitAccounts) == len(records[j].DebitAccounts) {
					return strings.Join(records[i].DebitAccounts, "|") < strings.Join(records[j].DebitAccounts, "|")
				}
				return len(records[i].DebitAccounts) > len(records[j].DebitAccounts)
			}
			return records[i].Frequency > records[j].Frequency
		})

		r.Templates[payee] = records
	}
}

// FindPayees returns payees that start with the given prefix.
func (r *RuntimeIntelligence) FindPayees(prefix string) []string {
	prefix = strings.ToLower(prefix)
	var matches []string

	for _, payee := range r.Payees {
		if strings.HasPrefix(strings.ToLower(payee), prefix) {
			matches = append(matches, payee)
		}
	}

	return matches
}

// FindAccounts returns account names that start with the given prefix.
func (r *RuntimeIntelligence) FindAccounts(prefix string) []string {
	return r.Accounts.Find(prefix)
}

// FindTemplates returns transaction templates for the given payee, ordered by frequency.
func (r *RuntimeIntelligence) FindTemplates(payee string) []TemplateRecord {
	if templates, exists := r.Templates[payee]; exists {
		return templates
	}
	return []TemplateRecord{}
}
