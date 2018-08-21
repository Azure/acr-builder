// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package templating

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"strings"
	"text/template"

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
func (e *Engine) Render(t *Template, values Values) (string, error) {
	if t == nil {
		return "", errors.New("template is required")
	}
	if values == nil {
		return "", errors.New("values is required")
	}

	rt := renderableTemplate{
		name:     t.Name,
		template: string(t.Data),
		values:   values,
	}

	return e.render(rt)
}

func (e *Engine) render(rt renderableTemplate) (rendered string, err error) {
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

	t = t.New(rt.name).Funcs(e.FuncMap)
	if _, err := t.Parse(rt.template); err != nil {
		return "", fmt.Errorf("Failed to parse template: %s. Err: %v", rt.name, err)
	}

	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, rt.name, rt.values); err != nil {
		return "", fmt.Errorf("Failed to execute template: %s. Err: %v", rt.name, err)
	}

	// NB: handle `missingkey=zero` by removing the string.
	rendered = strings.Replace(buf.String(), "<no value>", "", -1)
	return rendered, nil
}

type renderableTemplate struct {
	name     string
	template string
	values   Values
}
