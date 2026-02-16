---
name: proposal
description: Write an enhancement proposal for a new ORC feature. Use for significant new features, breaking changes, or cross-cutting architectural changes.
disable-model-invocation: true
---

# Write Feature Proposal

Guide for creating a proposal for a new feature or enhancement in ORC.

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

When in doubt, suggest opening a GitHub issue first to discuss whether an enhancement proposal is needed.

## Enhancement Lifecycle

Enhancements move through the following statuses:

| Status | Description |
|--------|-------------|
| `implementable` | The enhancement has been approved and is ready for implementation |
| `implemented` | The enhancement has been fully implemented and merged |
| `withdrawn` | The enhancement is no longer being pursued |

## Template and File Location

Use the enhancement template at `enhancements/TEMPLATE.md`.

For full process details, see `enhancements/README.md`.

### Creating the Proposal File

Simple enhancement (single file):
```bash
cp enhancements/TEMPLATE.md enhancements/your-feature-name.md
```

Enhancement with supporting files (images, diagrams):
```bash
mkdir enhancements/your-feature-name
cp enhancements/TEMPLATE.md enhancements/your-feature-name/your-feature-name.md
```

## Information to Gather from User

Before writing a proposal, ask the user about:

1. **Feature Overview**
   - What OpenStack resource or capability does this involve?
   - What problem does this solve for users?
   - Is this a new controller, enhancement to existing controller, or infrastructure change?

2. **Use Cases**
   - Who will use this feature? (end users, operators, other controllers)
   - What are the primary use cases?
   - Are there edge cases to consider?

3. **Dependencies**
   - Does this depend on other ORC resources?
   - Does this require new OpenStack API capabilities?
   - Are there upstream dependencies (gophercloud, controller-runtime)?

4. **Scope**
   - Is this a minimal viable feature or full implementation?
   - Are there phases or milestones to break this into?
   - What's explicitly out of scope?

5. **Testing**
   - How is this going to be tested?
   - Are there specific E2E test scenarios required?
   - What OpenStack capabilities are needed for testing?

6. **Existing Infrastructure** (for non-controller enhancements)
   - What related functionality already exists in ORC?
   - Are there existing endpoints, ports, or configurations to integrate with?
   - What frameworks/libraries does ORC already use for this area?

## Research Phase

Before writing the proposal, research relevant areas based on the enhancement type:

### For Controller Enhancements

1. **OpenStack API**
   - Read the OpenStack API documentation for the resource
   - Identify required vs optional fields
   - Understand resource lifecycle (creation, updates, deletion)
   - Check for async operations (polling requirements)

2. **Gophercloud Support**
   - Check if gophercloud has client support for this resource
   - Identify the module path and types
   - Note any missing functionality that needs upstream work

3. **Existing Patterns**
   - Look at similar controllers in ORC for patterns to follow
   - Identify if existing utilities can be reused
   - Check if new generic functionality is needed

4. **Dependencies**
   - Map out all ORC resource dependencies
   - Determine which are required vs optional
   - Identify deletion guard requirements

### For Infrastructure Enhancements (metrics, webhooks, etc.)

1. **Current Implementation**
   - Check existing code for related functionality (e.g., `cmd/manager/`, `internal/`)
   - Identify current ports, endpoints, and configurations
   - Verify technical details by reading the actual code

2. **Framework Capabilities**
   - Check controller-runtime documentation for built-in features
   - Identify what's provided vs what needs custom implementation

3. **Integration Points**
   - How does this integrate with existing infrastructure?
   - What configuration already exists that this should use?

## Filling Out the Template

Read the template at `enhancements/TEMPLATE.md` and fill in each section:

| Section | What to Include |
|---------|-----------------|
| **Metadata table** | Status (`implementable`), author, dates, tracking issue (TBD initially) |
| **Summary** | 1-2 paragraph overview of the enhancement |
| **Motivation** | Why this is needed, who benefits, links to issues |
| **Goals** | Specific, measurable objectives |
| **Non-Goals** | What's explicitly out of scope |
| **Proposal** | Detailed solution with API examples |
| **Risks and Edge Cases** | What could go wrong, mitigations (see risk checklist below) |
| **Alternatives Considered** | Other approaches and why rejected |
| **Implementation History** | Timeline of major milestones |

### For New Controller Proposals

**Note**: New controllers following existing patterns typically don't need an enhancement proposal. Only write a proposal if the controller requires new patterns or architectural changes.

If a proposal is needed, in the **Proposal** section, include:

```yaml
apiVersion: openstack.k-orc.cloud/v1alpha1
kind: ResourceName
metadata:
  name: example
spec:
  cloudCredentialsRef:
    secretName: openstack-credentials
    cloudName: openstack
  resource:
    # Required fields with descriptions
    # Optional fields with descriptions
  import:
    filter:
      # Filter fields
status:
  id: "uuid"
  conditions: [...]
  resource:
    # Observed state fields
```

Also describe:
- Controller behavior (creation, updates, deletion)
- Dependencies (required vs optional, deletion guards)
- Mutable vs immutable fields

### Risk Checklist

Address each of these in the **Risks and Edge Cases** section:

| Risk Category | Questions to Answer |
|---------------|---------------------|
| **API compatibility** | Will this break existing users? Are metric/label names stable? |
| **Security** | Are there security implications? (Often N/A for read-only features) |
| **Performance** | Could this impact controller performance at scale? |
| **Error handling** | What happens when things fail? |
| **Upgrade/downgrade** | How does this affect users upgrading or downgrading ORC? |
| **OpenStack compatibility** | Does this work across different OpenStack versions? (N/A for K8s-only) |
| **Interaction with existing features** | Could this conflict with existing behavior? |

### Verification Before Submission

Before finalizing the proposal:

1. **Internal consistency**: Verify anything referenced in one section is defined elsewhere
   - If you mention a metric/API/config in mitigations, ensure it's defined in Proposal
   - If you reference a flag, show its usage

2. **Technical accuracy**: Verify details against actual code
   - Check ports, endpoints, paths in the codebase
   - Verify framework capabilities match what you describe

3. **Completeness**: Ensure examples are complete and correct
   - Code examples should compile conceptually
   - Config examples should be valid YAML/JSON

## Tips for Writing Good Enhancements

1. **Be concise but complete** - Include enough detail for reviewers to understand the proposal without unnecessary verbosity.

2. **Focus on the "why"** - Motivation is often more important than implementation details.

3. **Think about edge cases** - The Risks and Edge Cases section is where you demonstrate you've thought through the implications.

4. **Consider alternatives** - Showing that you've evaluated other approaches strengthens your proposal.

5. **Keep it updated** - As implementation progresses, update the Implementation History section.

## Submission Process

1. **Fill out the template** with proposal details
2. **Open a pull request** with title: `Enhancement: Add support for feature X`
3. **Iterate based on feedback** - Discussion happens on the PR
4. **Create a tracking issue** once merged - Label with `enhancement` and link in metadata

## Review Process

- Any community member can propose an enhancement
- Maintainers review proposals and provide feedback on the PR
- Enhancements are approved using lazy consensus (typically one week review period)
- The enhancement author is typically expected to drive implementation

## Checklist

- [ ] Confirmed enhancement proposal is needed (not just a bug fix or small improvement)
- [ ] Gathered feature requirements from user
- [ ] Researched relevant areas (OpenStack API, gophercloud, or existing infrastructure)
- [ ] Reviewed similar implementations in ORC
- [ ] Copied template to `enhancements/`
- [ ] Filled in all template sections
- [ ] Addressed all items in risk checklist
- [ ] Documented alternatives considered
- [ ] Verified internal consistency (references match definitions)
- [ ] Verified technical accuracy against codebase
- [ ] Opened PR for review
