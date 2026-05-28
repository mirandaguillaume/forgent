## Advanced SQL Analytics Framework

<!--

Thank you for sending a pull request! Here are some tips:

1. If this is your first time, please read our contribution guide at https://github.com/grafana/grafana/blob/main/CONTRIBUTING.md

2. Ensure you include and run the appropriate tests as part of your Pull Request.

3. In a new feature or configuration option, an update to the documentation is necessary. Everything related to the documentation is under the docs folder in the root of the repository.

4. If the Pull Request is a work in progress, make use of GitHub's "Draft PR" feature and mark it as such.

5. If you can not merge your Pull Request due to a merge conflict, Rebase it. This gets it in sync with the main branch.

6. Name your PR as "<FeatureArea>: Describe your change", e.g. Alerting: Prevent race condition. If it's a fix or feature relevant for the changelog describe the user impact in the title. The PR title is used to auto-generate the changelog for issues marked with the "add to changelog" label.

7. If your PR content should be added to the What's New document for the next major or minor release, add the **add to what's new** label to your PR. Note that you should add this label to the main PR that introduces the feature; do not add this label to smaller PRs for the feature.

-->

**What is this feature?**

Remove reference to DuckDB

**Why do we need this feature?**

See the blog post describing the vulnerability, for which a fix was rolled out to all Grafana versions earlier today:
https://grafana.com/blog/2024/10/17/grafana-security-release-critical-severity-fix-for-cve-2024-9264/

The full extent of our documentation around this feature seems to be here:
https://github.com/grafana/grafana/pull/81666/files#diff-16fb71b28e5b6197615abba8a07bc73d9bf75ebc60311663e1c29976de5cc9a6

All we've documented is that the feature-flag exists. That flag still exists in the codebase, so I'm tempted not to touch that documentation for now. We can always follow up with more improvements after this PR.

**Who is this feature for?**

Everyone

**Which issue(s) does this PR fix?**:

This moves the security fix into Grafana repo itself, now that it has been publicly announced. We need this here so we can clean up and simplify our Security-releases pipeline once again.

**Special notes for your reviewer:**

Please check that:
- [x] It works as expected from a user's perspective.
   Behaviour of this patch was already verified manually by @toddtreece. See his confirmation of this on our private slack, [here](https://raintank-corp.slack.com/archives/C07P60RBAUD/p1727902629844259?thread_ts=1727901896.911799&cid=C07P60RBAUD)
   Tested again on the Ephemeral instance here: https://ephemeral1511182194942samjewe.grafana-dev.net/d/be18gem24w2dcf/test-sql-expressions-bugfix?from=now-1h&to=now&timezone=browser
- [ ] If this is a pre-GA feature, it is behind a feature toggle.
- [ ] The docs are updated, and if this is a [notable improvement](https://grafana.com/docs/writers-toolkit/contribute/release-notes/#how-to-determine-if-content-belongs-in-whats-new), it's added to our [What's New](https://grafana.com/docs/writers-toolkit/contribute/release-notes/) doc.
