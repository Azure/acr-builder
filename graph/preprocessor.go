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
	"io/ioutil"
	"net/http"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/Azure/acr-builder/util"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

var (
	errImproperDirectiveLength = errors.New("$ directive can only be overwritten by a single character")
	errImproperKeyName         = errors.New("alias key names only support alphanumeric characters")
	errImproperDirectiveChoice = errors.New("overwritten directives may not be alphanumeric characters")
	defaultDirective           = '$'
	aliasFormat                = regexp.MustCompile(`\A[a-zA-Z0-9]+\z`)
)

const versionKey = "version"

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

		alias.directive = val[0]
	}

	// Values may support all characters, no escaping and so forth necessary
	for key := range alias.AliasMap {
		matched := aliasFormat.MatchString(key)
		if !matched {
			return errImproperKeyName
		}
	}

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

// Loads in all global aliases switching definition based on os
func (alias *Alias) loadGlobalAlias() {
	//Identify defaults location.
	if runtime.GOOS == "windows" {
		readAliasFromBytes([]byte(globalDefaultYamlWindows), alias)
	} else { // Looking at Linux
		readAliasFromBytes([]byte(globalDefaultYamlLinux), alias)
	}
}

// Fetches and parses out remote alias files and adds their content
// to the passed in Alias. Note alias definitions already in alias
// will not be overwritten.
func addAliasFromRemote(alias *Alias, url string) error {

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

	return readAliasFromBytes(data, alias)
}

// Parses out local alias files and adds their content to the passed in
// Alias. Note alias definitions already in alias will not be
// overwritten.
func addAliasFromFile(alias *Alias, fileURI string) error {
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

// preprocessString handles the preprocessing (string replacement and resolution)
// of all aliases in an input yaml (passed in as a string). The resolved aliases are
// defined in the input alias file.
func preprocessString(alias *Alias, str string) (string, error) {
	// Load Remote/Local alias definitions
	if externalDefinitionErr := alias.loadExternalAlias(); externalDefinitionErr != nil {
		return "", externalDefinitionErr
	}

	alias.loadGlobalAlias()

	// Validate alias definitions
	if improperFormatErr := alias.resolveMapAndValidate(); improperFormatErr != nil {
		return "", improperFormatErr
	}

	var out strings.Builder
	var command strings.Builder
	ongoingCmd := false

	// Search and replace all strings with the directive
	for _, char := range str {
		if ongoingCmd {
			if char == alias.directive && command.Len() == 0 { // Escape Character Triggered
				out.WriteRune(alias.directive)
				ongoingCmd = false
			} else if !isAlphanumeric(char) { // Delineates the end of an alias
				resolvedCommand, commandPresent := alias.AliasMap[command.String()]
				// If command is not found we assume this to be the expect item itself.
				if !commandPresent {
					out.WriteString(string(alias.directive) + command.String() + string(char))
					ongoingCmd = false
					command.Reset()
				} else {
					out.WriteString(resolvedCommand)
					if char != alias.directive {
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

	return out.String(), nil
}

// PreprocessBytes handles byte encoded data that can be parsed through pre processing
func PreprocessBytes(data []byte) ([]byte, *Alias, error) {
	aliasData, remainingData := SeparateAliasFromRest(data)
	return SearchReplaceAlias(data, aliasData, remainingData)
}

// SearchReplaceAlias replaces aliasData in the Task
func SearchReplaceAlias(originalData, aliasData, data []byte) ([]byte, *Alias, error) {
	type wrapper struct {
		Alias Alias `yaml:"alias,omitempty"`
	}
	wrap := &wrapper{}
	if errUnmarshal := yaml.Unmarshal(aliasData, wrap); errUnmarshal != nil {
		return originalData, &Alias{}, errors.Wrap(errUnmarshal, "error during alias unmarshaling")
	}

	alias := &wrap.Alias

	// Alias Src defined. Guarantees alias map can be populated
	if alias.AliasMap == nil {
		alias.AliasMap = make(map[string]string)
	}
	// Search and Replace
	parsedStr, err := preprocessString(alias, string(data))
	return []byte(parsedStr), alias, err
}

// ExpandCommandAliases will resolve image names in cmd steps that are aliased without using directive.
// Invoked after resolving all directive using aliases
func ExpandCommandAliases(alias *Alias, task *Task) {
	for i, step := range task.Steps {
		parts := strings.Split(strings.TrimSpace(step.Cmd), " ")
		if val, ok := alias.AliasMap[parts[0]]; ok {
			// Image name should always go first
			parts[0] = val
			task.Steps[i].Cmd = strings.Join(parts, " ")
		}
	}
}

// SeparateAliasFromRest separates out alias blurb from the rest of the Task
func SeparateAliasFromRest(data []byte) ([]byte, []byte) {
	reader := bytes.NewReader(data)
	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanLines)

	var aliasBuffer bytes.Buffer
	var buffer bytes.Buffer

	inside := false
	aliasFieldName := regexp.MustCompile(`\Aalias\s*:.*\z`)
	genericTopLevelRe := regexp.MustCompile(`\A[^\s:]+[^:]*:.*\z`)
	commentRe := regexp.MustCompile(`\A\s*#.*`)
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

	return aliasBuffer.Bytes(), buffer.Bytes()
}

// FindVersion determines the current version of a task file
func FindVersion(data []byte) string {
	reader := bytes.NewReader(data)
	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		text := scanner.Text()
		trimmedText := strings.TrimSpace(text)

		// Skip comments and just whitespace.
		if trimmedText == "" || strings.HasPrefix(trimmedText, "#") {
			continue
		}

		// use text instead of trimmedText since ' version: ' is also invalid.
		if strings.HasPrefix(text, versionKey) {
			tokens := strings.SplitN(text, ":", 2)
			if len(tokens) == 2 && strings.TrimSpace(tokens[0]) == versionKey {
				return strings.TrimSpace(tokens[1])
			}
		}
		break
	}

	return ""
}
