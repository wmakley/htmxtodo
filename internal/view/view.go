package view

import (
	"embed"
	"errors"
	"fmt"
	"github.com/labstack/echo/v4"
	"html/template"
	"io"
	"io/fs"
	"log"
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

// New is a embeddedTemplateRenderer that compiles templates on every request. Good for development.
func New(config *Config) echo.Renderer {
	if config == nil {
		panic("config is nil")
	}

	var internalFS FS
	if config.CompileOnRender {
		internalFS = os.DirFS(".").(FS)
	} else {
		internalFS = config.EmbedFS
	}

	v := view{
		config:                 config,
		fs:                     internalFS,
		views:                  make(map[string]*template.Template),
		viewPartialCollections: make(map[string]*template.Template),
	}
	v.init(true)
	return &v
}

// FS has all file system interfaces needed by view.
type FS interface {
	fs.FS
	//fs.ReadFileFS
	fs.ReadDirFS
}

type Config struct {
	// If true, templates will be compiled on every request. Good for development.
	CompileOnRender bool
	Path            string
	EmbedFS         embed.FS
}

type view struct {
	config                 *Config
	fs                     FS
	views                  map[string]*template.Template
	viewPartialCollections map[string]*template.Template
	sharedPartials         *template.Template
}

func (v *view) init(debug bool) {
	sharedPartials := v.loadSharedPartials()
	layouts := v.listLayouts()
	viewDirs := v.listViewDirs()

	for _, viewDir := range viewDirs {
		views, partials := v.scanViewDir(viewDir)

		if len(partials) > 0 {
			key := strings.Replace(viewDir, "views/", "", 1)
			v.viewPartialCollections[key] = template.Must(template.ParseFS(v.fs, partials...))
		}

		for _, view := range views {
			allTemplates := make([]string, 0, len(sharedPartials)+len(layouts)+len(partials)+1)
			allTemplates = append(allTemplates, "views/layouts/main.html")
			allTemplates = append(allTemplates, view)
			allTemplates = append(allTemplates, sharedPartials...)
			allTemplates = append(allTemplates, partials...)

			tmpl := template.Must(template.ParseFS(v.fs, allTemplates...))

			viewName := strings.Replace(view, "views/", "", 1)
			v.views[viewName] = tmpl

			if debug {
				log.Println("loaded template:", viewName)
			}
		}
	}
}

func (v *view) mustParseKey(key string) viewKey {
	parts := strings.SplitN(key, "/", 2)
	if strings.Contains(parts[1], "/") {
		panic("subdirectories not supported")
	}
	return viewKey{
		viewDir:   parts[0],
		name:      parts[1],
		isPartial: strings.HasPrefix(parts[1], "_"),
		isShared:  parts[0] == "shared",
	}
}

type viewKey struct {
	viewDir   string
	name      string
	isPartial bool
	isShared  bool
}

// loadSharedPartials loads all partials in the "shared" view directory and returns their paths.
func (v *view) loadSharedPartials() []string {
	partials := v.listPartials("shared")

	v.sharedPartials = template.Must(template.ParseFS(v.fs, partials...))
	return partials
}

func (v *view) listLayouts() []string {
	entries, err := v.fs.ReadDir("views/layouts")
	if err != nil {
		panic(err)
	}

	layouts := make([]string, 0, len(entries))
	for _, f := range entries {
		if isTemplate(f) {
			layouts = append(layouts, "views/layouts/"+f.Name())
		}
	}
	return layouts
}

// scanViewDir returns a list of all views and partials in the given view directory.
func (v *view) scanViewDir(viewDir string) (views []string, partials []string) {
	entries, err := v.fs.ReadDir("views/" + viewDir)
	if err != nil {
		panic(err)
	}

	views = make([]string, 0, len(entries))
	partials = make([]string, 0, len(entries))

	for _, f := range entries {
		if !isTemplate(f) {
			continue
		}

		if isPartial(f) {
			partials = append(partials, "views/"+viewDir+"/"+f.Name())
		} else {
			views = append(views, "views/"+viewDir+"/"+f.Name())
		}
	}
	return views, partials
}

// listViewDirs returns a list of all view directories other than "shared" or "layouts"
func (v *view) listViewDirs() []string {
	entries, err := v.fs.ReadDir("views")
	if err != nil {
		panic(err)
	}

	viewDirs := make([]string, 0, len(entries))
	for _, f := range entries {
		if f.IsDir() && f.Name() != "shared" && f.Name() != "layouts" {
			viewDirs = append(viewDirs, f.Name())
		}
	}
	return viewDirs
}

// listPartials returns a list of all partials in the given view directory.
func (v *view) listPartials(viewDir string) []string {
	entries, err := v.fs.ReadDir("views/" + viewDir)
	if err != nil {
		panic(err)
	}

	partials := make([]string, 0, len(entries))
	for _, f := range entries {
		if isPartial(f) {
			partials = append(partials, "views/"+viewDir+"/"+f.Name())
		}
	}
	return partials
}

//// listViews returns a list of all views in the given view directory.
//func (v *view) listViews(viewDir string) []string {
//	entries, err := v.fs.ReadDir("views/" + viewDir)
//	if err != nil {
//		panic(err)
//	}
//
//	templates := make([]string, 0, len(entries))
//	for _, f := range entries {
//		if isTemplate(f) && !strings.HasPrefix(f.Name(), "_") {
//			templates = append(templates, "views/"+viewDir+"/"+f.Name())
//		}
//	}
//	return templates
//}

// isTemplate returns true if the path has a .html extension.
func isTemplate(path fs.DirEntry) bool {
	return strings.HasSuffix(path.Name(), ".html")
}

// isPartial returns true if the path is a partial template file.
// (starts with _ and has a .html extension)
func isPartial(path fs.DirEntry) bool {
	return isTemplate(path) && strings.HasPrefix(path.Name(), "_")
}

func (v *view) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	key := v.mustParseKey(name)

	if v.config.CompileOnRender {
		v.init(false)
	}

	if key.isShared {
		if err := v.sharedPartials.ExecuteTemplate(w, key.name, data); err != nil {
			if tplErr, ok := err.(*template.Error); ok && tplErr.ErrorCode == template.ErrNoSuchTemplate {
				return fmt.Errorf("template not found: %s", name)
			} else {
				panic(err)
			}
		}
		return nil
	}

	if key.isPartial {
		tmpl, ok := v.viewPartialCollections[key.viewDir]
		if !ok {
			return fmt.Errorf("template not found: %s", name)
		}
		if err := tmpl.ExecuteTemplate(w, key.name, data); err != nil {
			panic(err)
		}
		return nil
	}

	tmpl, ok := v.views[name]
	if !ok {
		return fmt.Errorf("template not found: %s", name)
	}

	if err := tmpl.Execute(w, data); err != nil {
		panic(err)
	}

	return nil
}
