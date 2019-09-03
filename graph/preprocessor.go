// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

// Alias preprocessor:
// The set of elements here are meant to process the alias definition portion of task.yaml
// files. This is done by unmarshaling these elements which will then be added in a hierarchical
// manner. Note the input must still be valid YAML.

package graph

import (
	"bufio"
	"bytes"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/Azure/acr-builder/util"
	yaml "gopkg.in/yaml.v2"
)

var (
	errImproperDirectiveLength = errors.New("$ directive can only be overwritten by a single character")
	errImproperKeyName         = errors.New("alias key names only support alphanumeric characters")
	errImproperDirectiveChoice = errors.New("overwritten directives may not be alphanumeric characters")
	defaultDirective           = '$'
	aliasFormat                = regexp.MustCompile(`\A[a-zA-Z0-9]+\z`)
)

// Alias intermediate step for processing before complete unmarshal
type Alias struct {
	AliasSrc        []string          `yaml:"src"`
	AliasMap        map[string]string `yaml:"values"`
	DirectiveParsed string            `yaml:"directive"`
	directive       rune
}

// Validates aliases making sure all are alphanumeric
// Additionally sets and validates directive overrides
func (alias *Alias) resolveMapAndValidate() error {
	log.Printf("(resolveMapAndValidate) started")
	// Set directive from Map
	alias.directive = defaultDirective
	if alias.DirectiveParsed != "" {
		val := []rune(alias.DirectiveParsed)
		if len(val) != 1 {
			return errImproperDirectiveLength
		}

		if matched := aliasFormat.MatchString(alias.DirectiveParsed); matched {
			return errImproperDirectiveChoice
		}
		log.Printf("(resolveMapAndValidate) directive overwritten to %c", val[0])
		alias.directive = val[0]
	}

	// Values may support all characters, no escaping and so forth necessary
	for key := range alias.AliasMap {
		matched := aliasFormat.MatchString(key)

		if !matched {
			return errImproperKeyName
		}
	}
	log.Printf("(resolveMapAndValidate) completed")
	return nil
}

// Loads in all Aliases defined as being a part of external resources.
func (alias *Alias) loadExternalAlias() error {
	// Iterating in reverse to easily and efficiently handle hierarchy. The later
	// declared the higher in the hierarchy of alias definitions.
	for i := len(alias.AliasSrc) - 1; i >= 0; i-- {
		aliasURI := alias.AliasSrc[i]
		if util.IsURL(aliasURI) {
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

// Fetches and parses out remote alias files and adds their content
// to the passed in Alias. Note alias definitions already in alias
// will not be overwritten.
func addAliasFromRemote(alias *Alias, url string) error {
	log.Printf("(addAliasFromRemote) called for %s", url)
	remoteClient := &http.Client{
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

	if res.StatusCode > 299 {
		httpErr, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}
		return errors.New(string(httpErr))

	}

	defer res.Body.Close()

	data, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		return readErr
	}
	log.Printf("(addAliasFromRemote) %s acquired successfully", url)
	return readAliasFromBytes(data, alias)
}

// Parses out local alias files and adds their content to the passed in
// Alias. Note alias definitions already in alias will not be
// overwritten.
func addAliasFromFile(alias *Alias, fileURI string) error {
	log.Printf("(addAliasFromFile) called for %s", fileURI)
	data, fileReadingError := ioutil.ReadFile(fileURI)
	if fileReadingError != nil {
		return fileReadingError
	}
	return readAliasFromBytes(data, alias)
}

// Parses out alias definitions from a given bytes array and appends
// them to the Alias. Note alias definitions already in alias will
// not be overwritten even if present in the array.
func readAliasFromBytes(data []byte, alias *Alias) error {
	log.Printf("(readAliasFromBytes) started")
	aliasMap := &map[string]string{}

	if err := yaml.Unmarshal(data, aliasMap); err != nil {
		return err
	}

	for key, value := range *aliasMap {
		if _, ok := alias.AliasMap[key]; !ok {
			alias.AliasMap[key] = value
		}
	}
	log.Printf("(readAliasFromBytes) completed")
	return nil
}

// preprocessString handles the preprocessing (string replacement and resolution)
// of all aliases in an input yaml (passed in as a string). The resolved aliases are
// defined in the input alias file.
func preprocessString(alias *Alias, str string, globalSrc string) (string, bool, error) {
	log.Printf("(readAliasFromBytes) started")

	// Load Remote/Local alias definitions
	if externalDefinitionErr := alias.loadExternalAlias(); externalDefinitionErr != nil {
		return "", false, externalDefinitionErr
	}
	log.Printf("(readAliasFromBytes) external definitions loaded")

	// Load Globally defined aliases
	if globalLoadErr := addAliasFromFile(alias, globalSrc); globalLoadErr != nil {
		return "", false, globalLoadErr
	}
	log.Printf("(readAliasFromBytes) global definitions loaded")
	// Validate alias definitions
	if improperFormatErr := alias.resolveMapAndValidate(); improperFormatErr != nil {
		return "", false, improperFormatErr
	}
	log.Printf("(readAliasFromBytes) alias map validated")

	var out strings.Builder
	var command strings.Builder
	ongoingCmd := false
	changed := false

	log.Printf("(readAliasFromBytes) replacement started")
	// Search and replace all strings with the directive
	for _, char := range str {
		if ongoingCmd {
			if char == alias.directive && command.Len() == 0 { // Escape Character Triggered
				out.WriteRune(alias.directive)
				ongoingCmd = false
				log.Printf("(readAliasFromBytes) escape character triggered")

			} else if !isAlphanumeric(char) { // Delineates the end of an alias
				resolvedCommand, commandPresent := alias.AliasMap[command.String()]
				// If command is not found we assume this to be the expect item itself.
				if !commandPresent {
					log.Printf("(readAliasFromBytes) unknown alias found, resolved to identical value")
					out.WriteString(string(alias.directive) + command.String())
					command.Reset()
				} else {
					log.Printf("(readAliasFromBytes) command resolved %s to %s", command.String(), resolvedCommand)
					out.WriteString(resolvedCommand)
					changed = true
					if char != alias.directive {
						log.Printf("(readAliasFromBytes) back to back alias")
						ongoingCmd = false
						out.WriteRune(char)
					}
					command.Reset()
				}
			} else {
				command.WriteRune(char)
			}
		} else if char == alias.directive {
			ongoingCmd = true
		} else {
			out.WriteRune(char)
		}
	}
	log.Printf("(readAliasFromBytes) completed")

	return out.String(), changed, nil
}

// preprocessBytes handles byte encoded data that can be parsed through pre processing
func preprocessBytes(data []byte, globalSrc string) ([]byte, Alias, bool, error) {
	type Wrapper struct {
		Alias Alias `yaml:"alias,omitempty"`
	}
	log.Printf("(preprocessBytes) separating alias bytes")
	wrap := &Wrapper{}
	aliasData, remainingData := basicAliasSeparation(data)
	log.Printf("(preprocessBytes) unmarshal alias bytes")
	if errUnmarshal := yaml.Unmarshal(aliasData, wrap); errUnmarshal != nil {
		return data, Alias{}, false, errUnmarshal
	}

	alias := &wrap.Alias
	if alias.AliasMap == nil {
		// Alias Src defined. Guarantees alias map can be populated
		alias.AliasMap = make(map[string]string)
	}
	// Search and Replace
	str := string(remainingData)
	parsedStr, changed, err := preprocessString(alias, str, globalSrc)

	log.Printf("(preprocessBytes) completed with: %s", parsedStr)
	return []byte(parsedStr), *alias, changed, err
}

// processSteps Will resolve image names in steps that are aliased without using directive.
// Invoked after resolving all directive using aliases
func processSteps(alias *Alias, task *Task) {
	log.Printf("(preprocessSteps) started")
	for i, step := range task.Steps {
		parts := strings.Split(strings.TrimSpace(step.Cmd), " ")
		if val, ok := alias.AliasMap[parts[0]]; ok {
			// Image name should always go first
			log.Printf("(preprocessSteps) command resolved %s to %s", parts[0], val)
			parts[0] = val
			task.Steps[i].Cmd = strings.Join(parts, " ")
		}
	}
}

// Provides simple separation of the top level items in a yaml file definition, however, need
// to update to be fully compliant (include JSON top level for example). Alternative construction
// of small compiler for this purpose is also under consideration.
func basicAliasSeparation(data []byte) ([]byte, []byte) {
	log.Printf("(basicAliasSeparation) started")
	reader := bytes.NewReader(data)
	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanLines)

	var aliasBuffer bytes.Buffer
	var buffer bytes.Buffer

	inside := false
	aliasFieldName := regexp.MustCompile(`\Aalias\s*:.*\z`)
	genericTopLevelRe := regexp.MustCompile(`\A[^\s:]+[^:]*:.*\z`)
	commentRe := regexp.MustCompile(`\A#.*`)
	for scanner.Scan() {
		text := scanner.Text()
		if matched := commentRe.MatchString(text); matched {
			continue
		}
		if matched := aliasFieldName.MatchString(text); matched && !inside {
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
	log.Printf("(basicAliasSeparation) completed. Alias section size :%d, remainder section size : %d, original size : %d", aliasBuffer.Len(), buffer.Len(), len(data))
	return aliasBuffer.Bytes(), buffer.Bytes()
}
