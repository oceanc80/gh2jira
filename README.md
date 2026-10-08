# gh2jira
(Forked from https://github.com/jmrodri/gh2jira)


Build Status:
[![Build Status][actions-img]](https://github.com/oceanc80/gh2jira/actions)
License:
[![License](http://img.shields.io/:license-apache-blue.svg)](http://www.apache.org/licenses/LICENSE-2.0.html)
Code coverage:
[![coveralls][coveralls-img]](https://coveralls.io/github/oceanc80/gh2jira?branch=main)

A utility that allows you to retrieve and reconcile relationships between Github and Jira issues

## Getting Started
### TokenStore Setup
The gh2jira utility requires a TokenStore configuration file containing GitHub and Jira access tokens. By default this is `tokenstore.yaml` and follows the schema below. The Jira credentials differ depending on whether you are connecting to a Jira Data Center or Jira Cloud instance.

#### Jira Data Center

```yaml
schema: gh2jira.tokenstore
authTokens:
  jira:
    token: your-personal-access-token
  github: your-github-token
```

#### Jira Cloud

```yaml
schema: gh2jira.tokenstore
authTokens:
  jira:
    email: you@example.com
    apiToken: your-atlassian-api-token
  github: your-github-token
```

The tool auto-detects the mode based on which fields are present. You must provide either `token` (Data Center) or both `email` and `apiToken` (Cloud), but not both.

### Creating Tokens
#### Setting Up Github Token
1. Follow the instructions [here](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token#creating-a-personal-access-token-classic) to create a personal access token, being sure to only limit the scope of the token to "public_repo" and "read:project".
2. Save to your TokenStore file under the key `authTokens.github`

#### Setting Up Jira Token (Data Center)
1. Follow the instructions [here](https://confluence.atlassian.com/enterprise/using-personal-access-tokens-1026032365.html#UsingPersonalAccessTokens-CreatingPATsintheapplication) to set up a Personal Access Token.
2. Save to your TokenStore file under the key `authTokens.jira.token`

#### Setting Up Jira Token (Cloud)
1. Go to [Atlassian API Token Management](https://id.atlassian.com/manage-profile/security/api-tokens) and create an API token.
2. Save your Atlassian account email under `authTokens.jira.email` and the API token under `authTokens.jira.apiToken` in your TokenStore file.

### Profiles
Profiles are a mechanism to store associated GitHub domains - Jira projects for easy reference, as well as the TokenStore to be used by each (defaulting to `tokenstore.yaml` if unspecified).  By default this is `profiles.yaml` and follows this schema:

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
Subcommands are collected into github- or jira-specific groupings as appropriate with the same verb used for consistency. For clarity we'll refer to commands which independently exist for either/both domain (github or jira) as domain-specific commands.  Commands which exist outside or across those domains will be referred to as domain-agnostic commands.

```
$ ./gh2jira --help
github <--> jira issue reconciler

Usage:
  gh2jira [flags]
  gh2jira [command]

Available Commands:
  clone       Clone given Github issues to Jira
  completion  Generate the autocompletion script for the specified shell
  github      Run a github subcommand
  help        Help about any command
  jira        Run a jira subcommand
  reconcile   reconcile github and jira issues
  report      Report read-only GitHub/Jira correlations across repositories

Flags:
      --github-project string   Github project domain to list if not using a profile, e.g.: operator-framework/operator-lifecycle-manager
  -h, --help                    help for gh2jira
      --jira-base-url string    Jira base URL, e.g.: https://issues.redhat.com (default "https://issues.redhat.com/")
      --jira-project string     Jira project if not using a profile, e.g.: OCPBUGS
      --profile-name string     profile name to use (implies profiles-file)
      --profiles-file string    filename containing optional profile attributes (default "profiles.yaml")
      --token-file string       file containing authentication tokens, if different than profile (default "tokenstore.yaml")

Use "gh2jira [command] --help" for more information about a command.
```

### Command flag precedence
Since command flags can be provided via multiple mechanisms, it's necessary to establish a precedence for deterministic override.

gh2jira follows this precedence order:
1. flags specified by specific options
2. flags specified by requested profile
3. default values for flags

For example, a user may use a profile name for a clone command but also want to override the target jira project for the creation.
Or the user may need to supply a different TokenStore file for a particular operation.

### Domain-specific subcommands

#### `github` subcommands
##### `list` subcommand

The `list` subcommand will display all open github issues of the given project.
You can filter the list by milestone, assignee and/or labels.

Multiple labels can be supplied either as a comma separated list or multiple `--label` flags.

For example, `--label kind/bug,kind/documentation` or `--label kind/bug --label
kind/documentation`.

The `--milestone` flag requires the milestone ID. So click on your Github
Milestones tab and look at the ID in the URL, use that.

```
$ ./gh2jira github list --help
List Github issues filtered by milestone, assignee, or label

Usage:
  gh2jira github list [flags]

Flags:
      --assignee string    username assigned the issue
  -h, --help               help for list
      --label strings      label i.e. --label "documentation,bug" or --label doc --label bug (default: none)
      --milestone string   the milestone ID from the url, not the display name

Global Flags:
      --github-project string   Github project domain to list if not using a profile, e.g.: operator-framework/operator-lifecycle-manager
      --jira-base-url string    Jira base URL, e.g.: https://issues.redhat.com (default "https://issues.redhat.com/")
      --jira-project string     Jira project if not using a profile, e.g.: OCPBUGS
      --profile-name string     profile name to use (implies profiles-file)
      --profiles-file string    filename containing optional profile attributes (default "profiles.yaml")
      --token-file string       file containing authentication tokens, if different than profile (default "tokenstore.yaml")
```

#### `jira` subcommands
##### `list` subcommand

The `list` subcommand will display all open jira issues of the specified project.

You can specify an additional JQL query string to be ANDed with the existing issue query.

```
$ ./gh2jira jira list --help
List open Jira issues filtered with optional additional JQL

Usage:
  gh2jira jira list [flags]

Flags:
  -h, --help           help for list
      --query string   Jira query (if provided, ANDed with project)

Global Flags:
      --github-project string   Github project domain to list if not using a profile, e.g.: operator-framework/operator-lifecycle-manager
      --jira-base-url string    Jira base URL, e.g.: https://issues.redhat.com (default "https://issues.redhat.com/")
      --jira-project string     Jira project if not using a profile, e.g.: OCPBUGS
      --profile-name string     profile name to use (implies profiles-file)
      --profiles-file string    filename containing optional profile attributes (default "profiles.yaml")
      --token-file string       file containing authentication tokens, if different than profile (default "tokenstore.yaml")
```

### Domain-agnostic subcommands
#### `clone` subcommand

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
      --github-project string   Github project domain to list if not using a profile, e.g.: operator-framework/operator-lifecycle-manager
      --jira-base-url string    Jira base URL, e.g.: https://issues.redhat.com (default "https://issues.redhat.com/")
      --jira-project string     Jira project if not using a profile, e.g.: OCPBUGS
      --profile-name string     profile name to use (implies profiles-file)
      --profiles-file string    filename containing optional profile attributes (default "profiles.yaml")
      --token-file string       file containing authentication tokens, if different than profile (default "tokenstore.yaml")
```

#### `reconcile` subcommand

The `reconcile` subcommand will identify state mismatches between open jira and linked github issues, according to a state mapping maintained in a local `workflows.yaml` file.
It generates a list of jira issues of the selected project with github issue links and reconciles the set of pairs.

```
$ ./gh2jira reconcile --help
reconcile github and jira issues

Usage:
  gh2jira reconcile [flags]

Flags:
  -h, --help            help for reconcile
  -o, --output string   output format for porcelain display (json or yaml) (default "json")
      --porcelain       display output in an easy-to-parse format for scripts

Global Flags:
      --github-project string   Github project domain to list if not using a profile, e.g.: operator-framework/operator-lifecycle-manager
      --jira-base-url string    Jira base URL, e.g.: https://issues.redhat.com (default "https://issues.redhat.com/")
      --jira-project string     Jira project if not using a profile, e.g.: OCPBUGS
      --profile-name string     profile name to use (implies profiles-file)
      --profiles-file string    filename containing optional profile attributes (default "profiles.yaml")
      --token-file string       file containing authentication tokens, if different than profile (default "tokenstore.yaml")
```

#### `report` subcommand

`report` reads Jira issues and their remote links, then looks up supported GitHub
issues and pull requests across repositories through the official GitHub API.
It never creates links or changes Jira/GitHub states. It includes closed Jira
items, so an open or reopened upstream item linked to a closed Jira item remains
visible. Jira and GitHub states are independent: closed upstream work with open
downstream/version-specific work is normal, and the report does not classify
matches or mismatches or load `workflows.yaml`.

```sh
./gh2jira report --jira-project OCPBUGS --output json
./gh2jira report --jira-project OCPBUGS --query 'labels = upstream OR labels = backport' --output yaml
./gh2jira report --jira-project OCPBUGS --output table
```

A Jira project must come from `--jira-project` or the selected `--profile-name`.
An explicit project flag overrides the profile project. The general configuration's
default project is never used as a report scope. `--query` is an optional JQL **filter expression**, safely
parenthesized and ANDed with the project. It cannot contain `ORDER BY`, comments,
unbalanced quotes/parentheses, or statement separators. Format and raw scope/filter checks
run before configuration files are read. A profile's effective project is then
validated before readers or network clients are invoked. A filter can
select fewer statuses; the default scope has no status restriction.

`--output` / `-o` accepts `json` (default), `yaml`, or `table`. Machine formats
contain an array sorted by Jira key/status, link status, repository, and issue
number. Each row has separate `jira.key` / `jira.status` and, for a supported
link, `github.repository`, `type` (`issue` or `pull_request`), `number`, canonical
`url`, `state`, and an explicit boolean `merged` for PRs only. Issue URLs that
resolve to PRs are reported as PRs. Repeated case/number/query/fragment aliases
within one Jira item produce one correlation. Source lookups are cached across
the report, while distinct Jira items for different downstream versions remain
separate rows.

`link_status` is `linked`, `no_supported_github_links`, or
`malformed_github_reference`. Malformed GitHub-looking references have a `problem`
explaining the rejection; raw malformed URLs are omitted to avoid exposing
embedded credentials. An item containing only malformed references also has a
`no_supported_github_links` row. Unrelated remote links and nil link objects are
ignored. Supported URLs require HTTPS, the exact `github.com` host without
userinfo or port, and `/owner/repository/issues/positive-ID` or
`/owner/repository/pull/positive-ID` (an optional trailing slash, query, and
fragment are normalized). Encoded paths and extra path segments are rejected.
Arbitrary remote URLs are never fetched with authentication.

Any search, remote-link, GitHub lookup, or output-write failure returns an error;
failed lookups are not presented as missing links or successful empty results.
An empty successful scope is `[]` in JSON/YAML or a header-only table. Missing
Jira status fields appear as an empty string.

**Keep report output private.** It omits Jira summaries, descriptions, customer
names, remote-link titles, and attachment contents, but keys, statuses, repository
names, and source URLs can still identify private work. Relationship metadata,
version fields, build automation, and backport tracking are outside this report.

[actions-img]: https://github.com/oceanc80/gh2jira/workflows/unit/badge.svg
[coveralls-img]: https://coveralls.io/repos/github/oceanc80/gh2jira/badge.svg?branch=main
