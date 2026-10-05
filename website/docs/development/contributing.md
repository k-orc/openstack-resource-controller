# Contributing

We're glad you're interested in contributing to ORC. Whether you're
fixing a bug, adding a new controller, reviewing code, or improving
documentation, your help is greatly appreciated.

## Types of Contributions

- **Bug fixes**: Check the [open issues][issues] for reported bugs. If you find
  a bug that isn't tracked yet, please open an issue.

- **New OpenStack resource controllers**: ORC aims to cover all OpenStack APIs
  that can be expressed declaratively. The [scaffolding guide](scaffolding.md)
  and [developer overview](overview.md) will help you get started.

- **Code reviews**: Reviewing [open pull requests][prs] is a valuable
  contribution. Fresh eyes catch bugs, improve code quality, and help
  contributors learn from each other.

- **Documentation**: Improvements to the website docs, code comments, and
  repository documentation are always welcome.

- **Tests**: We value good test coverage. This includes unit tests, API
  validation tests, and kuttl end-to-end tests. See
  [Writing Tests](writing-tests.md) for what's expected.

If you're unsure where to start, look for issues labelled as good first issues,
or ask on Slack.

[issues]: https://github.com/k-orc/openstack-resource-controller/issues
[prs]: https://github.com/k-orc/openstack-resource-controller/pulls

## Communication

- **Slack**: Join us on Kubernetes Slack in
  [#gophercloud](https://kubernetes.slack.com/archives/C05G4NJ6P6X). Visit
  [slack.k8s.io](https://slack.k8s.io) for an invitation.
- **GitHub Issues**: For bug reports, feature requests, and design discussions.

## Getting Started

Follow the [Development Quickstart](quickstart.md) to set up a kind cluster,
DevStack, and run ORC from source. The [developer overview](overview.md) covers
the controller architecture and how all the pieces fit together.

## Submitting Changes

For non-trivial changes, we recommend opening a GitHub issue first to discuss
the approach. This avoids spending time on work that may need a different
direction.

For bug fixes, please ensure a GitHub issue exists before submitting a PR, even
for small fixes. Having a tracking issue makes it easier to reference the bug
in commit messages, changelogs, and future discussions.

For significant new features or architectural changes, please submit an
[enhancement proposal][enhancements] and get it approved before starting
implementation.

[enhancements]: https://github.com/k-orc/openstack-resource-controller/tree/main/enhancements

**Workflow:**

1. Fork the repository and create a feature branch from `main`.
2. Make your changes. Follow the [coding standards](coding-standards.md).
3. Run checks locally before pushing:
   ```bash
   make generate   # Required after API type changes
   make lint       # Run linters
   make test       # Run unit tests
   ```
4. Open a pull request with a clear description of what you changed and why.
   Reference any related issue.
5. CI must pass: GitHub Actions runs tests and linting on every PR.
6. Address review feedback: at least one maintainer review is required before
   merging.
7. Keep a clean history: during review, push fixups as separate commits so
   reviewers can see what changed between rounds. Rebase them into the correct
   commits before merging.

## Contributing New Controllers

New controllers tend to produce large PRs. The guidelines below keep them
reviewable and make it easy to regenerate scaffolding if tooling or conventions
change.

### Commit structure

New controller branches must use a multi-commit structure that separates
generated code from hand-written code:

1. **Scaffolding commit**: the raw output of
   `go run ./cmd/scaffold-controller`. The commit message **must** contain the
   exact command that was run (with all flags) so it can be reproduced. Do not
   include any manual edits. See
   [Scaffolding a New Controller](scaffolding.md) for details.
2. **Generated code commit**: registration in
   `cmd/resource-generator/main.go`, `make generate` output, scope wiring in
   `internal/scope/`, controller registration in `cmd/manager/main.go`, and
   `make generate-bundle`. This is all mechanical boilerplate, no hand-written
   logic.
3. **Implementation commit(s)**: API type definitions (`Filter`,
   `ResourceSpec`, `ResourceStatus`), actuator implementation, status writer,
   and tests. This is the `TODO(scaffolding)` work.

Reviewers can skip the first two commits entirely and focus on the hand-written
implementation.

### Incremental PRs

For complex controllers, consider splitting the work across multiple pull
requests:

- A first PR with scaffolding + generated code + basic immutable
  create/delete/import.
- Follow-up PRs adding mutability, reconcilers for complex sub-resources, tags,
  additional dependencies, etc.

Smaller PRs are easier to review and less likely to need large reworks.

### Deferred mutability

The initial controller implementation may treat all spec fields as immutable.
Mutability for complex fields (`GetResourceReconcilers`, `updateResource`,
single-concern reconcilers like `reconcileExtraSpecs`) can be added in
follow-up PRs.

## AI-Assisted Contributions

Using AI tools (LLMs, coding assistants, etc.) to help write code is fine.
However:

- **Authors are responsible for the code they submit.** Review, understand, and
  test AI-generated code before committing it. The author of a commit is
  accountable for its correctness, not the tool that helped produce it.
- **Authorship**: The human contributor must be the commit author. You may
  optionally add the AI tool as a `Co-authored-by` trailer, but there is no
  requirement to use `Assisted-by`, `Generated-by`, or similar labels.
- **Autonomous agents**: Fully autonomous AI agents (e.g. those that open PRs
  without direct human involvement) must clearly identify themselves as such in
  the PR description.

## License

ORC is licensed under the Apache License 2.0. By contributing, you agree that
your contributions will be made under the same license.
