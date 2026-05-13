## Plan / Issues

<!-- Link the exec-plan, issue, or project-plan that triggered this PR. If none, write N/A.
If the matching execution plan has an `Addresses:` line with local issue paths, list the same
paths under Issues unless you are intentionally leaving one open. In that case, keep the path
here and say `remains open: <reason>` somewhere in the PR body. -->

- **Plan**: <!-- e.g., docs/exec-plan/todo/0013-feature-name.md -->
- **Issues**: <!-- e.g., docs/issues/0013-bug-name.md, or GitHub issue link -->

## Type of Change

- [ ] Project Plan update
- [ ] Execution Plan (new/updated plan)
- [ ] Feature implementation
- [ ] Bug fix
- [ ] Refactor
- [ ] Documentation only
- [ ] Chore (CI, tooling, deps)

## Human Instructions / Intent

<!--
Record the human instruction and intent that led to creating this PR.
This section documents how the PR was initiated, not a command that must
match text inside the diff.

If this PR creates or updates an execution plan, record the planning command
that produced the plan PR.
If this PR executes an approved plan, record the execution command that
produced the implementation PR.

Examples:
- Plan creation: `/plan-execution docs/exec-plan/todo/<sequence>-<plan-name>.md`
- Plan execution: `/execute-task docs/exec-plan/todo/<sequence>-<plan-name>.md`
- Project planning: `/plan-project docs/project-plan.md`

For chores/docs-only or other non-execution PRs, use:
- Human instruction: `N/A`
-->
Human instruction: `________________`
### Additional Context from Instructing Human

<!--
Record instructions, decisions, and intent from the human that are NOT already
captured in the exec-plan, specs, or code diff. This section preserves context
that would otherwise be lost when the conversation ends.

What belongs here:
- Library/framework preferences ("Use slog instead of log for structured logging")
- Implementation-style directives ("Keep it simple — no premature abstraction")
- Decisions made during AI-human dialogue — include the AI's question AND the
  human's answer so the context is self-contained
  (e.g., "AI proposed approach A (faster) vs B (simpler). Human chose B,
   reasoning: maintenance cost matters more than runtime speed for this component.")
- Scope decisions about discovered issues
  ("Found stale import in utils.go — out of scope, logged as docs/issues/stale-import.md")
- Quality/priority directives ("Ship this as-is; polish in next iteration")

If no additional instructions were given beyond the standard plan execution, write N/A.
-->

N/A

## Verification

<!-- How was this change verified? Fill in relevant items. -->

- [ ] Tests pass (command: `________________`)
- [ ] Lint passes (command: `________________`)
- [ ] Manual verification (describe below)

## Deterministic Golden Updates

<!--
Use this section when a deterministic regression golden under ai-arena changes.
If the checkbox is checked, explain why the update is allowed and where that same
rationale is recorded in the spec and/or exec-plan.
If no deterministic golden changed, leave the box unchecked and write N/A below.
-->

- [ ] This PR updates a deterministic regression golden
- Allowed update reason: `________________`
- Spec/plan reference that records the same reason: `________________`

## Checklist

- [ ] Branch created from latest `origin/main`
- [ ] `docs/specs/` updated (Spec-Code Parity) — _if code changed_
- [ ] Plan moved from `todo/` to `done/` — _if executing a plan_
- [ ] Resolved linked local issues from the plan's `Addresses:` line were moved to `docs/issues/done/`, or this PR explains why they remain open
- [ ] Workflow-linter warnings reviewed; all `fixable` warnings were resolved or explicitly justified in this PR
- [ ] New issues logged in `docs/issues/` — _if discovered during work_
- [ ] No unresolved blockers remain

## Dependencies

<!-- PRs or issues that must be merged before this one, or that are blocked by this one. -->

N/A

## Reviewer Notes

<!-- Specific areas to focus on during review, known trade-offs, or things that look wrong but are intentional. -->

N/A

## Links

<!-- External references: library docs, design references, related discussions, etc. -->

N/A

## Breaking Changes

<!-- Describe breaking changes. Delete this section if none. -->

N/A

## Screenshots / Logs

<!-- Attach verification artifacts if applicable (test output, screenshots, before/after metrics). Delete this section if not applicable. -->
