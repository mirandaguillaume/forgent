## Database Performance Optimizations

**What is this feature?**

Writes to annotations are sometimes rejected on MySQL due to a recurring deadlock with the annotation cleanup job.
Details here: https://github.com/grafana/grafana/issues/64979

**Why do we need this feature?**

The deadlock seems to be a result of the subquery, under normal circumstances a plain insert and delete should not lock, and there are no extended transactions at play.

This PR splits the subquery into a separate SQL statement. These statements do not share a transaction, and therefore allow locks to flush in between.

Unfortunately we could not reduce this to a single DELETE statement, due to the batching - DELETE LIMIT is not supported on all databases without needing to reintroduce the subquery. The IDs loaded into memory are of bounded size due to the batch size configuration.

**Which issue(s) does this PR fix?**:

https://github.com/grafana/grafana/issues/64979

**Special notes for your reviewer:**

Please check that:
- [x] It works as expected from a user's perspective.
- [x] If this is a pre-GA feature, it is behind a feature toggle.
- [x] The docs are updated, and if this is a [notable improvement](https://grafana.com/docs/writers-toolkit/writing-guide/contribute-release-notes/#how-to-determine-if-content-belongs-in-a-whats-new-document), it's added to our [What's New](https://grafana.com/docs/writers-toolkit/writing-guide/contribute-release-notes/) doc.
