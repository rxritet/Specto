package html

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

//go:embed assets/*.html
var embeddedFS embed.FS

// Engine parses and executes html/template templates.
//
// In production (devDir == "") templates are read from the embedded FS
// baked into the binary at compile time via go:embed.
//
// In development (devDir != "") templates are re-read from disk on every
// Render call, giving instant reload without recompiling.
type Engine struct {
	devDir string // empty → use embeddedFS
	funcs  template.FuncMap
	mu     sync.RWMutex // guards prod-cached templates
	cached *template.Template
}

// NewEngine creates a template engine.
//
//	NewEngine("")                       → production, embedded assets
//	NewEngine("internal/web/html/assets") → development, live reload
func NewEngine(devDir string) *Engine {
	e := &Engine{
		devDir: devDir,
		funcs:  defaultFuncMap(),
	}
	return e
}

// Render executes the named template into w with the given data.
func (e *Engine) Render(w io.Writer, name string, data any) error {
	t, err := e.templates()
	if err != nil {
		return fmt.Errorf("parse templates: %w", err)
	}
	if err := t.ExecuteTemplate(w, name, data); err != nil {
		return fmt.Errorf("execute %q: %w", name, err)
	}
	return nil
}

// templates returns the parsed template tree.
// Dev mode re-parses every call; prod mode caches after first parse.
func (e *Engine) templates() (*template.Template, error) {
	if e.devDir != "" {
		return e.parseFromDisk()
	}
	return e.parseFromEmbed()
}

// parseFromEmbed parses templates from the go:embed FS (cached).
func (e *Engine) parseFromEmbed() (*template.Template, error) {
	e.mu.RLock()
	if e.cached != nil {
		defer e.mu.RUnlock()
		return e.cached, nil
	}
	e.mu.RUnlock()

	e.mu.Lock()
	defer e.mu.Unlock()

	// Double-check after acquiring write lock.
	if e.cached != nil {
		return e.cached, nil
	}

	t := template.New("").Funcs(e.funcs)

	err := fs.WalkDir(embeddedFS, "assets", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if filepath.Ext(path) != ".html" {
			return nil
		}
		data, readErr := embeddedFS.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		// Template name = relative path inside assets/, e.g. "layout.html".
		name := path[len("assets/"):]
		if _, err := t.New(name).Parse(string(data)); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	e.cached = t
	return t, nil
}

// parseFromDisk re-reads every .html file under devDir (no caching).
func (e *Engine) parseFromDisk() (*template.Template, error) {
	t := template.New("").Funcs(e.funcs)

	err := filepath.WalkDir(e.devDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if filepath.Ext(path) != ".html" {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		// Derive a stable name relative to devDir.
		rel, _ := filepath.Rel(e.devDir, path)
		rel = filepath.ToSlash(rel)
		if _, err := t.New(rel).Parse(string(data)); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return t, nil
}

// defaultFuncMap returns template helper functions available in all templates.
func defaultFuncMap() template.FuncMap {
	return template.FuncMap{
		"safe": func(s string) template.HTML { return template.HTML(s) },
	}
}
