#!/bin/bash
set -euo pipefail

# Workflow linter for AI-Centered Development
# Mechanically enforces rules declared in .claude/vendor/workflow/AI_WORKFLOW.md
# All checks are warnings only (exit 0)

# Colors
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

MODE=""
PR_TITLE=""
PR_BODY=""
WARN_COUNT=0
FIXABLE_WARN_COUNT=0
ADVISORY_WARN_COUNT=0
DIFF_CHECKS_AVAILABLE=true
CHANGED_FILES=""
DELETED_FILES=""
WORKFLOW_DOC=".claude/vendor/workflow/AI_WORKFLOW.md"
WORKFLOW_DOGFOOD_DOC=".claude/vendor/workflow/docs/specs/ww-dogfooding-workflow.md"

usage() {
    echo "Usage: $0 --mode=pre-push|ci [--pr-title=TITLE] [--pr-body=BODY]" >&2
    exit 1
}

emit_warning() {
    local warning_class="$1"
    local finding="$2"
    local why="$3"
    local fix="${4:-}"
    local normalized_class="$warning_class"

    WARN_COUNT=$((WARN_COUNT + 1))

    case "$normalized_class" in
        fixable)
            FIXABLE_WARN_COUNT=$((FIXABLE_WARN_COUNT + 1))
            ;;
        advisory)
            ADVISORY_WARN_COUNT=$((ADVISORY_WARN_COUNT + 1))
            ;;
        *)
            echo "Internal warning: unknown workflow-lint warning class '${warning_class}', treating it as advisory" >&2
            normalized_class="advisory"
            ADVISORY_WARN_COUNT=$((ADVISORY_WARN_COUNT + 1))
            ;;
    esac

    echo -e "${YELLOW}[WARN:${normalized_class}]${NC} ${finding}" >&2
    echo "  WHY: ${why}" >&2
    if [ "$normalized_class" = "fixable" ] && [ -n "$fix" ]; then
        echo "  FIX: ${fix}" >&2
    fi
}

info() {
    echo -e "${GREEN}[INFO]${NC} $1" >&2
}

# Parse arguments
for arg in "$@"; do
    case "$arg" in
        --mode=*)
            MODE="${arg#--mode=}"
            ;;
        --pr-title=*)
            PR_TITLE="${arg#--pr-title=}"
            ;;
        --pr-body=*)
            PR_BODY="${arg#--pr-body=}"
            ;;
        --help|-h)
            usage
            ;;
        *)
            echo "Unknown argument: $arg" >&2
            usage
            ;;
    esac
done

if [ -z "$MODE" ]; then
    echo "Error: --mode is required" >&2
    usage
fi

if [ "$MODE" != "pre-push" ] && [ "$MODE" != "ci" ]; then
    echo "Error: --mode must be 'pre-push' or 'ci'" >&2
    usage
fi

info "Workflow linter running in ${MODE} mode"

# Determine base ref for diff
# In GitHub Actions, GITHUB_BASE_REF is set to the PR target branch
if [ -n "${GITHUB_BASE_REF:-}" ]; then
    BASE_REF="origin/${GITHUB_BASE_REF}"
else
    BASE_REF="origin/main"
fi

if ! git rev-parse --verify --quiet "${BASE_REF}" >/dev/null; then
    emit_warning \
        "advisory" \
        "Base ref '${BASE_REF}' not found; skipping diff-based workflow checks" \
        "Shallow or partially fetched clones can omit the branch the linter compares against, which would otherwise look like 'no changes'. Fetch the base branch locally before rerunning workflow-lint."
    DIFF_CHECKS_AVAILABLE=false
else
    # Get changed files relative to base
    # --diff-filter=D lists deleted files, ADMR lists added/deleted/modified/renamed
    if ! CHANGED_FILES=$(git diff --name-only --diff-filter=ADMR "${BASE_REF}...HEAD" 2>/dev/null); then
        emit_warning \
            "advisory" \
            "Unable to compute changed files relative to '${BASE_REF}'; skipping diff-based workflow checks" \
            "The repository state prevented git diff from computing the expected comparison range, so the linter will keep running only non-diff checks."
        DIFF_CHECKS_AVAILABLE=false
    fi

    if ! DELETED_FILES=$(git diff --name-only --diff-filter=D "${BASE_REF}...HEAD" 2>/dev/null); then
        emit_warning \
            "advisory" \
            "Unable to compute deleted files relative to '${BASE_REF}'; skipping diff-based workflow checks" \
            "The repository state prevented git diff from computing the expected comparison range, so the linter will keep running only non-diff checks."
        DIFF_CHECKS_AVAILABLE=false
    fi

    if $DIFF_CHECKS_AVAILABLE && [ -z "$CHANGED_FILES" ] && [ -z "$DELETED_FILES" ]; then
        info "No changes detected relative to ${BASE_REF}"
    fi
fi

# =============================================================================
# Check 1: Issue lifecycle (pre-push + ci)
# Files removed from docs/issues/ must appear in docs/issues/done/
# =============================================================================
check_issue_lifecycle() {
    if ! $DIFF_CHECKS_AVAILABLE; then
        return
    fi

    local deleted_issues
    deleted_issues=$(echo "$DELETED_FILES" | grep '^docs/issues/[^/]*\.md$' || true)

    if [ -z "$deleted_issues" ]; then
        return
    fi

    for issue_file in $deleted_issues; do
        local base_name
        local done_file
        base_name=$(basename "$issue_file")
        done_file="docs/issues/done/$base_name"
        # Check if the file was added to done/ in this diff
        if ! echo "$CHANGED_FILES" | grep -qF "$done_file"; then
            emit_warning \
                "fixable" \
                "Issue file '${issue_file}' was deleted instead of moved to done/" \
                "Issues must be preserved for audit trail (${WORKFLOW_DOC} Step 3)." \
                "git mv ${issue_file} docs/issues/done/${base_name}"
        fi
    done
}

# =============================================================================
# Check 2: Docs-change hint (ci only)
# If code files changed but no docs/ files changed, warn (unless [trivial])
# =============================================================================
check_docs_change_hint() {
    if ! $DIFF_CHECKS_AVAILABLE; then
        return
    fi

    if [ "$MODE" != "ci" ]; then
        return
    fi

    # Check for [trivial] marker in PR title or body
    if echo "$PR_TITLE" | grep -qi '\[trivial\]'; then
        return
    fi
    if echo "$PR_BODY" | grep -qi '\[trivial\]'; then
        return
    fi

    # Check if any code files changed (non-docs, non-config)
    local code_changed=false
    while IFS= read -r file; do
        [ -z "$file" ] && continue
        case "$file" in
            docs/*|*.md|.gitignore|.githooks/*|*.yml|*.yaml)
                # Not code files
                ;;
            *)
                code_changed=true
                break
                ;;
        esac
    done <<< "$CHANGED_FILES"

    if ! $code_changed; then
        return
    fi

    # Check if any docs/ files changed
    local docs_changed=false
    while IFS= read -r file; do
        [ -z "$file" ] && continue
        case "$file" in
            docs/*)
                docs_changed=true
                break
                ;;
        esac
    done <<< "$CHANGED_FILES"

    if ! $docs_changed; then
        emit_warning \
            "advisory" \
            "Code changed without updating docs/ (Spec-Code Parity review needed)" \
            "docs/specs/ should usually change with implementation updates (${WORKFLOW_DOC} Core Principle 2)."
    fi
}

# =============================================================================
# Check 3: Branch naming convention (pre-push + ci)
# Branch must match <type>/<description> where type is plan|feat|fix|chore|docs
# =============================================================================
check_branch_naming() {
    local branch
    branch=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || true)

    # Skip for main/master or detached HEAD
    if [ -z "$branch" ] || [ "$branch" = "main" ] || [ "$branch" = "master" ] || [ "$branch" = "HEAD" ]; then
        return
    fi

    local valid_types="plan|feat|fix|chore|docs"
    if ! echo "$branch" | grep -qE "^(${valid_types})/[a-z0-9]([a-z0-9-]*[a-z0-9])?$"; then
        emit_warning \
            "fixable" \
            "Invalid branch name: '${branch}'" \
            "Consistent naming enables automation and exec-plan mapping (${WORKFLOW_DOC} Branch Naming Convention)." \
            "Create a compliant branch with ww create <type>/<description> where type = plan|feat|fix|chore|docs and description is kebab-case (for example: feat/add-auth)"
    fi
}

# =============================================================================
# Check 4: Exec-plan existence (pre-push + ci)
# feat/* and fix/* branches require a matching exec-plan file
# =============================================================================
check_exec_plan_existence() {
    local branch
    branch=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || true)

    # Only check feat/* and fix/* branches
    if ! echo "$branch" | grep -qE "^(feat|fix)/"; then
        return
    fi

    local plan_name="${branch#*/}"
    local todo_file="docs/exec-plan/todo/${plan_name}.md"
    local done_file="docs/exec-plan/done/${plan_name}.md"

    if [ ! -f "$todo_file" ] && [ ! -f "$done_file" ]; then
        emit_warning \
            "fixable" \
            "Missing exec-plan for branch '${branch}'" \
            "feat/* and fix/* branches must have a plan before implementation (${WORKFLOW_DOC} Exec-Plan Mapping)." \
            "Create the matching plan first on plan/${plan_name}, then add docs/exec-plan/todo/${plan_name}.md"
    fi
}

# =============================================================================
# Check 5: Workflow docs should not reintroduce raw-git startup (pre-push + ci)
# Warn when migrated workflow-facing docs/skills contain startup snippets that
# bypass the global ww CLI.
# =============================================================================
check_workflow_doc_startup_commands() {
    if ! $DIFF_CHECKS_AVAILABLE; then
        return
    fi

    local workflow_files=(
        "AI_WORKFLOW.md"
        "AGENTS.md"
        "README.md"
        "skills/plan-execution/SKILL.md"
        "skills/execute-task/SKILL.md"
        "skills/triage-tasks/SKILL.md"
        "skills/plan-project/SKILL.md"
        "skills/review-task/SKILL.md"
        "skills/manage-workflow/SKILL.md"
    )
    # shellcheck disable=SC2016
    local raw_git_pattern='^[[:space:]]*git fetch origin([[:space:]]|$)|^[[:space:]]*git switch -c[[:space:]]|`git fetch origin`|`git switch -c [^`]+`|`git fetch origin && git switch -c [^`]+`'
    local file

    for file in "${workflow_files[@]}"; do
        if ! echo "$CHANGED_FILES" | grep -qxF "$file"; then
            continue
        fi

        if grep -nE "$raw_git_pattern" "$file" >/dev/null 2>&1; then
            emit_warning \
                "fixable" \
                "Workflow doc '${file}' reintroduces raw git startup commands" \
                "Normal planning/execution should dogfood the global ww CLI (${WORKFLOW_DOGFOOD_DOC})." \
                "Replace startup instructions with 'ww create ...' and 'cd \"\$(ww cd ...)\"'"
        fi
    done
}

# Run checks
check_issue_lifecycle
check_docs_change_hint
check_branch_naming
check_exec_plan_existence
check_workflow_doc_startup_commands

# Summary
if [ "$WARN_COUNT" -gt 0 ]; then
    echo -e "${YELLOW}Workflow linter summary:${NC}" >&2
    echo "  Total warnings: ${WARN_COUNT}" >&2
    echo "  Fixable: ${FIXABLE_WARN_COUNT}" >&2
    echo "  Advisory: ${ADVISORY_WARN_COUNT}" >&2
    if [ "$FIXABLE_WARN_COUNT" -gt 0 ]; then
        echo "  Reminder: resolve fixable warnings before push/PR unless a human instruction conflicts or the warning is a clear false positive." >&2
    fi
else
    info "Workflow linter: all checks passed"
fi

exit 0
