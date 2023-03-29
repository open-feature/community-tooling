# Community Tooling

This repository contains tools to handle our community, like our peribolos-config-merger.

## Releasing

Create a GitHub Release with a tag in the format of `v<major>.<minor>.<patch>` and this will trigger the automatic releasing process.

## Tools

### Peribolos config merger

The merger contains our custom generation logic for the peribolos configuration.

Within the `config` directory, we are storing our configurations.
(this can be overwritten with a config flag (`--config`) when executed).

#### Configuration Structure

Each directory within the `config` directory represents a GitHub Organization (therefore we will refrain from it as `org-folder`).
This directory will only be picked up when there is a `org.yaml` within the directory.

Within this `org-folder` we can have multiple teams/workgroups.
Those teams are represented with their subfolder (the name of the team) containing a `workgroup.yaml`.

The merger will fetch these configurations and generate a proper peribolos configuration based on this.

##### org.yaml

The `org.yaml` follows the default peribolos configuration format.

It will be used to:

- define members and admins of the organization
- define default settings for the organization such as:
  - members allowed to create repositories
  - default permissions for members
  - ...
- repositories and their configuration
- global teams

###### special teams

There are 3 special teams within the `org.yaml`.

##### &lt;workgroup&gt;/workgroup.yaml

Each workgroup represents an organizational unit, which needs to work on the same repositories.

A workgroup consists of following roles:

- approvers (triage permission)
- maintainers (maintain permission)
- admins (admin permission)

Based on the definition above a `workgroup.yaml` has the following structure:

```yaml
repos: # a list of repositories the team has access to
  - repo-1
  - repo-2

approvers:
  - approver-1

maintainers:
  - maintainer-1

admins:
  - admins-1
```

Repositories are not mutually exclusive to workgroups.
Hence, multiple workgroups can have access to the same repositories.

> **Note**
> Use admins carefully and only when it is really needed.
> Admins can change secrets etc.

Based on this configuration we will generate 3 teams:

- &lt;workgroup&gt;-approvers
- &lt;workgroup&gt;-maintainers
- &lt;workgroup&gt;-admins

Furthermore, we will assign those 3 teams also permissions for the defined repositories.

##### Example

Let's say we have a `workgroup.yaml` within `org-folder/workgroup`.

The content is:

```yaml
repos:
  - repo-1

approvers:
  - approver-1

maintainers:
  - maintainer-1

admins:
  - admins-1
```

Following Teams will be generated, with the respective permissions for the repository `repo-1`:

- workgroup-approvers: triage
- workgroup-maintainers: maintain
- workgroup-admins: admin
