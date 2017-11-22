package driver

import (
	"fmt"

	build "github.com/Azure/acr-builder/pkg"
)

//////////////////////////////////////////////////////////////////////////////////////////////////
// The functions in this class is an attempt organize logics to decide which object to create
// when given a list of parameters. It helps return errors when there are ambiguities and
// select the right factory to use to create these objects
// We are currently only using it on creating sources but the logic is generic enough
// so we can change the object created to be interface{} for reuse if needed
/////////////////////////////////////////////////////////////////////////////////////////////////

type factory struct {
	name       string
	isSelected bool
	create     func() (build.Source, error)
}

type parameter struct {
	name  string
	value string
}

func newFactory(name string, create func() (build.Source, error), required []parameter, optional []parameter) (*factory, error) {
	result := &factory{name: name, create: create}
	if len(required) > 0 {
		// if one required parameter is provided, all has to be here
		requiredProvided := required[0].value != ""
		for _, p := range required[1:] {
			if requiredProvided != (p.value != "") {
				return nil, fmt.Errorf("Required parameter %s is not given for %s", p.name, name)
			}
		}

		// if requires are not provided, none of the optional parameters should be provided
		if !requiredProvided {
			for _, p := range optional {
				if p.value != "" {
					requiredParamNames := []string{}
					for _, p := range required {
						requiredParamNames = append(requiredParamNames, p.name)
					}
					return nil, fmt.Errorf("Optional parameter %s is given for %s but none of the required parameters: %v were given",
						p.name, name, requiredParamNames)
				}
			}
		}

		result.isSelected = requiredProvided
		return result, nil
	}

	// if no required parameter, the selection of the source is implied by presence of any optional parameter
	for _, p := range optional {
		if p.value != "" {
			result.isSelected = true
			return result, nil
		}
	}
	return result, nil
}

// decide between a list of selections, the first option is the default
func decide(scenarioName string, selections ...*factory) (*factory, error) {
	var selected *factory
	for _, d := range selections {
		if selected == nil {
			if d.isSelected {
				selected = d
			}
		} else {
			if d.isSelected {
				return nil, fmt.Errorf("Ambiguous selection on %s, both %s and %s are selected", scenarioName, selected.name, d.name)
			}
		}
	}

	if selected == nil {
		// selections should never be empty!
		selected = selections[0]
	}
	return selected, nil
}
