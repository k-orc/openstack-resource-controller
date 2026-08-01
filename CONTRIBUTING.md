# Contributing to ORC

Welcome! We're glad you're interested in contributing to
**openstack-resource-controller** (ORC). Whether you're fixing a bug, adding a
new controller, reviewing code, or improving documentation, your help is
greatly appreciated.

## Types of Contributions

There are many ways to contribute to ORC:

- **Bug fixes**: Check the [open issues][issues] for reported bugs. If you find
  a bug that isn't tracked yet, please open an issue.

- **New OpenStack resource controllers**: ORC aims to cover all OpenStack APIs
  that can be expressed declaratively. If a resource you need isn't supported
  yet (see the [supported resources][supported-resources]),
  consider implementing a controller for it. The [developing
  controllers][dev-overview] guide will help you get started.

- **Code reviews**: Reviewing [open pull requests][prs] is a valuable
  contribution. Fresh eyes catch bugs, improve code quality, and help
  contributors learn from each other.

- **Documentation**: Improvements to the [website docs][website], code
  comments, and this repository's documentation are always welcome.

- **Tests**: We value good test coverage. This includes unit tests, API
  validation tests, and [kuttl][kuttl] end-to-end tests. See [Writing
  Tests][writing-tests] for what's expected.

If you're unsure where to start, look for issues labelled as good first issues,
or ask on Slack.

[issues]: https://github.com/k-orc/openstack-resource-controller/issues
[supported-resources]: https://k-orc.cloud/crd-reference/#resource-types
[dev-overview]: https://k-orc.cloud/development/overview/
[prs]: https://github.com/k-orc/openstack-resource-controller/pulls
[website]: https://k-orc.cloud/
[kuttl]: https://github.com/kudobuilder/kuttl
[writing-tests]: https://k-orc.cloud/development/writing-tests/

## Communication

- **Slack**: Join us on Kubernetes Slack in
  [#gophercloud](https://kubernetes.slack.com/archives/C05G4NJ6P6X). Visit
  [slack.k8s.io](https://slack.k8s.io) for an invitation.
- **GitHub Issues**: For bug reports, feature requests, and design discussions.

## Setting Up a Development Environment

Follow the [local development quickstart][quickstart] to set up a kind cluster,
DevStack, and run ORC from source. The [developer guide][dev-overview] covers
the controller architecture and how all the pieces fit together.

[quickstart]: https://k-orc.cloud/development/quickstart/

## Submitting Changes

### Before you start

For non-trivial changes, we recommend opening a GitHub issue first to discuss
the approach. This avoids spending time on work that may need a different
direction.

For significant new features or architectural changes, please submit an
[enhancement proposal][enhancements] and get it approved before starting
implementation.

[enhancements]: enhancements/README.md

### Workflow

1. **Fork** the repository and create a feature branch from `main`.
2. **Make your changes**. Follow the [coding standards][coding-standards].
3. **Run checks locally** before pushing:
   ```bash
   make generate   # Required after API type changes
   make lint       # Run linters
   make test       # Run unit tests
   ```
4. **Open a pull request** with a clear description of what you changed and
   why. Reference any related issue.
5. **CI must pass**: GitHub Actions runs tests and linting on every PR.
6. **Address review feedback**: At least one maintainer review is required
   before merging. Please be responsive to feedback.

[coding-standards]: https://k-orc.cloud/development/coding-standards/

## Becoming a Maintainer

Maintainership is informal and invite-based. There is no formal ladder, but the
general progression looks like:

1. **First-time contributor**: You submit your first PR and get it merged.
2. **Regular contributor**: You contribute consistently over time, be it code,
   reviews, helping others on Slack or in issues.
3. **Invited maintainer**: Existing maintainers invite new maintainers based on
   demonstrated trust, domain knowledge, and sustained engagement with the
   project.

The best way to move along this path is to show up consistently: write
high-quality code, review others' PRs thoughtfully, and help the community.

## License

ORC is licensed under the [Apache License 2.0](LICENSE). By contributing to
this project, you agree that your contributions will be made under the same
license.
