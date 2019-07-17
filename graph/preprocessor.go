// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

/* Alias preprocessor:
* The set of elements here are meant to process the alias definition portion of task.yaml
* files. This is done by unmarshaling these elements which will then be added in a hierarchical
* manner. Note the input must still be valid Yaml.
*
* Existing issues: Once the preTask is parsed out and the definitions are resolved, the read in
* yaml will be processed to include the appropriate aliases, however this will include the previously
* parsed in definitions which is not as efficient.
*
 */

package graph

import (
	"errors"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v2"
)

var (
	errImproperAlias             = errors.New("alias can only use alphanumeric characters")
	errMissingAlias              = errors.New("no alias was specified")
	errMissingMatch              = errors.New("match for Alias may not be empty")
	errUnknownAlias              = errors.New("unknown Alias")
	errImproperDirectiveOverload = errors.New("$ directive can only be overwritten by a single character")
	errImproperKeyName           = errors.New("alias key names only support alphanumeric characters and _ , - , ! characters")
	directive                    = '$'
)

// PreTask intermediate step for processing before complete unmarshall
type PreTask struct {
	AliasSrc  []*string         `yaml:"alias-src"`
	AliasMap  map[string]string `yaml:"alias"`
	directive rune
}

// Prevents recursive definitions from occuring
func (preTask *PreTask) resolveMapAndValidate() error {
	//Set directive from Map
	preTask.directive = directive
	if _, ok := preTask.AliasMap[string(directive)]; ok {
		if len(preTask.AliasMap[string(directive)]) != 1 {
			return errImproperDirectiveOverload
		}
		preTask.directive = rune(preTask.AliasMap[string(directive)][0])
	}
	// Values may support all characters, no escaping and so forth necessary
	for key := range preTask.AliasMap {
		matched, err := regexp.MatchString("\\A[a-z, A-Z, 0-9, _,-,!]+\\z", key)
		if err != nil {
			return err
		}
		if !matched {
			return errImproperKeyName
		}
	}
	return nil
}

/* Loads in all Aliases defined as being a part of external resources. */
func (preTask *PreTask) loadExternalAlias() error {
	// Iterating in reverse to easily and efficiently handle hierarchy. The later
	// declared the higher in the hierarchy of alias definitions.
	for i := len(preTask.AliasSrc) - 1; i >= 0; i-- {
		aliasURI := *preTask.AliasSrc[i]
		if strings.HasPrefix(aliasURI, "https://") || strings.HasPrefix(aliasURI, "http://") { // Rewrite in nice case insensitive regex
			if err := addAliasFromRemote(preTask, aliasURI); err != nil {
				return err
			}
		} else {
			if err := addAliasFromFile(preTask, aliasURI); err != nil {
				return err
			}
		}
	}
	return nil
}

/* Fetches and Parses out remote alias files and adds their content
to the passed in PreTask. Note alias definitions already in preTask
will not be overwritten. */
func addAliasFromRemote(preTask *PreTask, url string) error {
	remoteClient := http.Client{
		Timeout: time.Second * 2, // Maximum of 2 secs
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	res, getErr := remoteClient.Do(req)
	if getErr != nil {
		return getErr
	}

	data, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		return readErr
	}

	return readAliasFromBytes(data, preTask)
}

/* Parses out local alias files and adds their content to the passed in
   PreTask. Note alias definitions already in preTask will not be
   overwritten. */
func addAliasFromFile(preTask *PreTask, fileURI string) error {

	data, fileReadingError := ioutil.ReadFile(fileURI)
	if fileReadingError != nil {
		return fileReadingError
	}
	return readAliasFromBytes(data, preTask)
}

/* Parses out alias  definitions from a given bytes array and appends
   them to the PreTask. Note alias definitions already in preTask will
   not be overwritten even if present in the array. */
func readAliasFromBytes(data []byte, preTask *PreTask) error {

	aliasMap := &map[string]string{}

	if err := yaml.Unmarshal(data, aliasMap); err != nil {
		return err
	}

	for key, value := range *aliasMap {
		if _, ok := preTask.AliasMap[key]; !ok {
			preTask.AliasMap[key] = value
		}
	}
	return nil
}

// PreprocessString handles managing alias definitions from a provided string definitions expected to be in JSON format.
func PreprocessString(preTask *PreTask, str string) (string, error) {
	//preTask.loadGlobalDefinitions TODO?

	// Load Remote/Local alias definitions
	if externalDefinitionErr := preTask.loadExternalAlias(); externalDefinitionErr != nil {
		return "", externalDefinitionErr
	}
	//Validate alias definitions
	if improperFormatErr := preTask.resolveMapAndValidate(); improperFormatErr != nil {
		return "", improperFormatErr
	}
	var out strings.Builder
	var command strings.Builder
	ongoingCmd := false

	// Search and Replace
	for _, char := range str {
		if ongoingCmd {
			//Maybe just checking if non alphanumeric, only allow alpha numeric aliases?
			if strings.Contains(")}/ .,;]&|'~\n\t", string(char)) { // Delineates the end of an alias
				resolvedCommand, commandPresent := preTask.AliasMap[command.String()]
				if !commandPresent {
					return "", errUnknownAlias
				}

				out.WriteString(resolvedCommand)
				if char != preTask.directive {
					ongoingCmd = false
					out.WriteRune(char)
				}
				command.Reset()

			} else {
				command.WriteRune(char)
			}
		} else if char == preTask.directive {

			if ongoingCmd { // Escape character triggered
				out.WriteRune(preTask.directive)
				ongoingCmd = false
				continue
			}

			ongoingCmd = true
			continue
		} else {
			out.WriteRune(char)
		}
	}

	return out.String(), nil
}

// PreprocessBytes Handles files or byte encoded data that can be parsed through pre processing
func PreprocessBytes(data []byte) ([]byte, error) {
	preTask := &PreTask{}

	if err := yaml.Unmarshal(data, preTask); err != nil {
		return nil, err
	}

	// Search and Replace
	str := string(data[:])
	parsedStr, err := PreprocessString(preTask, str)
	return []byte(parsedStr), err
}
