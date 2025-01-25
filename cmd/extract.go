package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

// dirFlag specifies the directory path provided as a flag.
// filePattern defines the pattern for matching file names.
// categoriesArg holds the argument for specifying categories.
var (
	dirFlag       string
	filePattern   string
	categoriesArg string
)

// extractCmd defines a Cobra command for extracting code snippets based on specified categories in annotated files.
var extractCmd = &cobra.Command{
	Use:   "extract",
	Short: "Extract code snippets by specified categories",
	Long: `Extract scans your files for code snippets annotated with:
	
# start: {"foundation": ["messages"], ...}
... code ...
# end: {"foundation": ["messages"], ...}

It requires you to specify the categories you’re looking for (e.g., foundation, tests).
Usage example:
brio extract --categories "messages:foundation,tests" --dir ./ --files "*.py"
`,
	Run: func(cmd *cobra.Command, args []string) {
		// 1. Parse user-supplied categories into a map.
		catMap := parseCategoryArg(categoriesArg)

		// 2. Collect all matching files.
		files, err := collectFiles(dirFlag, filePattern)
		if err != nil {
			log.Fatalf("Error collecting files: %v", err)
		}

		// 3. Extract snippets from those files that match the categories.
		matchedSnippets := extractSnippets(files, catMap)

		// 4. Print the results in Markdown (you can adapt to other formats).
		printSnippetsMarkdown(matchedSnippets)
	},
}

// init initializes the command-line interface by adding the extractCmd as a subcommand to rootCmd.
// It defines flags for the extractCmd, allowing users to specify directory, file pattern, and categories to process.
func init() {
	// Register extractCmd as a subcommand of the rootCmd.
	rootCmd.AddCommand(extractCmd)

	// Define local flags for this command only.
	extractCmd.Flags().StringVarP(&dirFlag, "dir", "d", ".", "Directory to scan")
	extractCmd.Flags().StringVarP(&filePattern, "files", "f", "*.py", "File pattern to match (e.g. *.py)")
	extractCmd.Flags().StringVarP(&categoriesArg, "categories", "c", "",
		"Categories to extract, e.g. 'messages:foundation,tests'")
}

// parseCategoryArg parses a string argument with categories and domains into a map of categories to their associated domains.
func parseCategoryArg(categoryArg string) map[string][]string {
	result := make(map[string][]string)
	if categoryArg == "" {
		return result
	}

	parts := strings.Split(categoryArg, ",")
	var currentDomain string

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.Contains(part, ":") {
			// e.g., "messages:foundation"
			splitPart := strings.SplitN(part, ":", 2)
			currentDomain = strings.TrimSpace(splitPart[0])
			category := strings.TrimSpace(splitPart[1])
			addToCategoryMap(result, category, currentDomain)
		} else {
			// e.g., "tests" with inherited domain
			addToCategoryMap(result, part, currentDomain)
		}
	}

	return result
}

// addToCategoryMap adds a domain to the specified category in the map.
// If the category does not exist, it initializes it with an empty slice.
// Prevents duplicate domains within a category.
// Ensures the slice contains an empty string if the domain is an empty string and the category is new.
func addToCategoryMap(catMap map[string][]string, category, domain string) {
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
	} else {
		// If domain is empty string, ensure the slice contains an empty string
		if len(catMap[category]) == 0 {
			catMap[category] = append(catMap[category], "")
		}
	}
}

// collectFiles scans the provided directory and returns a list of files matching the specified pattern.
// dir is the root directory to start the search. pattern is the glob pattern for matching file names.
// Returns a slice of matching file paths or an error if traversal fails.
func collectFiles(dir, pattern string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			matched, err := filepath.Match(pattern, filepath.Base(path))
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

// parseTagJSON extracts JSON data from a line of text and parses it into a map of string slices.
// Returns an error if JSON parsing fails or no JSON is found.
func parseTagJSON(line string) (map[string][]string, error) {
	startIdx := strings.Index(line, "{")
	endIdx := strings.LastIndex(line, "}")
	if startIdx == -1 || endIdx == -1 || endIdx < startIdx {
		return nil, fmt.Errorf("no JSON found in line: %s", line)
	}
	jsonStr := line[startIdx : endIdx+1]

	var data map[string][]string
	err := json.Unmarshal([]byte(jsonStr), &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// snippet represents a code snippet with its associated metadata including file path, line range, categories, and content.
type snippet struct {
	File       string
	StartLine  int
	EndLine    int
	Categories map[string][]string
	Content    []string
}

// snippetData represents a snippet of code extracted from a file, including its associated metadata and content lines.
type snippetData struct {
	categories map[string][]string
	startLine  int
	lines      []string
}

// extractSnippets scans a list of files for code snippets annotated with start and end tags containing category metadata.
// It extracts the matching snippets based on the provided category map and returns them as a slice of snippet objects.
func extractSnippets(files []string, catMap map[string][]string) []snippet {
	var results []snippet

	// Regexes for identifying lines that contain # start: / # end: plus JSON.
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

			// Check for start line
			if startPattern.MatchString(line) {
				// If we already have an active snippet, forcibly close it or discard it.
				activeSnippet = nil

				tagData, err := parseTagJSON(line)
				if err != nil {
					continue
				}
				activeSnippet = &snippetData{
					categories: tagData,
					startLine:  lineNum,
					lines:      []string{},
				}
				continue
			}

			// Check for end line
			if endPattern.MatchString(line) {
				if activeSnippet != nil {
					_, _ = parseTagJSON(line)
					// For simplicity, we won't merge endTagData with startTagData,
					// but you could if both matter.

					snippetObj := snippet{
						File:       filePath,
						StartLine:  activeSnippet.startLine,
						EndLine:    lineNum,
						Categories: activeSnippet.categories,
						Content:    activeSnippet.lines,
					}

					// If snippet matches user-requested categories, add it to results
					if snippetMatches(snippetObj, catMap) {
						results = append(results, snippetObj)
					}
					activeSnippet = nil
				}
				continue
			}

			// If we're inside a snippet, add the line to the snippet content
			if activeSnippet != nil {
				activeSnippet.lines = append(activeSnippet.lines, line)
			}
		}
		_ = f.Close() // close file
	}

	return results
}

// snippetMatches checks if a snippet matches the requested category-domain mapping specified in catMap.
// If catMap is empty, the function returns true, indicating all snippets should match.
// The function iterates through the snippet's categories and checks for intersections with the requested domains in catMap.
func snippetMatches(s snippet, catMap map[string][]string) bool {
	// If user specified no categories, everything matches.
	if len(catMap) == 0 {
		return true
	}

	// e.g. snippet categories: {"foundation": ["messages"], "model": ["messages"]}
	// catMap might be: {"foundation": ["messages"], "tests": ["messages"]}
	for snippetCat, snippetDomains := range s.Categories {
		if requestedDomains, found := catMap[snippetCat]; found {
			// If category is requested with no domain => matches any domain for that category.
			if len(requestedDomains) == 0 {
				return true
			}
			// Otherwise, check domain intersection
			for _, sd := range snippetDomains {
				for _, rd := range requestedDomains {
					if sd == rd {
						return true
					}
				}
			}
		}
	}
	return false
}

// printSnippetsMarkdown prints a list of code snippets in Markdown format, including file name, line range, and categories.
func printSnippetsMarkdown(snips []snippet) {
	if len(snips) == 0 {
		fmt.Println("No snippets found for the given categories.")
		return
	}

	for _, s := range snips {
		fmt.Printf("## File: %s (lines %d-%d)\n\n", s.File, s.StartLine, s.EndLine)

		// Display categories
		catInfo := []string{}
		for cat, domains := range s.Categories {
			catInfo = append(catInfo, fmt.Sprintf(`%s -> %v`, cat, domains))
		}
		fmt.Printf("**Categories**: %s\n\n", strings.Join(catInfo, ", "))

		// Code block
		fmt.Println("```python")
		for _, line := range s.Content {
			fmt.Println(line)
		}
		fmt.Println("```")
	}
}
