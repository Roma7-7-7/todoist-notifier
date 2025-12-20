---
description: Review git changes and update all related documentation
tools:
  read: true
  edit: true
  bash: true
---

You are reviewing code changes to ensure documentation stays in sync. Follow these steps:

1. **Analyze the changes:**
   - Run `!git diff HEAD` to see staged changes
   - If no staged changes, run `!git diff main` to see all changes on current branch
   - Identify what has changed (new features, API changes, configuration updates, architectural changes)

2. **Identify documentation to update:**
   - CLAUDE.md - for build commands, architecture, development notes
   - README.md - for user-facing documentation, setup instructions
   - Code comments - for complex logic or public APIs
   - Any other .md files in the repository

3. **Update documentation:**
   - Read each relevant documentation file
   - Update sections that are affected by the code changes
   - Add new sections if new features or patterns were introduced
   - Remove or update outdated information
   - Ensure examples and commands are still accurate

4. **Verify completeness:**
   - Check if any new configuration options need documenting
   - Verify environment variables are documented
   - Ensure new dependencies are mentioned if relevant
   - Confirm build/test commands still work as documented

5. **Report:**
   - Summarize what documentation was updated
   - Note any documentation that didn't need changes
   - Highlight if any changes need human review

Be thorough but focused - only update documentation that is actually affected by the code changes.
