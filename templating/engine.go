// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package templating

import (
	"bytes"
	"errors"
	"fmt"
	"log"
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

// NewEngine creates a new engine.
func NewEngine() *Engine {
	fm := FuncMap()
	return &Engine{
		FuncMap: fm,
	}
}

// FuncMap returns a FuncMap representing all of the functionality of the engine.
func FuncMap() template.FuncMap {
	return sprig.TxtFuncMap()
}

// Render renders a template.
func (e *Engine) Render(t *Template, values Values) (map[string]string, error) {
	if t == nil {
		return nil, errors.New("template is required")
	}

	if values == nil {
		return nil, errors.New("values is required")
	}

	templates := map[string]renderableTemplate{}
	templates[t.Name] = renderableTemplate{
		template: string(t.Data),
		values:   values,
	}

	return e.render(templates)
}

func (e *Engine) render(templates map[string]renderableTemplate) (rendered map[string]string, err error) {
	// If a template panics, recover the engine.
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Template rendering recovered. Value: %v\n", r)
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

	// Gather all the template filenames.
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
