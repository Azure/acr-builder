// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package templating

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"reflect"
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

// Base64Encode Overrides the function b64enc in Sprig to provide nil behavior
func Base64Encode(v interface{}) string {
	if v == nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString([]byte(reflect.ValueOf(v).String()))
}

// FuncMap returns a FuncMap representing all of the functionality of the engine.
func FuncMap() template.FuncMap {
	//We are overriding the b64enc function with custom implementation
	modMap := sprig.TxtFuncMap()
	modMap["b64enc"] = Base64Encode
	return modMap
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
	return e.RenderGoTemplate(rt.name, rt.template, rt.values)
}

// RenderGoTemplate renders a go template given template and data.
func (e *Engine) RenderGoTemplate(name string, input string, data interface{}) (rendered string, err error) {
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

	t = t.New(name).Funcs(e.FuncMap)
	if _, err := t.Parse(input); err != nil {
		return "", fmt.Errorf("failed to parse template: %s. Err: %v", name, err)
	}

	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, name, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %s. Err: %v", name, err)
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
