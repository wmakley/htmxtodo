package view

import (
	"errors"
	"fmt"
	"github.com/Masterminds/sprig/v3"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"html/template"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	// CompileOnRender will cause templates will be compiled on every request. Good for development.
	CompileOnRender bool
	// FS allows use of embedded file system. If nil, os.DirFS(".") will be used.
	FS FS
	// Path to the views directory containing templates (inside FS).
	Path string
}

// FS has all file system interfaces needed by view.
type FS interface {
	fs.FS
	fs.ReadDirFS
}

const (
	Suffix        = ".tmpl"
	Shared        = "shared"
	Layouts       = "layouts"
	DefaultLayout = "main" + Suffix
)

func New(config Config) fiber.Views {
	var internalFS FS
	if config.FS == nil || config.CompileOnRender {
		internalFS = os.DirFS(".").(FS)
	} else {
		internalFS = config.FS
	}

	v := view{
		config:                 &config,
		fs:                     internalFS,
		views:                  make(map[string]*template.Template),
		viewPartialCollections: make(map[string]*template.Template),
	}
	//v.init(true)
	return &v
}

type view struct {
	config                 *Config
	fs                     FS
	views                  map[string]*template.Template
	viewPartialCollections map[string]*template.Template
	sharedPartials         *template.Template
}

func (v *view) Load() error {
	log.Debug("view.Load()")

	sharedPartials := v.loadSharedPartials()
	layouts := v.listLayouts()
	viewDirs := v.listViewDirs()

	mainLayout := filepath.Join(v.config.Path, Layouts, DefaultLayout)

	// File system prefix to strip off to generate view names.
	stripPrefix := v.config.Path + string(os.PathSeparator)

	for _, viewDir := range viewDirs {
		views, partials := v.scanViewDir(viewDir)

		if len(partials) > 0 {
			key := strings.Replace(viewDir, v.config.Path, "", 1)
			t := template.New(key).Funcs(sprig.FuncMap())
			v.viewPartialCollections[key] = template.Must(t.ParseFS(v.fs, partials...))
		}

		for _, view := range views {
			allTemplates := make([]string, 0, len(sharedPartials)+len(layouts)+len(partials)+1)
			allTemplates = append(allTemplates, mainLayout)
			allTemplates = append(allTemplates, view)
			allTemplates = append(allTemplates, sharedPartials...)
			allTemplates = append(allTemplates, partials...)

			tmpl := template.New(DefaultLayout).Funcs(v.FuncMap())
			tmpl = template.Must(tmpl.ParseFS(v.fs, allTemplates...))

			viewName := strings.Replace(view, stripPrefix, "", 1)
			v.views[viewName] = tmpl

			log.Debug("view.Load(): loaded template:", viewName)
		}
	}

	return nil
}

// Render renders a template by name. Name should be the path to the template file, relative to the views directory.
// If the name has no suffix, Suffix will be assumed.
func (v *view) Render(w io.Writer, name string, data interface{}, layouts ...string) error {
	key, err := parseTemplateName(name)
	if err != nil {
		return err
	}

	if v.config.CompileOnRender {
		if err := v.Load(); err != nil {
			panic(err)
		}
	}

	if key.isShared {
		if err := v.sharedPartials.ExecuteTemplate(w, key.name, data); err != nil {
			var tplError *template.Error
			if errors.As(err, &tplError) && tplError.ErrorCode == template.ErrNoSuchTemplate {
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

	tmpl, ok := v.views[key.FullPath()]
	if !ok {
		return fmt.Errorf("template not found: %s", name)
	}

	if err := tmpl.Execute(w, data); err != nil {
		panic(err)
	}

	return nil
}

func parseTemplateName(name string) (viewKey, error) {
	parts := strings.SplitN(name, "/", 2)
	if strings.Contains(parts[1], "/") {
		return viewKey{}, errors.New("template name parse error: subdirectories not supported")
	}

	nameWithExtension := parts[1]
	if !strings.Contains(nameWithExtension, ".") {
		nameWithExtension += Suffix
	}

	return viewKey{
		viewDir:   parts[0],
		name:      nameWithExtension,
		isPartial: strings.HasPrefix(parts[1], "_"),
		isShared:  parts[0] == Shared,
	}, nil
}

type viewKey struct {
	viewDir   string
	name      string
	isPartial bool
	isShared  bool
}

func (k viewKey) FullPath() string {
	return filepath.Join(k.viewDir, k.name)
}

func (v *view) FuncMap() template.FuncMap {
	return sprig.FuncMap()
}

// loadSharedPartials loads all partials in the "shared" view directory and returns their paths.
func (v *view) loadSharedPartials() []string {
	partials := v.listPartials(Shared)

	t := template.New(partials[0]).Funcs(v.FuncMap())

	v.sharedPartials = template.Must(t.ParseFS(v.fs, partials...))
	return partials
}

func (v *view) listLayouts() []string {
	layoutsPath := filepath.Join(v.config.Path, Layouts)
	entries, err := v.fs.ReadDir(layoutsPath)
	if err != nil {
		panic(err)
	}

	layouts := make([]string, 0, len(entries))
	for _, f := range entries {
		if isTemplate(f) {
			layouts = append(layouts, filepath.Join(layoutsPath, f.Name()))
		}
	}
	return layouts
}

// scanViewDir returns a list of all views and partials in the given view directory.
func (v *view) scanViewDir(viewDir string) (views []string, partials []string) {
	basePath := filepath.Join(v.config.Path, viewDir)
	entries, err := v.fs.ReadDir(basePath)
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
			partials = append(partials, filepath.Join(basePath, f.Name()))
		} else {
			views = append(views, filepath.Join(basePath, f.Name()))
		}
	}
	return views, partials
}

// listViewDirs returns a list of all view directories other than "shared" or "layouts"
func (v *view) listViewDirs() []string {
	entries, err := v.fs.ReadDir(v.config.Path)
	if err != nil {
		panic(err)
	}

	viewDirs := make([]string, 0, len(entries))
	for _, f := range entries {
		if f.IsDir() && f.Name() != Shared && f.Name() != Layouts {
			viewDirs = append(viewDirs, f.Name())
		}
	}
	return viewDirs
}

// listPartials returns a list of all partials in the given view directory.
func (v *view) listPartials(viewDir string) []string {
	basePath := filepath.Join(v.config.Path, viewDir)
	entries, err := v.fs.ReadDir(basePath)
	if err != nil {
		panic(err)
	}

	partials := make([]string, 0, len(entries))
	for _, f := range entries {
		if isPartial(f) {
			partials = append(partials, filepath.Join(basePath, f.Name()))
		}
	}
	return partials
}

// isTemplate returns true if the path has a .tmpl extension.
func isTemplate(path fs.DirEntry) bool {
	return strings.HasSuffix(path.Name(), Suffix)
}

// isPartial returns true if the path is a partial template file.
// (starts with _ and has a .tmpl extension)
func isPartial(path fs.DirEntry) bool {
	return isTemplate(path) && strings.HasPrefix(path.Name(), "_")
}
