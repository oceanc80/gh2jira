package token

import (
	"errors"
	"os"

	"gopkg.in/yaml.v3"
)

type Tokens struct {
	GithubToken string `yaml:"githubToken"`
	JiraToken   string `yaml:"jiraToken"`
}

func ReadTokensYaml(file string) (*Tokens, error) {
	data, err := readFile(file)
	if err != nil {
		return nil, err
	}

	var tokens Tokens
	err = yaml.Unmarshal([]byte(data), &tokens)
	if err != nil {
		return nil, err
	} else if tokens.GithubToken == "" {
		return nil, errors.New("missing required github token")
	} else if tokens.JiraToken == "" {
		return nil, errors.New("missing required jira token")
	}

	return &tokens, nil
}

var readFile = func(file string) ([]byte, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	return data, nil
}
