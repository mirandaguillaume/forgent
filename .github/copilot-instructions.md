# Project Instructions

## Available Skills

- **Git History Reviewer**: history-first-based skill consuming pr_diff, git_blame to produce review_issues
- **Issue Confidence Scorer**: scoring-based skill consuming review_issues, claudemd_files to produce scored_issues
- **Pr Eligibility Checker**: gate-check-based skill consuming pr_url to produce eligibility_status
- **Pr Summarizer**: diff-first-based skill consuming pr_url to produce pr_summary
- **Tdd Runner**: test-first-based skill consuming file_tree, source_code to produce test_results
- **Claudemd Discoverer**: file-discovery-based skill consuming pr_diff, file_tree to produce claudemd_files
- **Code Comment Auditor**: compliance-check-based skill consuming pr_diff, source_code to produce review_issues
- **Review Commenter**: diff-first-based skill consuming git_diff, test_results, lint_results to produce review_comments
- **Claudemd Auditor**: compliance-check-based skill consuming pr_diff, claudemd_files to produce review_issues
- **Pr History Reviewer**: history-first-based skill consuming pr_diff, pr_history to produce review_issues
- **Risk Scorer**: diff-first-based skill consuming git_diff, test_results, lint_results to produce risk_score
- **Type Checker**: static-analysis-based skill consuming file_tree, source_code to produce type_errors
- **Bug Scanner**: diff-first-based skill consuming pr_diff to produce review_issues
- **Coverage Reporter**: test-first-based skill consuming file_tree, source_code to produce coverage_report
- **Pr Commenter**: output-format-based skill consuming scored_issues, pr_url to produce review_comment
- **Ts Linter**: static-analysis-based skill consuming file_tree, source_code to produce lint_results

## Available Agents

- **Ci Reviewer**: Runs type-checking, linting, tests, coverage, then reviews the PR diff and scores risk
- **Staged Reviewer**: Multi-stage code review pipeline with preflight, analysis, and publish stages

## Global Guardrails

- timeout: 5min
- max_history_depth: 20
- timeout: 2min
- min_score_threshold: 80
- timeout: 1min
- fail_closed
- timeout: 2min
- max_summary_length: 500
- timeout: 10min
- max_retries: 2
- fail_fast_on_syntax_error
- timeout: 1min
- max_depth: 5
- timeout: 5min
- max_comments: 15
- timeout: 5min
- no_approve_without_tests
- timeout: 5min
- require_citation
- timeout: 5min
- max_prs: 10
- timeout: 3min
- require_risk_score
- timeout: 5min
- max_file_size: 500KB
- timeout: 5min
- no_nitpicks
- no_style_issues
- timeout: 10min
- timeout: 2min
- require_full_sha
- max_issues: 20
- timeout: 5min
- max_file_size: 500KB
