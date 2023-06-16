package jira

import (
	"fmt"
	"os"
	"strings"

	"sigs.k8s.io/yaml"
)

type Config struct {
	JiraBaseURL string `json:"jiraBaseURL"`
}

func LoadConfig(configFile string) (*Config, error) {
	configData, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	c := &Config{}
	if err := yaml.Unmarshal(configData, c); err != nil {
		return nil, err
	}

	if err := c.Validate(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Config) Validate() error {
	validateErrors := []error{}
	if c.JiraBaseURL == "" {
		validateErrors = append(validateErrors, fmt.Errorf("config must specify `jiraBaseURL`"))
	}
	return newAggregateError(validateErrors)
}

type aggregateError []error

func newAggregateError(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	if len(errs) == 1 {
		return errs[0]
	}
	nonNilErrors := errs[:0]
	for _, err := range errs {
		if err != nil {
			nonNilErrors = append(nonNilErrors, err)
		}
	}
	return aggregateError(nonNilErrors)
}

func (errs aggregateError) Error() string {
	errMsgs := make([]string, 0, len(errs))
	for _, err := range errs {
		errMsgs = append(errMsgs, err.Error())
	}

	return fmt.Sprintf("multiple errors: %s", strings.Join(errMsgs, "; "))
}
