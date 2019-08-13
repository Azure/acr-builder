// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

/* Alias preprocessor:
* The set of elements here are meant to process the alias definition portion of task.yaml
* files. This is done by unmarshaling these elements which will then be added in a hierarchical
* manner. Note the input must still be valid Yaml.
*
* TODO:
* Acquire list of globally accessible image endpoints
 */

package graph

import (
	"bufio"
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v2"
)

var (
	errImproperDirectiveLength = errors.New("$ directive can only be overwritten by a single character")
	errImproperKeyName         = errors.New("alias key names only support alphanumeric characters")
	errImproperDirectiveChoice = errors.New("overwritten directives may not be alphanumeric characters")
	directive                  = '$'
	re                         = regexp.MustCompile("\\A[a-zA-Z0-9]+\\z")
)

// Alias intermediate step for processing before complete unmarshall
type Alias struct {
	AliasSrc  []*string         `yaml:"src"`
	AliasMap  map[string]string `yaml:"values"`
	directive rune
}

// Prevents recursive definitions from occuring
func (alias *Alias) resolveMapAndValidate() error {
	//Set directive from Map
	alias.directive = directive
	if value, ok := alias.AliasMap[string(directive)]; ok {
		if len(value) != 1 {
			return errImproperDirectiveLength
		}

		if matched := re.MatchString(value); matched {
			return errImproperDirectiveChoice
		}

		alias.directive = rune(value[0])
	}

	// Values may support all characters, no escaping and so forth necessary
	for key := range alias.AliasMap {
		matched := re.MatchString(key)

		if !matched && key != string(directive) {
			return errImproperKeyName
		}
	}
	return nil
}

/* Loads in all Aliases defined as being a part of external resources. */
func (alias *Alias) loadExternalAlias() error {
	// Iterating in reverse to easily and efficiently handle hierarchy. The later
	// declared the higher in the hierarchy of alias definitions.
	for i := len(alias.AliasSrc) - 1; i >= 0; i-- {
		aliasURI := *alias.AliasSrc[i]
		if strings.HasPrefix(aliasURI, "https://") || strings.HasPrefix(aliasURI, "http://") { // Rewrite in nice case insensitive regex
			if err := addAliasFromRemote(alias, aliasURI); err != nil {
				return err
			}
		} else {
			if err := addAliasFromFile(alias, aliasURI); err != nil {
				return err
			}
		}
	}
	return nil
}

/* Fetches and Parses out remote alias files and adds their content
to the passed in Alias. Note alias definitions already in alias
will not be overwritten. */
func addAliasFromRemote(alias *Alias, url string) error {
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

	if res.StatusCode%100 != 2 {
		httpErr, _ := ioutil.ReadAll(res.Body)
		return errors.New(string(httpErr))
	}

	defer res.Body.Close()

	data, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		return readErr
	}

	return readAliasFromBytes(data, alias)
}

/* Parses out local alias files and adds their content to the passed in
Alias. Note alias definitions already in alias will not be
overwritten. */
func addAliasFromFile(alias *Alias, fileURI string) error {

	data, fileReadingError := ioutil.ReadFile(fileURI)
	if fileReadingError != nil {
		return fileReadingError
	}
	return readAliasFromBytes(data, alias)
}

/* Parses out alias  definitions from a given bytes array and appends
them to the Alias. Note alias definitions already in alias will
not be overwritten even if present in the array. */
func readAliasFromBytes(data []byte, alias *Alias) error {

	aliasMap := &map[string]string{}

	if err := yaml.Unmarshal(data, aliasMap); err != nil {
		return err
	}

	for key, value := range *aliasMap {
		if _, ok := alias.AliasMap[key]; !ok {
			alias.AliasMap[key] = value
		}
	}
	return nil
}

// PreprocessString handles managing alias definitions from a provided string definitions expected to be in JSON format.
func preprocessString(alias *Alias, str string) (string, bool, error) {
	//alias.loadGlobalDefinitions TODO?

	// Load Remote/Local alias definitions
	if externalDefinitionErr := alias.loadExternalAlias(); externalDefinitionErr != nil {
		return "", false, externalDefinitionErr
	}
	//Validate alias definitions
	if improperFormatErr := alias.resolveMapAndValidate(); improperFormatErr != nil {
		return "", false, improperFormatErr
	}
	var out strings.Builder
	var command strings.Builder
	ongoingCmd := false
	changed := false

	// Search and Replace all strings with $
	for _, char := range str {
		if ongoingCmd {
			if matched := re.MatchString(string(char)); !matched { // Delineates the end of an alias
				resolvedCommand, commandPresent := alias.AliasMap[command.String()]
				if !commandPresent {
					return "", false, errors.New("unknown Alias: " + command.String())
				}

				out.WriteString(resolvedCommand)
				changed = true
				if char != alias.directive {
					ongoingCmd = false
					out.WriteRune(char)
				}
				command.Reset()

			} else {
				command.WriteRune(char)
			}
		} else if char == alias.directive {

			if ongoingCmd { // Escape character triggered
				out.WriteRune(alias.directive)
				ongoingCmd = false
				continue
			}
			ongoingCmd = true
		} else {
			out.WriteRune(char)
		}
	}

	return out.String(), changed, nil
}

// PreprocessBytes Handles byte encoded data that can be parsed through pre processing
func preprocessBytes(data []byte) ([]byte, Alias, bool, error) {
	type Wrapper struct {
		Alias Alias `yaml:"alias,omitempty"`
	}
	wrap := &Wrapper{}
	aliasData, remainingData, err := basicAliasSeparation(data)
	if errUnMarshal := yaml.Unmarshal(aliasData, wrap); errUnMarshal != nil {
		return data, Alias{}, false, errUnMarshal
	}

	alias := &wrap.Alias
	if alias.AliasMap == nil {
		//Nothing to change
		if alias.AliasSrc == nil {
			return data, *alias, false, nil
		}
		//Alias Src defined. Guarantees alias map can be populated
		alias.AliasMap = make(map[string]string)
	}

	// Search and Replace
	str := string(remainingData)
	parsedStr, changed, err := preprocessString(alias, str)

	return []byte(parsedStr), *alias, changed, err
}

// processSteps Will resolve image names in steps that are aliased without using $.
// Invoked after resolving $
func processSteps(alias *Alias, task *Task) {
	for i, step := range task.Steps {
		parts := strings.Split(step.Cmd, " ")
		if _, ok := alias.AliasMap[parts[0]]; ok {
			// Image name should always go first
			parts[0] = alias.AliasMap[parts[0]]
			task.Steps[i].Cmd = strings.Join(parts, " ")
		}
	}
}

func basicAliasSeparation(data []byte) ([]byte, []byte, error) {
	reader := bytes.NewReader(data)
	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanLines)

	var aliasBuffer bytes.Buffer
	var buffer bytes.Buffer

	inside := false
	aliasRe := regexp.MustCompile(`\Aalias\s*:.*\z`)
	genericTopLevelRe := regexp.MustCompile(`\A[^\s:]+[^:]*:.*\z`)
	commentRe := regexp.MustCompile(`\A#.*`)

	for scanner.Scan() {
		text := scanner.Text()
		if matched := commentRe.MatchString(text); matched {
			continue
		}
		if matched := aliasRe.MatchString(text); matched && !inside {
			inside = true
		} else if matched := genericTopLevelRe.MatchString(text); matched && inside {
			inside = false
		}

		if inside {
			aliasBuffer.WriteString(text + "\n")
		} else {
			buffer.WriteString(text + "\n")
		}
	}
	return aliasBuffer.Bytes(), buffer.Bytes(), nil
}
