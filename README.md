# gh2jira

Build Status:
[![Build Status][actions-img]](https://github.com/jmrodri/gh2jira/actions)
License:
[![License](http://img.shields.io/:license-apache-blue.svg)](http://www.apache.org/licenses/LICENSE-2.0.html)
Code coverage:
[![coveralls][coveralls-img]](https://coveralls.io/github/jmrodri/gh2jira?branch=main)

A utility that allows you to copy a Github issue to Jira

## Getting Started
### TokenStore Setup
The gh2jira utility requires a TokenStore configuration file containing GitHub and Jira access tokens.  By default this is `tokenstore.yaml` and follows the schema:

```yaml
schema: gh2jira.tokenstore
authTokens:
  jira: foo
  github: baz
```


### Creating Tokens
#### Setting Up Github Token
1. Follow the instructions [here](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token#creating-a-personal-access-token-classic) to create a personal access token, being sure to only limit the scope of the token to "public_repo" and "read:project".
2. Save to your TokenStore file under the key `authTokens.github`

#### Setting Up Jira Personal Access Token
1. Follow the instructions [here](https://confluence.atlassian.com/enterprise/using-personal-access-tokens-1026032365.html#UsingPersonalAccessTokens-CreatingPATsintheapplication) to set up a Personal Access Token.
2. Save to your TokenStore file under the key `authTokens.jira`

### Profiles
gh2jira now supports Profiles, which are a mechanism to store associated GitHub domains - Jira projects for easy reference, as well as the TokenStore to be used by each (defaulting to `tokenstore.yaml` if unspecified).  By default this is `profiles.yaml` and follows this schema:

```yaml
profiles:
- description: foobaz
  githubConfig:
     project: somedomain/someproject
  jiraConfig:
     project: baz
  tokenStore: foobaz.yaml
- description: foobar   # defaulting tokenStore
  githubConfig:
     project: otherdomain/otherproject
  jiraConfig:
     project: nimrod
```

### Build the Utility
Run `make` from the root of the directory.

## Usage
There are 2 main subcommands: `list` & `clone`. The `list` subcommand will
display all open github issues of the given project. The `clone` subcommand will
copy the given Github issue to your Jira instance.

```
$ ./gh2jira --help
github to jira issue cloner

Usage:
  gh2jira [flags]
  gh2jira [command]

Available Commands:
  clone       Clone given Github issues to Jira
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  list        List Github issues

Flags:
      --github-project string        Github project domain to list if not using a profile, e.g.: operator-framework/operator-sdk
  -h, --help                         help for gh2jira
      --jira-base-url string         Jira base URL, e.g.: https://issues.redhat.com (default "https://issues.redhat.com/")
      --jira-project string          Jira project if not using a profile, e.g.: OSDK
      --profile-name profiles-file   profile name to use (implies profiles-file)
      --profiles-file string         filename containing optional profile attributes (default "profiles.yaml")
      --token-file string            file containing authentication tokens, if different than profile (default "tokenstore.yaml")

Use "gh2jira [command] --help" for more information about a command.
```

### Command flag precedence
Since command flags can be provided via multiple mechanisms, it's necessary to establish a precedence for deterministic override.

gh2jira follows this precedence order:
1. specified by specific flag
2. specified by requested profile
3. default values for options

For example, a user may use a profile name for a clone command but also want to override the target jira project for the creation.
Or the user may need to supply a different TokenStore file for a particular operation.


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
      --assignee string    username assigned the issue
  -h, --help               help for list
      --label strings      label i.e. --label "documentation,bug" or --label doc --label bug (default: none)
      --milestone string   the milestone ID from the url, not the display name

Global Flags:
      --github-project string        Github project domain to list if not using a profile, e.g.: operator-framework/operator-sdk
      --jira-base-url string         Jira base URL, e.g.: https://issues.redhat.com (default "https://issues.redhat.com/")
      --jira-project string          Jira project if not using a profile, e.g.: OSDK
      --profile-name profiles-file   profile name to use (implies profiles-file)
      --profiles-file string         filename containing optional profile attributes (default "profiles.yaml")
      --token-file string            file containing authentication tokens, if different than profile (default "tokenstore.yaml")
```

### `clone` subcommand

The `clone` subcommand will copy the given Github issue to your Jira instance.
*WARNING!* This will write to your Jira instance, consider using the `--dryrun`
flag.

The `--dryrun` flag will print out the Jira issue it would send to Jira.

```
$ ./gh2jira clone --help
Clone given Github issues to Jira.
WARNING! This will write to your jira instance. Use --dryrun to see what will happen

Usage:
  gh2jira clone <ISSUE_ID> [ISSUE_ID ...] [flags]

Flags:
      --dryrun   display what would happen without taking actually doing it
  -h, --help     help for clone

Global Flags:
      --github-project string        Github project domain to list if not using a profile, e.g.: operator-framework/operator-sdk
      --jira-base-url string         Jira base URL, e.g.: https://issues.redhat.com (default "https://issues.redhat.com/")
      --jira-project string          Jira project if not using a profile, e.g.: OSDK
      --profile-name profiles-file   profile name to use (implies profiles-file)
      --profiles-file string         filename containing optional profile attributes (default "profiles.yaml")
      --token-file string            file containing authentication tokens, if different than profile (default "tokenstore.yaml")
```

[actions-img]: https://github.com/jmrodri/gh2jira/workflows/unit/badge.svg
[coveralls-img]: https://coveralls.io/repos/github/jmrodri/gh2jira/badge.svg?branch=main
