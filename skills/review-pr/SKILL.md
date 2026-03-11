---
name: review-pr
description: Review a PR or branch diff using the knowledge graph for full structural context. Outputs a structured review with blast-radius analysis.
argument-hint: "[PR number or branch name]"
---

# Review PR

Perform a comprehensive code review of a pull request or branch diff using the knowledge graph.

**Token optimization:** Before starting, call `get_docs_section(section_name="review-pr")` for the optimized workflow. Never include full files unless explicitly asked.

## Steps

1. **Identify the changes** for the PR:
   - If a PR number or branch is provided, use `git diff main...<branch>` to get changed files
   - Otherwise auto-detect from the current branch vs main/master

2. **Update the graph** by calling `build_or_update_graph(base="main")`.

3. **Get the full review context** by calling `get_review_context(base="main")`.

4. **Analyze impact** by calling `get_impact_radius(base="main")`.

5. **Deep-dive each changed file**:
   - Read the full source of files with significant changes
   - Use `query_graph(pattern="callers_of", target=<func>)` for high-risk functions
   - Use `query_graph(pattern="tests_for", target=<func>)` to verify test coverage

6. **Generate structured review output**:

   ```
   ## PR Review: <title>

   ### Summary
   <1-3 sentence overview>

   ### Risk Assessment
   - **Overall risk**: Low / Medium / High
   - **Blast radius**: X files, Y functions impacted
   - **Test coverage**: N changed functions covered / M total

   ### File-by-File Review
   #### <file_path>
   - Changes: <description>
   - Impact: <who depends on this>
   - Issues: <bugs, style, concerns>

   ### Recommendations
   1. <actionable suggestion>
   ```
