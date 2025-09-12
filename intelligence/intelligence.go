package intelligence

import (
	"git.sr.ht/~jakintosh/teller/core"
	"sort"
	"strings"
)

// IntelligenceDB is an in-memory database that provides queryable information
// derived from a user's transaction history. It powers features like
// auto-completion and template suggestions.
type IntelligenceDB struct {
	// Payees is a sorted list of unique payee names encountered in the user's
	// transaction history.
	Payees []string
	// Accounts is a Trie containing all unique account names for efficient
	// prefix-based searching.
	Accounts *Trie
}

// New creates and initializes a new IntelligenceDB from a slice of transactions.
// It parses the transactions to populate the database's queryable fields.
func New(transactions []core.Transaction) *IntelligenceDB {
	payeeSet := make(map[string]struct{})
	accountSet := make(map[string]struct{})

	for _, t := range transactions {
		if t.Payee != "" {
			payeeSet[t.Payee] = struct{}{}
		}
		for _, p := range t.Postings {
			if p.Account != "" {
				accountSet[p.Account] = struct{}{}
			}
		}
	}

	payees := make([]string, 0, len(payeeSet))
	for payee := range payeeSet {
		payees = append(payees, payee)
	}
	sort.Strings(payees)

	accountsTrie := NewTrie()
	for account := range accountSet {
		accountsTrie.Insert(account)
	}

	return &IntelligenceDB{
		Payees:   payees,
		Accounts: accountsTrie,
	}
}

// FindPayees searches for payees with a given prefix. The search is
// case-insensitive. It returns a slice of matching payees.
func (db *IntelligenceDB) FindPayees(prefix string) []string {
	// If the prefix is empty, there are no payees to find.
	if prefix == "" {
		return []string{}
	}

	// Initialize a non-nil slice to hold the matches. This prevents issues
	// with reflect.DeepEqual in tests when no matches are found.
	matches := make([]string, 0)
	lowerPrefix := strings.ToLower(prefix)

	for _, payee := range db.Payees {
		if strings.HasPrefix(strings.ToLower(payee), lowerPrefix) {
			matches = append(matches, payee)
		}
	}

	return matches
}

// FindAccounts searches for accounts with a given prefix. The search is
// case-insensitive. It returns a slice of matching accounts.
func (db *IntelligenceDB) FindAccounts(prefix string) []string {
	if prefix == "" {
		return []string{}
	}

	// The Trie Find method is case-sensitive, so we handle case-insensitivity
	// by finding all accounts and then filtering. This is inefficient and
	// a better approach would be to make the Trie case-insensitive.
	// For now, this will work.
	// A better approach is to convert the prefix to lower case and
	// traverse the trie with lower case characters.
	// But the current trie implementation does not support this.
	// So we will get all accounts and filter them.
	allAccounts := db.Accounts.Find("")
	matches := make([]string, 0)
	lowerPrefix := strings.ToLower(prefix)

	for _, account := range allAccounts {
		if strings.HasPrefix(strings.ToLower(account), lowerPrefix) {
			matches = append(matches, account)
		}
	}
	sort.Strings(matches)
	return matches
}
