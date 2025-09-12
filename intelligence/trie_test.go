package intelligence_test

import (
	"git.sr.ht/~jakintosh/teller/intelligence"
	"reflect"
	"sort"
	"testing"
)

func TestTrie(t *testing.T) {
	trie := intelligence.NewTrie()
	words := []string{"apple", "app", "apricot", "banana"}
	for _, word := range words {
		trie.Insert(word)
	}

	testCases := []struct {
		prefix   string
		expected []string
	}{
		{"ap", []string{"apple", "app", "apricot"}},
		{"b", []string{"banana"}},
		{"apple", []string{"apple"}},
		{"c", []string{}},
		{"", []string{"apple", "app", "apricot", "banana"}},
	}

	for _, tc := range testCases {
		t.Run(tc.prefix, func(t *testing.T) {
			results := trie.Find(tc.prefix)
			sort.Strings(results)
			sort.Strings(tc.expected)
			if !reflect.DeepEqual(results, tc.expected) {
				t.Errorf("for prefix '%s', expected %v, but got %v", tc.prefix, tc.expected, results)
			}
		})
	}
}
