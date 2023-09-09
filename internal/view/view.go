package view

import (
	"embed"
	"errors"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"html/template"
	"io"
	"io/fs"
	"os"
	"strings"
)

func New(config Config) fiber.Views {
	var internalFS FS
	if config.CompileOnRender {
		internalFS = os.DirFS(".").(FS)
	} else {
		internalFS = config.EmbedFS
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

// FS has all file system interfaces needed by view.
type FS interface {
	fs.FS
	fs.ReadDirFS
}

type Config struct {
	// If true, templates will be compiled on every request. Good for development.
	CompileOnRender bool
	Path            string
	EmbedFS         embed.FS
}

const Dir = "views/"
const Suffix = ".tmpl"

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

	for _, viewDir := range viewDirs {
		views, partials := v.scanViewDir(viewDir)

		if len(partials) > 0 {
			key := strings.Replace(viewDir, Dir, "", 1)
			v.viewPartialCollections[key] = template.Must(template.ParseFS(v.fs, partials...))
		}

		for _, view := range views {
			allTemplates := make([]string, 0, len(sharedPartials)+len(layouts)+len(partials)+1)
			allTemplates = append(allTemplates, Dir+"layouts/main"+Suffix)
			allTemplates = append(allTemplates, view)
			allTemplates = append(allTemplates, sharedPartials...)
			allTemplates = append(allTemplates, partials...)

			tmpl := template.Must(template.ParseFS(v.fs, allTemplates...))

			viewName := strings.Replace(view, Dir, "", 1)
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
		isShared:  parts[0] == "shared",
	}, nil
}

type viewKey struct {
	viewDir   string
	name      string
	isPartial bool
	isShared  bool
}

func (k viewKey) FullPath() string {
	return k.viewDir + "/" + k.name
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

// isTemplate returns true if the path has a .tmpl extension.
func isTemplate(path fs.DirEntry) bool {
	return strings.HasSuffix(path.Name(), Suffix)
}

// isPartial returns true if the path is a partial template file.
// (starts with _ and has a .tmpl extension)
func isPartial(path fs.DirEntry) bool {
	return isTemplate(path) && strings.HasPrefix(path.Name(), "_")
}
