package plugins

// CommentStyle represents how comments are formatted in a language
type CommentStyle struct {
	// Single line comment prefix (e.g., "//", "#")
	Single string
	// Multi-line comment start and end tokens (e.g., ["/*", "*/"])
	Multi struct {
		Start string
		End   string
	}
}

// Plugin defines the interface that all language plugins must implement
type Plugin interface {
	// GetName returns the name of the language
	GetName() string
	// GetExtensions returns the file extensions this plugin handles
	GetExtensions() []string
	// GetCommentStyle returns the comment style for this language
	GetCommentStyle() CommentStyle
	GetMarkdownIdentifier() string
}

// registry stores all available plugins
var registry = make(map[string]Plugin)

// Register adds a plugin to the registry
func Register(p Plugin) {
	for _, ext := range p.GetExtensions() {
		registry[ext] = p
	}
}

// Get returns the appropriate plugin for a given file extension
func Get(extension string) (Plugin, bool) {
	plugin, exists := registry[extension]
	return plugin, exists
}

// ListExtensions returns all supported file extensions
func ListExtensions() []string {
	extensions := make([]string, 0, len(registry))
	for ext := range registry {
		extensions = append(extensions, ext)
	}
	return extensions
}
