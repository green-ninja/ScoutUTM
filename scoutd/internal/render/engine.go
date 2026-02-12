package render

// Engine renders daemon config files from templates.
type Engine struct {
	templateDir string
	outputDir   string
	devMode     bool
}

// NewEngine creates a template rendering engine.
func NewEngine(templateDir, outputDir string, devMode bool) *Engine {
	return &Engine{
		templateDir: templateDir,
		outputDir:   outputDir,
		devMode:     devMode,
	}
}
