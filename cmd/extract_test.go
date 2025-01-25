package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestParseCategoryArg checks if parseCategoryArg correctly parses strings like "messages:foundation,tests".
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
			input: "alerts:foundation, messages:tests",
			expected: map[string][]string{
				"foundation": {"alerts"},
				"tests":      {"messages"},
			},
		},
		{
			input:    "", // No categories
			expected: map[string][]string{},
		},
		{
			input: "foundation",
			expected: map[string][]string{
				"foundation": {""}, // domain is empty
			},
		},
	}

	for _, tc := range tests {
		result := parseCategoryArg(tc.input)
		assert.Equal(t, tc.expected, result, "Input: %s", tc.input)
	}
}

// TestParseTagJSON verifies that parseTagJSON can extract and unmarshal JSON from comment lines.
func TestParseTagJSON(t *testing.T) {
	// Valid line
	line := `# start: {"foundation": ["messages"], "model": ["messages"]}`
	data, err := parseTagJSON(line)
	assert.Nil(t, err)
	assert.Equal(t, []string{"messages"}, data["foundation"])
	assert.Equal(t, []string{"messages"}, data["model"])

	// Invalid line (no JSON)
	lineInvalid := `# start: foundation: [messages]`
	data, err = parseTagJSON(lineInvalid)
	assert.Nil(t, data)
	assert.NotNil(t, err)

	// Incomplete JSON
	lineIncomplete := `# start: {"foundation": ["messages"]`
	data, err = parseTagJSON(lineIncomplete)
	assert.Nil(t, data)
	assert.NotNil(t, err)
}

// TestSnippetMatches ensures that snippetMatches correctly identifies whether a snippet’s categories
// intersect with user-requested categories in catMap.
func TestSnippetMatches(t *testing.T) {
	// Build a snippet
	snip := snippet{
		Categories: map[string][]string{
			"foundation": {"messages"},
			"model":      {"messages"},
		},
	}

	// Case 1: catMap is empty => should match everything
	emptyCatMap := map[string][]string{}
	assert.True(t, snippetMatches(snip, emptyCatMap), "Empty catMap should match everything")

	// Case 2: catMap with intersecting category and domain
	catMap := map[string][]string{
		"foundation": {"messages"},
	}
	assert.True(t, snippetMatches(snip, catMap), "Should match because foundation:messages intersects")

	// Case 3: catMap with same category, different domain => no match
	catMapNoMatch := map[string][]string{
		"foundation": {"alerts"},
	}
	assert.False(t, snippetMatches(snip, catMapNoMatch), "Different domain => no match")

	// Case 4: catMap with category that has empty domain => matches all domains for that category
	catMapEmptyDomain := map[string][]string{
		"foundation": {}, // user typed just "foundation" with no domain
	}
	assert.True(t, snippetMatches(snip, catMapEmptyDomain), "Empty domain => match any domain for that category")
}

// TestExtractSnippets simulates reading from a temporary file with annotated code to ensure
// the extraction logic works end-to-end. It won't run the actual Cobra command; instead, it tests
// `extractSnippets` directly.
func TestExtractSnippets(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Create a test file with some snippet annotations
	fileContent := `# start: {"foundation": ["messages"], "model": ["messages"]}
class Message(TenantModel):
    pass
# end: {"foundation": ["messages"]}

# start: {"tests": ["messages"]}
def test_message():
    assert True
# end: {"tests": ["messages"]}`

	// Write the file
	filePath := filepath.Join(tempDir, "test_snippets.py")
	err := os.WriteFile(filePath, []byte(fileContent), 0644)
	assert.Nil(t, err)

	// We'll call extractSnippets directly
	files := []string{filePath}

	// catMap => "foundation": ["messages"]
	catMap := map[string][]string{
		"foundation": {"messages"},
	}

	snips := extractSnippets(files, catMap)
	assert.Len(t, snips, 1, "Should find exactly one matching snippet (the foundation one).")

	// Validate the snippet
	snippet := snips[0]
	assert.Equal(t, filePath, snippet.File)
	assert.Equal(t, map[string][]string{"foundation": {"messages"}, "model": {"messages"}}, snippet.Categories)
	assert.Contains(t, snippet.Content[0], "class Message(TenantModel):")
	assert.Contains(t, snippet.Content[1], "pass")
}

// TestParseCategoryArgMultipleDomains tests multiple domain usage, e.g. "messages:foundation, alerts:foundation".
func TestParseCategoryArgMultipleDomains(t *testing.T) {
	input := "messages:foundation, alerts:foundation"
	expected := map[string][]string{
		"foundation": {"messages", "alerts"},
	}

	result := parseCategoryArg(input)
	assert.Equal(t, expected, result)
}
