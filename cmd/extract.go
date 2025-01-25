package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var extractCmd = &cobra.Command{
	Use:   "extract [category_list]",
	Short: "Extract code snippets by specified categories",
	Long: `Extract code snippets from your codebase. Example usage:

  brio extract messages:foundation,tests --dir ./ --files *.py

Where "messages:foundation,tests" means:
- look for snippet tags containing categories "foundation" or "tests" specifically referencing "messages".
`,
	Run: func(cmd *cobra.Command, args []string) {
		// 1. Get flags
		dir, _ := cmd.Flags().GetString("dir")
		filePattern, _ := cmd.Flags().GetString("files")
		categories, _ := cmd.Flags().GetString("categories")

		// 2. Parse categories (e.g. "messages:foundation,tests")
		//    We'll store them in a map: { foundation: [messages], tests: [messages], ... }
		catMap := parseCategoryArg(categories)

		// 3. Collect files
		files, err := collectFiles(dir, filePattern)
		if err != nil {
			log.Fatalf("Error collecting files: %v", err)
		}

		// 4. Extract snippets
		matchedSnippets := extractSnippets(files, catMap)

		// 5. Print snippets (Markdown format example)
		printSnippetsMarkdown(matchedSnippets)
	},
}

func init() {
	// Register the extract command as a subcommand of the root
	rootCmd.AddCommand(extractCmd)

	// Define flags
	extractCmd.Flags().StringP("dir", "d", ".", "Directory to scan")
	extractCmd.Flags().StringP("files", "f", "*.py", "File pattern to match (e.g., *.py, *.go)")
	extractCmd.Flags().StringP("categories", "c", "", "Categories to extract (e.g. 'messages:foundation,tests')")
}

// parseCategoryArg converts something like "messages:foundation,tests" into a map:
//
//	{
//	  "foundation": ["messages"],
//	  "tests": ["messages"]
//	}
//
// If the user provides multiple sets like "messages:foundation,tests, alerts:foundation",
// we'd have "foundation": ["messages", "alerts"], "tests": ["messages"].
func parseCategoryArg(categoryArg string) map[string][]string {
	result := make(map[string][]string)
	if categoryArg == "" {
		return result
	}

	// Split by spaces or comma? Here we consider spaces optional, so let's just separate on whitespace first.
	// But let's keep it simple: "messages:foundation,tests" -> let's split by spaces (if user typed them) then by commas.
	rawSets := strings.Split(categoryArg, " ")
	for _, set := range rawSets {
		// further split by commas
		parts := strings.Split(set, ",")
		for _, part := range parts {
			// e.g. "messages:foundation"
			if strings.Contains(part, ":") {
				splitPart := strings.SplitN(part, ":", 2)
				keyDomain := splitPart[0] // e.g. messages
				category := splitPart[1]  // e.g. foundation

				// We may want to handle multiple categories after the colon, but let's keep it single
				addToCategoryMap(result, category, keyDomain)
			} else {
				// e.g. "tests" with no domain
				addToCategoryMap(result, part, "") // blank domain
			}
		}
	}
	return result
}

// addToCategoryMap is a helper to append new domain keys
func addToCategoryMap(catMap map[string][]string, category, domain string) {
	// category is like "foundation" or "tests", domain is like "messages" or ""
	if _, exists := catMap[category]; !exists {
		catMap[category] = []string{}
	}
	if domain != "" {
		// Avoid duplicates
		for _, d := range catMap[category] {
			if d == domain {
				return
			}
		}
		catMap[category] = append(catMap[category], domain)
	}
}

// collectFiles returns all files in `dir` that match `filePattern` (e.g., "*.py").
func collectFiles(dir, filePattern string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			matched, err := filepath.Match(filePattern, filepath.Base(path))
			if err != nil {
				return err
			}
			if matched {
				files = append(files, path)
			}
		}
		return nil
	})
	return files, err
}

// parseTagJSON tries to parse the JSON inside "# start: { ... }" or "# end: { ... }".
func parseTagJSON(line string) (map[string][]string, error) {
	// Example line might be:
	// # start: {"foundation": ["messages"], "model": ["messages"]}
	// We want to extract the substring after ": " and parse as JSON.
	// This naive approach: find the first '{' and parse from there to the end of the line (or until '}'?).
	startIdx := strings.Index(line, "{")
	endIdx := strings.LastIndex(line, "}")
	if startIdx == -1 || endIdx == -1 || endIdx < startIdx {
		return nil, fmt.Errorf("No JSON found in line")
	}
	jsonStr := line[startIdx : endIdx+1]

	var data map[string][]string
	err := json.Unmarshal([]byte(jsonStr), &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// snippet represents a captured code snippet with metadata
type snippet struct {
	File       string
	StartLine  int
	EndLine    int
	Categories map[string][]string // The parsed JSON from "# start:" or "# end:"
	Content    []string
}

// snippetData tracks the current snippet being built as we parse.
type snippetData struct {
	categories map[string][]string
	startLine  int
	lines      []string
}

func extractSnippets(files []string, catMap map[string][]string) []snippet {
	var results []snippet

	// We'll define a regex to match the start/end lines:
	// e.g. "# start:" or "# end:"
	// We'll parse the JSON after it.
	startPattern := regexp.MustCompile(`(?i)#\s*start:\s*\{`)
	endPattern := regexp.MustCompile(`(?i)#\s*end:\s*\{`)

	for _, filePath := range files {
		f, err := os.Open(filePath)
		if err != nil {
			log.Printf("Failed to open file %s: %v", filePath, err)
			continue
		}
		scanner := bufio.NewScanner(f)

		var activeSnippet *snippetData
		lineNum := 0

		for scanner.Scan() {
			lineNum++
			line := scanner.Text()

			// Check for # start:
			if startPattern.FindStringIndex(line) != nil {
				// If we already have an activeSnippet, that means we never ended the previous snippet.
				// Let's finalize it or discard it. For simplicity, finalize it if needed.
				if activeSnippet != nil {
					// We can finalize it if we want, but let's discard or finalize. We'll discard for clarity.
					activeSnippet = nil
				}

				tagData, err := parseTagJSON(line)
				if err != nil {
					// Invalid JSON or not parseable
					continue
				}
				activeSnippet = &snippetData{
					categories: tagData,
					startLine:  lineNum,
					lines:      []string{}, // content starts after this line
				}
				continue
			}

			// Check for # end:
			if endPattern.FindStringIndex(line) != nil {
				if activeSnippet != nil {
					// parse end tag
					_, err := parseTagJSON(line)
					if err != nil {
						// If there's an error, let's just ignore it
						// or we could forcibly end the snippet anyway
					}
					// We'll consider the snippet ended if the end tag shares at least one category with start tag
					// or is empty. For a simpler approach, assume matching if the snippet is active.
					snippetObj := snippet{
						File:       filePath,
						StartLine:  activeSnippet.startLine,
						EndLine:    lineNum,
						Categories: activeSnippet.categories, // or we could merge with endTagData
						Content:    activeSnippet.lines,
					}

					// Decide if snippet matches user-requested categories:
					if snippetMatches(snippetObj, catMap) {
						results = append(results, snippetObj)
					}

					// Clear active snippet
					activeSnippet = nil
				}
				continue
			}

			// If we're in an active snippet, record this line
			if activeSnippet != nil {
				activeSnippet.lines = append(activeSnippet.lines, line)
			}
		}
		f.Close()
	}
	return results
}

// snippetMatches checks if the snippet's categories overlap with the user-requested categories in catMap
func snippetMatches(s snippet, catMap map[string][]string) bool {
	// If catMap is empty, we return true (meaning user didn't specify categories, so we match everything).
	if len(catMap) == 0 {
		return true
	}

	// s.Categories might look like {"foundation": ["messages"], "model": ["messages"] }
	// catMap might look like {"foundation": ["messages"], "tests": ["messages"]}
	for snippetCat, snippetDomains := range s.Categories {
		if requestedDomains, catRequested := catMap[snippetCat]; catRequested {
			// If category is requested, check domain intersection
			if len(requestedDomains) == 0 {
				// e.g. user typed "foundation" with no domain. That means any domain for this category matches.
				return true
			}
			// Otherwise, see if snippetDomains has intersection with requestedDomains
			for _, sd := range snippetDomains {
				for _, rd := range requestedDomains {
					if sd == rd {
						// domain match => snippet qualifies
						return true
					}
				}
			}
		}
	}

	// If we never found a match, return false
	return false
}

// printSnippetsMarkdown prints the extracted snippets in a Markdown-friendly manner
func printSnippetsMarkdown(snips []snippet) {
	if len(snips) == 0 {
		fmt.Println("No snippets found for the given categories.")
		return
	}

	for _, s := range snips {
		fmt.Printf("## File: %s (lines %d-%d)\n\n", s.File, s.StartLine, s.EndLine)

		// Display categories
		// e.g. "Categories: foundation -> [messages], model -> [messages]"
		catInfo := []string{}
		for cat, domains := range s.Categories {
			catInfo = append(catInfo, fmt.Sprintf("%s -> %v", cat, domains))
		}
		fmt.Printf("**Categories**: %s\n\n", strings.Join(catInfo, ", "))

		fmt.Println("```python") // or whatever language highlight you prefer
		for _, line := range s.Content {
			fmt.Println(line)
		}
		fmt.Println("```")
	}
}
