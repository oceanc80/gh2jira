# gh2jira

Build Status:
[![Build Status][actions-img]](https://github.com/jmrodri/gh2jira/actions)
License:
[![License](http://img.shields.io/:license-apache-blue.svg)](http://www.apache.org/licenses/LICENSE-2.0.html)
Code coverage:
[![coveralls][coveralls-img]](https://coveralls.io/github/jmrodri/gh2jira?branch=main)

A utility that allows you to copy a Github issue to Jira

## Getting Started
The gh2jira utility requires a `config.yaml` file with Github Token, Jira Personal Access Token and Jira URL.Example `config.yaml` file:

```yaml
schema: gh2jira.config
jiraBaseUrl: blah
authTokens:
  jira: foo
  github: baz
```

### Creating Tokens
#### Setting Up Github Token
1. Follow the instructions [here](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token#creating-a-personal-access-token-classic) to create a personal access token, being sure to only limit the scope of the token to "public_repo" and "read:project".
2. Save to the `config.yaml` file under the key `github`

#### Setting Up Jira Personal Access Token
1. Follow the instructions [here](https://confluence.atlassian.com/enterprise/using-personal-access-tokens-1026032365.html#UsingPersonalAccessTokens-CreatingPATsintheapplication) to set up a Personal Access Token.
2. Save to the `config.yaml` file under the key `jira`

### Configuring the JiraURL
Specify the required Jira URL in the `config.yaml` file under the key `jiraBaseURL`
### Build the Utility
Run `go build` from the root of the directory.

## Usage
There are 2 main subcommands: `list` & `clone`. The `list` subcommand will
display all open github issues of the given project. The `clone` subcommand will
copy the given Github issue to your Jira instance.

```
$ ./gh2jira --help
github to jira issue cloner

Usage:
  gh2jira [command]

Available Commands:
  clone       Clone given Github issues to Jira
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  list        List Github issues

Flags:
  -h, --help   help for gh2jira

Use "gh2jira [command] --help" for more information about a command.
```

### `list` subcommand

The `list` subcommand will display all open github issues of the given project.
You can filter the list by milestone, assignee and/or labels.

Multiple labels can be supplied either as a comma separated list or multiple `--label` flags.

For example, `--label kind/bug,kind/documentation` or `--label kind/bug --label
kind/documentation`.

The `--milestone` flag requires the milestone ID. So click on your Github
Milestones tab and look at the ID in the URL, use that.

```
$ ./gh2jira list --help
List Github issues filtered by milestone, assignee, or label

Usage:
  gh2jira list [flags]

Flags:
      --assignee string    username of the issue is assigned
      --config-file string  file containing jira base url and github and jira tokens (default "config.yaml")
  -h, --help               help for list
      --label strings      label i.e. --label "documentation,bug" or --label doc --label bug
      --milestone string   the milestone ID from the url, not the display name
      --project string     Github project to list e.g. ORG/REPO (default "operator-framework/operator-sdk")
```

### `clone` subcommand

The `clone` subcommand will copy the given Github issue to your Jira instance.
*WARNING!* This will write to your Jira instance, consider using the `--dryrun`
flag.

The `--dryrun` flag will print out the Jira issue it would send to Jira.

```
$ ./gh2jira clone --help
Clone given Github issues to Jira. WARNING! This will write to your jira instance. Use --dryrun to see what will happen

Usage:
  gh2jira clone <ISSUE_ID> [ISSUE_ID ...] [flags]

Flags:
      --config-file string  file containing jira base url and github and jira tokens (default "config.yaml")
      --dryrun                  display what we would do without cloning
      --github-project string   Github project to clone from e.g. ORG/REPO (default "operator-framework/operator-sdk")
  -h, --help                    help for clone
      --project string          Jira project to clone to (default "OSDK")
```

[actions-img]: https://github.com/jmrodri/gh2jira/workflows/unit/badge.svg
[coveralls-img]: https://coveralls.io/repos/github/jmrodri/gh2jira/badge.svg?branch=main
