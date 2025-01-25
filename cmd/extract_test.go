package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseCategoryArg(t *testing.T) {
	tests := []struct {
		input    string
		expected map[string][]string
	}{
		{
			input: "messages:foundation,tests",
			expected: map[string][]string{
				"foundation": {"messages"},
				"tests":      {"messages"},
			},
		},
		{
			input:    "",
			expected: map[string][]string{},
		},
		{
			input: "foundation",
			expected: map[string][]string{
				"foundation": {""},
			},
		},
	}

	for _, tc := range tests {
		result := parseCategoryArg(tc.input)
		assert.Equal(t, tc.expected, result, "Input: %s", tc.input)
	}
}

func TestParseTagJSON(t *testing.T) {
	line := `# start: {"foundation": ["messages"], "model": ["messages"]}`
	data, err := parseTagJSON(line)
	assert.Nil(t, err)
	assert.Equal(t, []string{"messages"}, data["foundation"])
	assert.Equal(t, []string{"messages"}, data["model"])

	// Invalid JSON
	invalid := `# start: foundation: [messages]`
	data, err = parseTagJSON(invalid)
	assert.Nil(t, data)
	assert.NotNil(t, err)
}

func TestSnippetMatches(t *testing.T) {
	snip := snippet{
		Categories: map[string][]string{
			"foundation": {"messages"},
			"model":      {"messages"},
		},
	}

	// Empty catMap => matches everything
	assert.True(t, snippetMatches(snip, map[string][]string{}))

	// Matching category + domain
	assert.True(t, snippetMatches(snip, map[string][]string{"foundation": {"messages"}}))

	// Matching category, but domain mismatch
	assert.False(t, snippetMatches(snip, map[string][]string{"foundation": {"alert"}}))

	// Category matches, but catMap has no domain => matches
	assert.True(t, snippetMatches(snip, map[string][]string{"foundation": {}}))
}

func TestExtractSnippets(t *testing.T) {
	tempDir := t.TempDir()

	// Test file content with two snippets
	fileContent := `# start: {"foundation": ["messages"], "model": ["messages"]}
class Message(TenantModel):
    pass
# end: {"foundation": ["messages"]}

# start: {"tests": ["messages"]}
def test_message():
    assert True
# end: {"tests": ["messages"]}`

	filePath := filepath.Join(tempDir, "test_snippets.py")
	err := os.WriteFile(filePath, []byte(fileContent), 0644)
	assert.Nil(t, err)

	files := []string{filePath}
	catMap := map[string][]string{
		"foundation": {"messages"},
	}

	snips := extractSnippets(files, catMap)
	assert.Len(t, snips, 1, "Should only match the snippet labeled foundation:messages")

	s := snips[0]
	assert.Equal(t, filePath, s.File)
	assert.Contains(t, s.Categories, "foundation")
	assert.Contains(t, s.Content[0], "class Message(TenantModel)")
	assert.Contains(t, s.Content[1], "pass")
}
