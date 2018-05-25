package templating

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strings"
	"text/template"

	"github.com/Azure/acr-builder/util"
	"github.com/Masterminds/sprig"
)

// Engine is a wrapper around Go templates.
type Engine struct {
	FuncMap    template.FuncMap
	StrictMode bool
}

// New creates a new engine.
func New() *Engine {
	fm := FuncMap()
	return &Engine{
		FuncMap: fm,
	}
}

// FuncMap returns a FuncMap representing all of the functionality of the engine.
func FuncMap() template.FuncMap {
	return sprig.TxtFuncMap()
}

// RenderAllTemplates renders all of a job's templates with the specified default values.
func (e *Engine) RenderAllTemplates(job *Job, values Values) (map[string]string, error) {
	return e.Render(job, values, nil)
}

// Render renders a job's templates, skipping any of those not specified in keep.
func (e *Engine) Render(job *Job, values Values, keep map[string]bool) (map[string]string, error) {
	if job == nil {
		return nil, errors.New("job is required")
	}

	if values == nil {
		return nil, errors.New("values is required")
	}

	templates := map[string]renderableTemplate{}

	for _, t := range job.Templates {
		// If keep is nil, keep everything
		if keep == nil || keep[t.Name] {
			templates[t.Name] = renderableTemplate{
				template: string(t.Data),
				values:   values,
			}
		} else {
			fmt.Printf("Skipped rendering for %s\n", t.Name)
		}
	}

	return e.render(templates)
}

func (e *Engine) render(templates map[string]renderableTemplate) (rendered map[string]string, err error) {
	// If a template panics, recover the engine.
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Template rendering recovered. Value: %v\n", r)
		}
	}()

	t := template.New("_")
	if e.StrictMode {
		t.Option("missingkey=error")
	} else {
		// NB: zero will attempt to add default values for types it knows.
		// It still emits <no value> for others. This is corrected below.
		t.Option("missingkey=zero")
	}

	// Gather all the templates from the job's files.
	files := []string{}

	// Sort the templates for consistent ordering
	keys := sortTemplates(templates)
	for _, k := range keys {
		r := templates[k]
		t = t.New(k).Funcs(e.FuncMap)

		if _, err := t.Parse(r.template); err != nil {
			return map[string]string{}, fmt.Errorf("Failed to parse template: %s. Err: %v", k, err)
		}

		files = append(files, k)
	}

	// Render all of the templates.
	rendered = make(map[string]string, len(files))
	var buf bytes.Buffer
	for _, f := range files {
		if err := t.ExecuteTemplate(&buf, f, templates[f].values); err != nil {
			return map[string]string{}, fmt.Errorf("Failed to execute template: %s. Err: %v", f, err)
		}

		// NB: handle `missingkey=zero` by removing the string.
		rendered[f] = strings.Replace(buf.String(), "<no value>", "", -1)
		buf.Reset()
	}

	return rendered, nil
}

// sortTemplates sorts the provided map of templates by path length.
func sortTemplates(templates map[string]renderableTemplate) []string {
	ret := make([]string, len(templates))
	i := 0
	for key := range templates {
		ret[i] = key
		i++
	}
	sort.Sort(sort.Reverse(util.SortablePathLen(ret)))
	return ret
}

type renderableTemplate struct {
	template string
	values   Values
}
