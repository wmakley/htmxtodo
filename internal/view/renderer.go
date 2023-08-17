package view

import (
	"embed"
	"fmt"
	"github.com/labstack/echo/v4"
	"html/template"
	"io"
)

// NewEmbeddedTemplateRenderer returns a renderer that uses the given embed.FS to pre-load all templates.
func NewEmbeddedTemplateRenderer(fs embed.FS) echo.Renderer {
	views := make(map[string]*template.Template)

	layout := "views/layouts/master.html"

	views["lists/index"] = template.Must(template.ParseFS(fs, layout, "views/lists/index.html"))
	views["lists/show"] = template.Must(template.ParseFS(fs, layout, "views/lists/show.html"))

	return &embeddedTemplateRenderer{
		views: views,
	}
}

type embeddedTemplateRenderer struct {
	views map[string]*template.Template
}

func (t *embeddedTemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	//fmt.Println("Render:", name)

	tmpl, ok := t.views[name]
	if !ok {
		// easier to find source of a panic
		panic(fmt.Sprintf("template not found: %s", name))
		//return fmt.Errorf("template not found: %s", name)
	}

	return tmpl.Execute(w, data)
}

// NewCompiledOnDemandRenderer is a embeddedTemplateRenderer that compiles templates on every request. Good for development.
func NewCompiledOnDemandRenderer() echo.Renderer {
	return &compiledOnDemandRenderer{}
}

type compiledOnDemandRenderer struct {
}

func (t *compiledOnDemandRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	//fmt.Println("Render:", name)

	layout := "views/layouts/master.html"

	path := fmt.Sprintf("views/%s.html", name)

	tmpl := template.Must(template.ParseFiles(layout, path))

	return tmpl.Execute(w, data)
}
