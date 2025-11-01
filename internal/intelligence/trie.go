package intelligence

import "sort"

// TrieNode represents a node in the Trie data structure.
type TrieNode struct {
	children map[rune]*TrieNode
	isEnd    bool
	word     string // Store the complete word at terminal nodes
}

// Trie is a prefix tree for efficient prefix-based string searching.
type Trie struct {
	root *TrieNode
}

// NewTrie creates a new empty Trie.
func NewTrie() *Trie {
	return &Trie{
		root: &TrieNode{
			children: make(map[rune]*TrieNode),
		},
	}
}

// Insert adds a word to the Trie.
func (t *Trie) Insert(word string) {
	current := t.root
	for _, char := range word {
		if current.children[char] == nil {
			current.children[char] = &TrieNode{
				children: make(map[rune]*TrieNode),
			}
		}
		current = current.children[char]
	}
	current.isEnd = true
	current.word = word
}

// Find returns all words in the Trie that start with the given prefix.
func (t *Trie) Find(prefix string) []string {
	current := t.root

	// Navigate to the prefix node
	for _, char := range prefix {
		if current.children[char] == nil {
			return []string{} // Prefix not found
		}
		current = current.children[char]
	}

	// Collect all words from this prefix node
	var results []string
	t.collectWords(current, &results)

	// Sort results for consistent output
	sort.Strings(results)
	return results
}

// collectWords performs DFS to collect all complete words from a given node.
func (t *Trie) collectWords(node *TrieNode, results *[]string) {
	if node.isEnd {
		*results = append(*results, node.word)
	}

	for _, child := range node.children {
		t.collectWords(child, results)
	}
}
