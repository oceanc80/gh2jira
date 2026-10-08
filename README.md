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

The `--dryrun` flag describes the intended action without writing anything.
Action output does not print source titles or bodies.

#### Repeatable cloning with `--if-missing`

Cloning without `--if-missing` still creates a new Jira issue each time. Opt in
to reuse with the same profile and destination flags as ordinary cloning:

```sh
./gh2jira clone --if-missing --profile-name my-profile 42 43
./gh2jira clone --if-missing --github-project org/repo --jira-project DST --jira-issue-type Bug 42
./gh2jira clone --if-missing --dryrun --profile-name my-profile 42
```

The entire batch of source IDs must be positive decimal integers. Invalid IDs
are rejected before reading configuration, connecting, or creating any records.
Other failures stop the command with an error; earlier successful sources in a
batch are not rolled back.

`--if-missing` searches the configured Jira project **and issue type**, including
closed issues. It reuses exactly one issue with the source's canonical repository
issue URL in eligible remote links or a matching `gh2jira-source-<SHA-256>` label.
Links without relationship metadata remain eligible for legacy compatibility.
Explicit `implements` and `fixes` relationships also count (case-insensitive,
with surrounding whitespace ignored). Other explicit relationships, including
`depends on`, do not establish clone identity. Every eligible link is examined:
an explicit dependency link cannot hide a legacy match or resolve ambiguity
between intentional version-specific issues. The creation marker remains
authoritative for recovery, even if only a dependency link is present; recovery
adds a separate unclassified source link and preserves the dependency link.
Titles, summaries, and issue numbers alone are never identity. Repository and
host are part of the URL identity; query strings, fragments, trailing slash,
HTTP/HTTPS, and repository/host case are normalized. The marker hashes this URL
for identity only; stored source URLs, remote-link titles, and checklist links
retain original owner/repository case. Checklist expansion normalizes CRLF to
LF and changes only the issue-number path segment, not numbers in repository
names or subsequent references in checklist text.
The marker contains no title, body, token, or raw source URL. It is added as part of
Jira creation, on both Cloud and Data Center. If creation succeeds but linking
fails, rerun with `--if-missing`: the marker identifies the existing issue and
the command adds the missing source link. Preserve these labels for recovery.
Existing relationships and issue contents are not overwritten. No GitHub
comments, updates, or other public writes are performed.

Multiple matching issues produce an ambiguity error listing their Jira keys,
with no create or link writes for that source. This preserves intentional
many-to-many and version-specific relationships. For a deliberate new
version-specific Jira issue, omit `--if-missing`. To choose an existing issue,
use an existing-record link command if available in your installation, or add
the GitHub URL to that issue through Jira's web-link UI. This branch does not
add a link command or automatically select a version-specific issue.

**Performance and consistency limitations:** To find legacy records that only
have remote links (no marker), each source scans all visible issues in the
destination project/type and reads their remote links, including closed issues.
This costs paginated search requests plus one remote-link request per issue,
per source. Search and remote-link read errors fail closed. Missing search
metadata, skipped/repeated DC offsets, missing/null issue fields, repeated Cloud
pagination tokens, and null remote-link payloads also fail closed; an explicit
empty remote-link array is valid absence. A later-page error discards earlier
search pages rather than permitting mutation from partial results. Dry-run performs
the same reads and reports create, reuse, or missing-link recovery, but performs
no writes, even during recovery. Jira must allow labels on the create screen;
creation errors are returned rather than retrying without the recovery marker.

This is sequential best-effort repeatability, **not** a multi-process transaction
or cross-machine exactly-once guarantee. Concurrent invocations can race between
search and create. Jira search-index delay can hide newly created issues,
including between sources in the same batch; wait for indexing before rerunning
after a partial success. Permissions/access visibility can hide existing records
or links, causing apparent absence. Removing a marker after a failed link or
moving/renaming a source repository before recovery can also defeat detection.
Ensure the account can see the full destination scope. No summary-based fallback
is used.

```
$ ./gh2jira clone --help
Clone given Github issues to Jira.
WARNING! This will write to your jira instance. Use --dryrun to see what will happen

Usage:
  gh2jira clone <ISSUE_ID> [ISSUE_ID ...] [flags]

Flags:
      --dryrun   display what would happen without taking actually doing it
      --if-missing   reuse a unique existing source relationship, or create if missing
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

[actions-img]: https://github.com/oceanc80/gh2jira/workflows/unit/badge.svg
[coveralls-img]: https://coveralls.io/repos/github/oceanc80/gh2jira/badge.svg?branch=main
