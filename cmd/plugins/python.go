package plugins

type PythonPlugin struct{}

func init() {
	Register(&PythonPlugin{})
}

func (p *PythonPlugin) GetName() string {
	return "Python"
}

func (p *PythonPlugin) GetExtensions() []string {
	return []string{".py", ".pyc"}
}

func (p *PythonPlugin) GetCommentStyle() CommentStyle {
	return CommentStyle{
		Single: "#",
		Multi: struct {
			Start string
			End   string
		}{
			Start: `"""`,
			End:   `"""`,
		},
	}
}
