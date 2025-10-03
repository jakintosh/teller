package intelligence

import (
	"sort"
	"strings"

	"git.sr.ht/~jakintosh/teller/core"
	"github.com/shopspring/decimal"
)

type templateBucket struct {
	debit     []string
	credit    []string
	frequency int
}

// TemplateRecord stores a transaction structure and its frequency.
type TemplateRecord struct {
	DebitAccounts  []string
	CreditAccounts []string
	Frequency      int
}

// IntelligenceDB is the in-memory data store for all suggestion features.
type IntelligenceDB struct {
	Payees    []string
	Accounts  *Trie
	Templates map[string][]TemplateRecord
}

// NewIntelligenceDB creates a new intelligence database from parsed transactions.
func NewIntelligenceDB(transactions []core.Transaction) (*IntelligenceDB, error) {
	db := &IntelligenceDB{
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
	db.Payees = make([]string, 0, len(payeeSet))
	for payee := range payeeSet {
		db.Payees = append(db.Payees, payee)
	}
	sort.Strings(db.Payees)

	// Insert all accounts into the Trie
	for account := range accountSet {
		db.Accounts.Insert(account)
	}

	// Analyze transaction templates
	templateFreq := make(map[string]map[string]templateBucket) // payee -> key -> bucket

	for _, tx := range transactions {
		if tx.Payee == "" || len(tx.Postings) == 0 {
			continue
		}

		var debitAccounts []string
		var creditAccounts []string
		for _, posting := range tx.Postings {
			if posting.Account == "" {
				continue
			}
			amount, err := decimal.NewFromString(strings.TrimSpace(posting.Amount))
			if err != nil {
				continue
			}
			if amount.Sign() >= 0 {
				debitAccounts = append(debitAccounts, posting.Account)
			} else {
				creditAccounts = append(creditAccounts, posting.Account)
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

		db.Templates[payee] = records
	}

	return db, nil
}

// FindPayees returns payees that start with the given prefix.
func (db *IntelligenceDB) FindPayees(prefix string) []string {
	prefix = strings.ToLower(prefix)
	var matches []string

	for _, payee := range db.Payees {
		if strings.HasPrefix(strings.ToLower(payee), prefix) {
			matches = append(matches, payee)
		}
	}

	return matches
}

// FindAccounts returns account names that start with the given prefix.
func (db *IntelligenceDB) FindAccounts(prefix string) []string {
	return db.Accounts.Find(prefix)
}

// FindTemplates returns transaction templates for the given payee, ordered by frequency.
func (db *IntelligenceDB) FindTemplates(payee string) []TemplateRecord {
	if templates, exists := db.Templates[payee]; exists {
		return templates
	}
	return []TemplateRecord{}
}
