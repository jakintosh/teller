package core

// LoadIssue describes a non-fatal problem encountered while building startup data.
type LoadIssue struct {
	Stage   string
	Message string
}
