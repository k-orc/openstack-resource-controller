---
name: release-notes
description: Draft release notes for a new ORC version. Use when preparing a release to generate changelog and GitHub release body from git history.
disable-model-invocation: true
---

# Draft ORC Release Notes

Guide for drafting release notes when preparing a new ORC release.

## When to Use

Use this skill when:
- Preparing a new ORC release
- The user asks to draft or write release notes
- The user asks to prepare a changelog entry

## Step 1: Gather Release Parameters

Ask the user for:
1. **New version number** (e.g., `v2.5.0`)
2. **Release date** (default: today)

Then determine the previous release tag automatically:
```bash
git tag --sort=-v:refname | head -1
```

## Step 2: Collect Git History

Run these commands to gather the raw data (replace `<prev>` with the previous tag):

```bash
# Full commit log
git log <prev>..HEAD --oneline

# Contributors with commit counts
git shortlog -sne <prev>..HEAD

# New controller directories (compare directory listings)
diff <(git ls-tree -d --name-only <prev> internal/controllers/ | sort) \
     <(git ls-tree -d --name-only HEAD internal/controllers/ | sort) \
  | grep '^>'

# All authors who ever contributed before this release
git log <prev> --format='%aN' | sort -u > /tmp/old-contributors.txt

# Authors in this release
git log <prev>..HEAD --format='%aN' | sort -u > /tmp/new-contributors.txt

# First-time contributors
comm -13 /tmp/old-contributors.txt /tmp/new-contributors.txt
```

For each first-time contributor, find the PR number of their first contribution:
```bash
git log <prev>..HEAD --author="<name>" --oneline --reverse | head -1
```
Then look up the corresponding PR number from the merge commit message (format: `Merge pull request #NNN`).

## Step 3: Categorize Changes

Review every commit and sort into sections. Use these rules:

### New controllers
Commits that add a new controller directory under `internal/controllers/`. Format:
```
- <Kind>: Manage <OpenStack service> <resource description>
```
Examples:
- `Keypair: Manage Nova SSH keypairs`
- `Volume: Manage Cinder block storage volumes`
- `Domain: Manage Keystone identity domains`

### New features
Feature additions or enhancements to existing controllers or infrastructure. When scoped to a specific controller, prefix with the controller name and colon:
```
- <Controller>: <Description of the feature>
```
Examples:
- `Server: Added ability to specify SSH keypair`
- `Added support for generating and publishing OLM bundle images`

### Bug fixes
Bug fixes, especially those referencing GitHub issues. Include the issue link when available:
```
- <Description> (Fixes [#NNN](https://github.com/k-orc/openstack-resource-controller/issues/NNN))
```
Examples:
- `Allow to use application credentials with access rules (Fixes [#596](https://github.com/k-orc/openstack-resource-controller/issues/596))`
- `Documentation: Fixed examples in getting-started guide`

### Breaking changes
API incompatibilities or behavioral changes that require user action. Only include this section if there are breaking changes. List the specific type/field changes.

### Update considerations
Important information users need to know when upgrading. Only include this section when relevant (e.g., new minimum OpenStack version requirements).

### Infrastructure improvements
Group related items. Common categories:
- Go version bumps
- Dependency bumps (group into a single bullet: k8s libs, controller-runtime, gophercloud)
- CI changes (OpenStack version support, new test infrastructure)
- Documentation improvements
- Tooling changes

### Commits to skip
Do NOT include in release notes:
- Changes related to newly introduced controllers: if a controller is new in this release, its features and bug fixes are already implied by the "New controllers" entry and must not be duplicated in "New features" or "Bug fixes"
- Dependabot/automated dependency bumps (summarize as a single infrastructure bullet)
- Merge commits
- Code style fixes, typo fixes, linting fixes
- Internal refactoring with no user-visible impact

## Step 4: Write the Opening Summary

The opening summary is optional. Include one when:
- The release has a strong unifying theme
- There is a major new capability worth highlighting

Format: 1-2 sentences before the first section heading.

Example from v2.3.0:
> This release brings support for updating resources after creation for all relevant controllers. You can now modify your OpenStack infrastructure in-place without recreating resources, enabling true lifecycle management for production workloads.

## Step 5: Produce Two Outputs

### Output 1: GitHub Release Body

This is the markdown body for the GitHub release (used with `gh release create`).

Template:
```markdown
## What's Changed

<!-- Optional: opening summary paragraph -->

### New controllers

- Kind1: Description
- Kind2: Description

### New features

- Controller: Feature description
- Feature description

### Bug fixes

- Fix description (Fixes [#NNN](https://github.com/k-orc/openstack-resource-controller/issues/NNN))

### Infrastructure improvements

- Category: Description

## New Contributors

- @username made their first contribution in [#NNN](https://github.com/k-orc/openstack-resource-controller/pull/NNN)

**Full Changelog**: [<prev>...<new>](https://github.com/k-orc/openstack-resource-controller/compare/<prev>...<new>)
```

Notes:
- Omit any section that has no entries (e.g., skip "Breaking changes" if there are none)
- The "New Contributors" section lists first-time contributors with their GitHub username and first PR
- The "Full Changelog" link uses the GitHub compare URL between the two tags

### Output 2: Changelog Entry

This is prepended to `website/docs/changelog.md`, right after the `# Changelog` heading.

Template:
```markdown
## v<MAJOR>.<MINOR> - <Month> <Day>, <Year>

<!-- Optional: opening summary paragraph -->

### New controllers

- Kind1: Description

### New features

- Feature description

### Bug fixes

- Fix description (Fixes [#NNN](https://github.com/k-orc/openstack-resource-controller/issues/NNN))

### Infrastructure improvements

- Description
```

Notes:
- The heading uses `v<MAJOR>.<MINOR>` (no patch version) with the full date
- No "New Contributors" or "Full Changelog" sections
- Otherwise identical content to the GitHub release body

## Step 6: Review Checklist

Before presenting the draft to the user, verify:

- [ ] All new controller directories have corresponding entries in "New controllers"
- [ ] All GitHub issue references use the correct issue number and full URL
- [ ] First-time contributors are identified with correct GitHub usernames and PR numbers
- [ ] Dependency bumps are summarized (not listed individually)
- [ ] The compare URL uses the correct previous and new tag names
- [ ] Sections with no entries are omitted entirely
- [ ] Each bullet is concise (one line, no multi-sentence descriptions)
- [ ] Controller-scoped items are prefixed with the controller name
- [ ] The changelog entry heading uses the short version (vX.Y) and the full date
- [ ] The opening summary (if included) accurately represents the release highlights

## Style Guidelines

- Use past tense for completed work ("Added", "Fixed", "Bumped")
- Capitalize the first word of each bullet
- End bullets without a period
- Group related changes into a single bullet when possible (especially dependency bumps)
- Use the OpenStack service name to describe new controllers (Nova, Neutron, Cinder, Keystone, Glance)
- Reference specific version numbers for dependency bumps (e.g., "gophercloud to v2.9.0")
