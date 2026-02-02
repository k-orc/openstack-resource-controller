# ORC Enhancement Process

This document describes the process for proposing significant changes to ORC.
The process is intentionally lightweight, inspired by the [Kubernetes
Enhancement Proposal (KEP)][kep] process but tailored to ORC's scope and
community size.

[kep]: https://github.com/kubernetes/enhancements/tree/master/keps

## When to Write an Enhancement

Write an enhancement proposal when you want to:

- Add a significant new feature or capability
- Make breaking changes to existing APIs
- Deprecate or remove functionality
- Make cross-cutting architectural changes
- Change behavior that users depend on

You do **not** need an enhancement for:

- Bug fixes
- Small improvements or refactoring
- Documentation updates
- Adding support for additional OpenStack resource fields
- Test improvements

When in doubt, open a GitHub issue first to discuss whether an enhancement
proposal is needed.

## Enhancement Lifecycle

Enhancements move through the following statuses:

| Status | Description |
|--------|-------------|
| `implementable` | The enhancement has been approved and is ready for implementation. |
| `implemented` | The enhancement has been fully implemented and merged. |
| `withdrawn` | The enhancement is no longer being pursued. |

## How to Submit an Enhancement

1. **Copy the template** from [TEMPLATE.md](TEMPLATE.md) to a new file named
   after your feature:
   ```
   enhancements/your-feature-name.md
   ```

   If your enhancement requires supporting files (images, diagrams), create a
   directory instead:
   ```
   enhancements/your-feature-name/
   ├── your-feature-name.md
   └── diagram.png
   ```

2. **Fill out the template** with your proposal details.

3. **Open a pull request** with your enhancement proposal. Use a descriptive
   title like: `Enhancement: Add support for feature X`

4. **Iterate based on feedback**. Discussion happens on the PR.

5. **Create a tracking issue** once the enhancement is merged. Label the issue
   with `enhancement` and link it in your enhancement's metadata table.

## Review Process

- Any community member can propose an enhancement
- Maintainers review proposals and provide feedback on the PR
- Enhancements are approved using lazy consensus: if no maintainer has objected
  after a reasonable review period (typically one week), the enhancement can be
  merged
- The enhancement author is typically expected to drive implementation, though
  others may volunteer

## Directory Structure

```
enhancements/
├── README.md                    # This document
├── TEMPLATE.md                  # Template for new enhancements
├── your-feature-name.md         # Simple enhancement (single file)
└── complex-feature/             # Enhancement with supporting files
    ├── complex-feature.md
    └── architecture.png
```

## Tips for Writing Good Enhancements

1. **Be concise but complete**. Include enough detail for reviewers to
   understand the proposal without unnecessary verbosity.

2. **Focus on the "why"**. Motivation is often more important than
   implementation details.

3. **Think about edge cases**. The Risks and Edge Cases section is where you
   demonstrate you've thought through the implications.

4. **Consider alternatives**. Showing that you've evaluated other approaches
   strengthens your proposal.

5. **Keep it updated**. As implementation progresses, update the Implementation
   History section.
