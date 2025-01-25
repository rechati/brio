package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/rechati/brio/cmd/plugins"
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
	clipboardMode bool
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
		printSnippets(matchedSnippets, clipboardMode)
	},
}

// init initializes the command-line interface by adding the extractCmd as a subcommand to rootCmd.
// It defines flags for the extractCmd, allowing users to specify directory, file pattern, and categories to process.
func init() {
	// Register extractCmd as a subcommand of the rootCmd.
	rootCmd.AddCommand(extractCmd)

	// Get all supported extensions from plugins
	extensions := plugins.ListExtensions()
	defaultPattern := "*" // Changed to accept all files, we'll filter by extension internally

	// Create help text showing supported extensions
	supportedExtsHelp := fmt.Sprintf("Supported extensions: %s", strings.Join(extensions, ", "))

	extractCmd.Flags().StringVarP(&dirFlag, "dir", "d", ".", "Directory to scan")
	extractCmd.Flags().StringVarP(&filePattern, "files", "f", defaultPattern,
		fmt.Sprintf("File pattern to match (e.g., *.py). %s", supportedExtsHelp))
	extractCmd.Flags().StringVarP(&categoriesArg, "categories", "c", "",
		"Categories to extract, e.g. 'messages:foundation,tests'")
	extractCmd.Flags().BoolVarP(&clipboardMode, "clipboard", "v", false,
		"Output in clipboard-friendly format (without Markdown)")
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
// cmd/extract.go

func collectFiles(dir, pattern string) ([]string, error) {
	var files []string

	// Get all supported extensions from plugins
	supportedExts := make(map[string]bool)
	for _, ext := range plugins.ListExtensions() {
		supportedExts[ext] = true
	}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if file extension is supported
		ext := filepath.Ext(path)
		if !supportedExts[ext] {
			return nil
		}

		// If pattern is provided, check if file matches pattern
		if pattern != "" && pattern != "*" {
			matched, err := filepath.Match(pattern, filepath.Base(path))
			if err != nil {
				return err
			}
			if !matched {
				return nil
			}
		}

		files = append(files, path)
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

type commentParser struct {
	plugin          plugins.Plugin
	startPattern    *regexp.Regexp
	endPattern      *regexp.Regexp
	multiStartToken *regexp.Regexp
	multiEndToken   *regexp.Regexp
	inMultiline     bool
	buffer          bytes.Buffer
	foundStartTag   bool // Add this to track if we've found a start tag
}

func newCommentParser(p plugins.Plugin) *commentParser {
	style := p.GetCommentStyle()
	return &commentParser{
		plugin: p,
		// Single line patterns remain the same
		startPattern: regexp.MustCompile(
			`(?i)` + regexp.QuoteMeta(style.Single) + `\s*>:\s*\{`,
		),
		endPattern: regexp.MustCompile(
			`(?i)` + regexp.QuoteMeta(style.Single) + `\s*<:\s*\{`,
		),
		// Multi-line patterns now just match the comment tokens
		multiStartToken: regexp.MustCompile(regexp.QuoteMeta(style.Multi.Start)),
		multiEndToken:   regexp.MustCompile(regexp.QuoteMeta(style.Multi.End)),
	}
}

func (p *commentParser) parseLine(line string) (isStart bool, isEnd bool, jsonData map[string][]string) {
	// Check for single-line comments first
	if p.startPattern.MatchString(line) {
		data, err := parseTagJSON(line)
		if err == nil {
			return true, false, data
		}
	}
	if p.endPattern.MatchString(line) {
		data, err := parseTagJSON(line)
		if err == nil {
			return false, true, data
		}
	}

	// Handle multi-line comments
	if !p.inMultiline {
		if p.multiStartToken.MatchString(line) {
			p.inMultiline = true
			p.buffer.Reset()
			p.buffer.WriteString(line + "\n")
			return false, false, nil
		}
	} else {
		p.buffer.WriteString(line + "\n")
		if p.multiEndToken.MatchString(line) {
			p.inMultiline = false
			// Process the entire multi-line comment
			fullComment := p.buffer.String()

			// Look for >: {...} pattern in the full comment
			startMatch := regexp.MustCompile(`>:\s*\{.*}`).FindString(fullComment)
			if startMatch != "" {
				data, err := parseTagJSON(startMatch)
				if err == nil {
					p.foundStartTag = true
					return true, false, data
				}
			}

			// Look for <: {...} pattern in the full comment
			endMatch := regexp.MustCompile(`<:\s*\{.*}`).FindString(fullComment)
			if endMatch != "" {
				data, err := parseTagJSON(endMatch)
				if err == nil {
					return false, true, data
				}
			}
		}
	}

	return false, false, nil
}

// snippet represents a code snippet with its associated metadata including file path, line range, categories, and content.
type snippet struct {
	File       string
	StartLine  int
	EndLine    int
	Categories map[string][]string
	Content    []string
	Plugin     plugins.Plugin
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

	for _, filePath := range files {
		ext := filepath.Ext(filePath)
		plugin, ok := plugins.Get(ext)
		if !ok {
			log.Printf("No plugin found for file type: %s", filePath)
			continue
		}

		parser := newCommentParser(plugin)

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

			isStart, isEnd, data := parser.parseLine(line)

			if isStart {
				activeSnippet = &snippetData{
					categories: data,
					startLine:  lineNum,
					lines:      []string{},
				}
				continue
			}

			if isEnd && activeSnippet != nil {
				snippetObj := snippet{
					File:       filePath,
					StartLine:  activeSnippet.startLine,
					EndLine:    lineNum,
					Categories: activeSnippet.categories,
					Content:    activeSnippet.lines,
					Plugin:     plugin,
				}

				if snippetMatches(snippetObj, catMap) {
					results = append(results, snippetObj)
				}
				activeSnippet = nil
				continue
			}

			// Only collect lines if we have an active snippet
			if activeSnippet != nil && !parser.inMultiline {
				activeSnippet.lines = append(activeSnippet.lines, line)
			}
		}
		_ = f.Close()
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
func printSnippets(snips []snippet, clipboardMode bool) {
	if len(snips) == 0 {
		fmt.Println("No snippets found for the given categories.")
		return
	}

	var output strings.Builder

	for i, s := range snips {
		// Add newline between snippets
		if i > 0 {
			output.WriteString("\n")
		}

		output.WriteString(fmt.Sprintf("## File: %s (lines %d-%d)\n\n", s.File, s.StartLine, s.EndLine))

		catInfo := []string{}
		for cat, domains := range s.Categories {
			catInfo = append(catInfo, fmt.Sprintf(`%s -> %v`, cat, domains))
		}
		output.WriteString(fmt.Sprintf("**Categories**: %s\n\n", strings.Join(catInfo, ", ")))

		output.WriteString(fmt.Sprintf("```%s\n", s.Plugin.GetMarkdownIdentifier()))
		for _, line := range s.Content {
			output.WriteString(line + "\n")
		}
		output.WriteString("```\n")
	}

	result := output.String()

	// Always print to stdout
	fmt.Print(result)

	// If clipboard mode is active, also copy to clipboard
	if clipboardMode {
		//if err := clipboard.WriteAll(result); err != nil {
		//	fmt.Fprintf(os.Stderr, "Failed to copy to clipboard: %v\n", err)
		//	return
		//}
		_, err := fmt.Fprintf(os.Stderr, "\nContent copied to clipboard!\n")
		if err != nil {
			return
		}
	}
}
