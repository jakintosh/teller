package intelligence

import (
	"fmt"
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

// BuildReport captures metrics and issues encountered while constructing the intelligence DB.
type BuildReport struct {
	UniquePayees    int
	UniqueTemplates int
	Issues          []string
}

// IntelligenceDB is the in-memory data store for all suggestion features.
type IntelligenceDB struct {
	Payees    []string
	Accounts  *Trie
	Templates map[string][]TemplateRecord
}

// NewIntelligenceDB creates a new intelligence database from parsed transactions.
func NewIntelligenceDB(transactions []core.Transaction) (*IntelligenceDB, BuildReport, error) {
	db := &IntelligenceDB{
		Accounts:  NewTrie(),
		Templates: make(map[string][]TemplateRecord),
	}

	// Extract unique payees
	payeeSet := make(map[string]bool)
	// Extract unique accounts
	accountSet := make(map[string]bool)
	// Capture non-fatal issues encountered during analysis.
	var issues []string

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
			if tx.Payee == "" {
				issues = append(issues, fmt.Sprintf("transaction on %s missing payee", tx.Date.Format("2006-01-02")))
			}
			if len(tx.Postings) == 0 {
				issues = append(issues, fmt.Sprintf("transaction on %s for payee %q has no postings", tx.Date.Format("2006-01-02"), tx.Payee))
			}
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
				issues = append(issues, fmt.Sprintf("payee %q has posting with missing account", tx.Payee))
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
				issues = append(issues, fmt.Sprintf("payee %q account %q has invalid amount %q", tx.Payee, account, rawAmount))
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
			issues = append(issues, fmt.Sprintf("payee %q transaction on %s has %d postings without amounts", tx.Payee, tx.Date.Format("2006-01-02"), len(missing)))
		}

		for _, entry := range postings {
			if !entry.hasAmount {
				issues = append(issues, fmt.Sprintf("payee %q account %q skipped due to missing amount", tx.Payee, entry.account))
				continue
			}
			if entry.amount.Sign() >= 0 {
				debitAccounts = append(debitAccounts, entry.account)
			} else {
				creditAccounts = append(creditAccounts, entry.account)
			}
		}

		if len(debitAccounts) == 0 && len(creditAccounts) == 0 {
			issues = append(issues, fmt.Sprintf("payee %q transaction on %s produced empty template", tx.Payee, tx.Date.Format("2006-01-02")))
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
	var totalTemplates int
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
		totalTemplates += len(records)
	}

	report := BuildReport{
		UniquePayees:    len(db.Payees),
		UniqueTemplates: totalTemplates,
		Issues:          issues,
	}

	return db, report, nil
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
