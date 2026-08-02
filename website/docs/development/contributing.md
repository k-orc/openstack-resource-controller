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

## License

ORC is licensed under the Apache License 2.0. By contributing, you agree that
your contributions will be made under the same license.
