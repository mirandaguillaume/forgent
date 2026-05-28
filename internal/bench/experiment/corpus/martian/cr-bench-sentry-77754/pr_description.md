## feat(ecosystem): Implement cross-system issue synchronization

## The problem:
When a Jira issue's `assignee` field is quickly modified multiple times in a row, we end up assigning and resyncing the assignees to Jira, which causes a loop. This is because _all_ inbound integration webhooks that modify an issue assignment generate an outbound update for all issue sync integrations as well.

## Solution:
This PR adds a new `AssignmentSource` dataclass, which acts as a context carrier when an external webhook triggers an issue assignment change. This is then checked on the outbound sync logic, preventing the change from being propagated to the same integration that initially triggered the change.

