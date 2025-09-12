package intelligence

// TrieNode represents a single node in the Trie.
type TrieNode struct {
	children map[rune]*TrieNode
	isEnd    bool
}

// NewTrieNode creates and returns a new TrieNode.
func NewTrieNode() *TrieNode {
	return &TrieNode{
		children: make(map[rune]*TrieNode),
		isEnd:    false,
	}
}

// Trie represents a Trie data structure for storing and searching strings.
type Trie struct {
	root *TrieNode
}

// NewTrie creates and returns a new Trie.
func NewTrie() *Trie {
	return &Trie{
		root: NewTrieNode(),
	}
}

// Insert adds a word to the Trie.
func (t *Trie) Insert(word string) {
	node := t.root
	for _, char := range word {
		if _, ok := node.children[char]; !ok {
			node.children[char] = NewTrieNode()
		}
		node = node.children[char]
	}
	node.isEnd = true
}

// Find returns all words in the Trie that start with the given prefix.
func (t *Trie) Find(prefix string) []string {
	node := t.root
	for _, char := range prefix {
		if _, ok := node.children[char]; !ok {
			return []string{}
		}
		node = node.children[char]
	}

	var results []string
	t.collect(node, prefix, &results)
	return results
}

// collect recursively finds all words starting from a given node.
func (t *Trie) collect(node *TrieNode, prefix string, results *[]string) {
	if node.isEnd {
		*results = append(*results, prefix)
	}

	for char, childNode := range node.children {
		t.collect(childNode, prefix+string(char), results)
	}
}
