package view

import (
	"embed"
	"errors"
	"fmt"
	"github.com/labstack/echo/v4"
	"html/template"
	"io"
	"os"
	"strings"
)

// NewEmbeddedTemplateRenderer returns a renderer that uses the given embed.FS to pre-compile all templates.
func NewEmbeddedTemplateRenderer(fs embed.FS) echo.Renderer {
	views := make(map[string]*template.Template)

	topLevelEntries, err := fs.ReadDir("views")
	if err != nil {
		panic(err)
	}

	topLevelDirNames := make([]string, 0)
	for _, f := range topLevelEntries {
		if f.IsDir() && f.Name() != "shared" {
			topLevelDirNames = append(topLevelDirNames, f.Name())
		}
	}

	globalSharedTemplateEntries, err := fs.ReadDir("views/shared")
	if err != nil {
		panic(err)
	}

	// Create a list of all global shared templates, to be included in every template
	globalSharedTemplates := make([]string, 0, len(globalSharedTemplateEntries))
	for _, f := range globalSharedTemplateEntries {
		if f.IsDir() {
			continue
		}
		globalSharedTemplates = append(globalSharedTemplates, fmt.Sprintf("views/shared/%s", f.Name()))
	}

	for _, topLevelDirName := range topLevelDirNames {
		entries, err := fs.ReadDir(fmt.Sprintf("views/%s", topLevelDirName))
		if err != nil {
			panic(err)
		}

		// Create a list of all templates in the directory
		templates := make([]string, 0, len(entries))

		for _, f := range entries {
			if f.IsDir() {
				continue
			}
			templates = append(templates, fmt.Sprintf("views/%s/%s", topLevelDirName, f.Name()))
		}

		// Get the paths of all shared templates for the directory
		sharedEntries, err := fs.ReadDir(fmt.Sprintf("views/%s/shared", topLevelDirName))
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				panic(err)
			}
			sharedEntries = make([]os.DirEntry, 0)
		}

		localShared := make([]string, 0, len(sharedEntries))
		for _, f := range sharedEntries {
			if f.IsDir() {
				continue
			}
			localShared = append(localShared, fmt.Sprintf("views/%s/shared/%s", topLevelDirName, f.Name()))
		}

		// Compile each template with all shared templates
		for _, templatePath := range templates {
			allTemplates := make([]string, 0, len(localShared)+len(globalSharedTemplates)+1)
			allTemplates = append(allTemplates, templatePath)

			// TODO: local should override global, not sure what will happen here
			allTemplates = append(allTemplates, localShared...)
			allTemplates = append(allTemplates, globalSharedTemplates...)

			viewName := strings.Replace(templatePath, "views/", "", 1)
			views[viewName] = template.Must(template.ParseFS(fs, allTemplates...))
		}
	}

	return &embeddedTemplateRenderer{
		views: views,
	}
}

type embeddedTemplateRenderer struct {
	views map[string]*template.Template
}

func (t *embeddedTemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	// Panic on all template render errors.
	// They are all hard programmer mistakes that must be fixed.

	tmpl, ok := t.views[name]
	if !ok {
		panic(fmt.Sprintf("template not found: %s", name))
	}

	if err := tmpl.Execute(w, data); err != nil {
		panic(err)
	}

	return nil
}

// NewCompiledOnDemandRenderer is a embeddedTemplateRenderer that compiles templates on every request. Good for development.
func NewCompiledOnDemandRenderer() echo.Renderer {
	return &compiledOnDemandRenderer{}
}

type compiledOnDemandRenderer struct {
}

func (t *compiledOnDemandRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	parts := strings.SplitN(name, "/", 2)
	viewDir := parts[0]
	if strings.Contains(parts[1], "/") {
		panic("subdirectories not supported")
	}

	baseFile := "views/" + name

	allViewDirEntries, err := os.ReadDir("views/" + viewDir)
	if err != nil {
		panic(err)
	}

	globalPartialEntries, err := os.ReadDir("views/shared")
	if err != nil {
		panic(err)
	}
	globalPartials := make([]string, 0, len(globalPartialEntries))
	for _, f := range globalPartialEntries {
		if f.IsDir() {
			continue
		}
		if !strings.HasPrefix(f.Name(), "_") {
			continue
		}
		if !strings.HasSuffix(f.Name(), ".html") {
			continue
		}
		globalPartials = append(globalPartials, "views/shared/"+f.Name())
	}

	viewPartials := make([]string, 0, 5)
	for _, f := range allViewDirEntries {
		if f.IsDir() {
			continue
		}
		if !strings.HasPrefix(f.Name(), "_") {
			continue
		}
		if !strings.HasSuffix(f.Name(), ".html") {
			continue
		}
		viewPartials = append(viewPartials, "views/"+viewDir+"/"+f.Name())
	}

	allTemplates := make([]string, 0, len(globalPartials)+len(viewPartials)+1)
	allTemplates = append(allTemplates, baseFile)
	for _, f := range globalPartials {
		allTemplates = append(allTemplates, f)
	}
	// View partials override global partials
	for _, f := range viewPartials {
		allTemplates = append(allTemplates, f)
	}

	tmpl := template.Must(template.ParseFiles(allTemplates...))

	if err = tmpl.Execute(w, data); err != nil {
		panic(err)
	}

	return nil
}
