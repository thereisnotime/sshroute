package ssh

import "testing"

// FuzzParseArgs exercises the SSH argument parser with arbitrary inputs.
// Run with: go test -fuzz=FuzzParseArgs ./internal/ssh/
func FuzzParseArgs(f *testing.F) {
	// Seed corpus from realistic SSH invocations.
	f.Add([]byte("user@host.example.com"))
	f.Add([]byte("-p 2222 user@host"))
	f.Add([]byte("-i ~/.ssh/key -l alice host"))
	f.Add([]byte("-J bastion host command arg"))
	f.Add([]byte(""))
	f.Add([]byte("-v -v -v host"))

	f.Fuzz(func(t *testing.T, data []byte) {
		// Must not panic regardless of input.
		_ = ParseArgs(tokenize(string(data)))
	})
}

// tokenize splits a string on whitespace, mimicking shell word splitting.
func tokenize(s string) []string {
	if s == "" {
		return nil
	}
	var tokens []string
	cur := ""
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' {
			if cur != "" {
				tokens = append(tokens, cur)
				cur = ""
			}
		} else {
			cur += string(r)
		}
	}
	if cur != "" {
		tokens = append(tokens, cur)
	}
	return tokens
}
