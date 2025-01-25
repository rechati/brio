package plugins

type TypeScriptPlugin struct{}

func init() {
	Register(&TypeScriptPlugin{})
}

func (p *TypeScriptPlugin) GetName() string {
	return "TypeScript"
}

func (p *TypeScriptPlugin) GetExtensions() []string {
	return []string{".ts", ".tsx"}
}

func (p *TypeScriptPlugin) GetCommentStyle() CommentStyle {
	return CommentStyle{
		Single: "//",
		Multi: struct {
			Start string
			End   string
		}{
			Start: "/*",
			End:   "*/",
		},
	}
}

func (p *TypeScriptPlugin) GetMarkdownIdentifier() string {
	return "typescript"
}
