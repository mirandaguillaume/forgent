## Dual Storage Architecture

**What is this feature?**

Mode 3 will work the following way:

reads (get, list) : only use unified storage
writes (create, update, delete, delete-collection) : write to unified storage and then asynchronously, write to legacy, just as a safe measure.

**Why do we need this feature?**

[Add a description of the problem the feature is trying to solve.]

**Who is this feature for?**

[Add information on what kind of user the feature is for.]

**Which issue(s) does this PR fix?**:

Fixes https://github.com/grafana/search-and-storage-team/issues/53

**Special notes for your reviewer:**

Please check that:
- [ ] It works as expected from a user's perspective.
- [ ] If this is a pre-GA feature, it is behind a feature toggle.
- [ ] The docs are updated, and if this is a [notable improvement](https://grafana.com/docs/writers-toolkit/writing-guide/contribute-release-notes/#how-to-determine-if-content-belongs-in-a-whats-new-document), it's added to our [What's New](https://grafana.com/docs/writers-toolkit/writing-guide/contribute-release-notes/) doc.
