package intelligence

import (
	"sort"
	"strings"

	"git.sr.ht/~jakintosh/teller/core"
)

// TemplateRecord stores a transaction structure and its frequency.
type TemplateRecord struct {
	Accounts  []string
	Frequency int
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
	templateFreq := make(map[string]map[string]int) // payee -> template_key -> frequency

	for _, tx := range transactions {
		if tx.Payee == "" || len(tx.Postings) == 0 {
			continue
		}

		// Extract account names from postings
		var accounts []string
		for _, posting := range tx.Postings {
			if posting.Account != "" {
				accounts = append(accounts, posting.Account)
			}
		}

		if len(accounts) == 0 {
			continue
		}

		// Sort accounts to create a consistent template key
		sort.Strings(accounts)
		templateKey := strings.Join(accounts, "|")

		// Initialize payee map if needed
		if templateFreq[tx.Payee] == nil {
			templateFreq[tx.Payee] = make(map[string]int)
		}

		// Increment frequency
		templateFreq[tx.Payee][templateKey]++
	}

	// Convert to TemplateRecord slices and sort by frequency
	for payee, templates := range templateFreq {
		var records []TemplateRecord
		for templateKey, frequency := range templates {
			accounts := strings.Split(templateKey, "|")
			records = append(records, TemplateRecord{
				Accounts:  accounts,
				Frequency: frequency,
			})
		}

		// Sort by frequency (descending)
		sort.Slice(records, func(i, j int) bool {
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