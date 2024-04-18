// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package workflow

import (
	"fmt"
	"os"

	"sigs.k8s.io/yaml"
)

type StateMapping struct {
	GHState string   `json:"ghstate"`
	JStates []string `json:"jstates"`
}

type Workflows struct {
	Schema   string         `json:"schema"`
	Name     string         `json:"name"`
	Mappings []StateMapping `json:"mappings"`
}

var stateMappings map[string][]string

const workflowfile string = "workflows.yaml"
const defaultWorkflow string = "jira"
const schemaName string = "gh2jira.workflows"

func ReadWorkflows() error {
	b, err := readFile(workflowfile)
	if err != nil {
		return err
	}

	var ws Workflows
	err = yaml.Unmarshal(b, &ws)
	if err != nil {
		return err
	}
	if ws.Schema != schemaName {
		return fmt.Errorf("invalid schema: %q should be %q", ws.Schema, schemaName)
	}

	stateMappings = make(map[string][]string)
	if ws.Name == defaultWorkflow {
		for _, m := range ws.Mappings {
			stateMappings[m.GHState] = m.JStates
		}
	}

	return nil
}

func ValidateState(ghstate string, jirastate string) (bool, error) {

	if len(stateMappings) == 0 {
		return false, fmt.Errorf("no state mappings found")
	}

	jstates, ok := stateMappings[ghstate]
	if !ok {
		return false, fmt.Errorf("no state mapping found for %q", ghstate)
	}

	for _, s := range jstates {
		if s == jirastate {
			return true, nil
		}
	}

	return false, nil
}

// overrideable func for mocking os.ReadFile
var readFile = func(file string) ([]byte, error) {
	return os.ReadFile(file)
}
