## Comprehensive workflow reminder management for booking lifecycle events

## What does this PR do?

Fixes that workflow emails are still sent for cancelled and rescheduled bookings. The fix of PR #6991 was wrong and didn't work as expected. 

The SendGrid endpoint `DELETE /v3/user/scheduled_sends/${referenceId}` was not used correctly. This endpoint deletes a cancelled scheduled email from `scheduled_sends` by removing the 'cancel' status (so email will again be scheduled). 

The reason why cancelling our scheduled emails stopped working is that all cancerlled emails are saved in `scheduled_sends` until the scheduled date but the max. of pending cancellations is 100. Once we reached 100 pending cancellations, new cancellations failed and emails were still sent out. 

The solution implemented in this PR: 
- A new field `cancelled` is added to the `workflowReminder` model
- Once an already scheduled email is cancelled, the field is set to true (nothing else is done at this point in time)
- The CRON job `scheduleEmailReminders` is responsible for the final cancellation of the scheduled email (runs every 15 mins)
  - cancels all scheduled emails that are cancelled and that have their scheduled date within the next hour

Fixes #7225

**Environment**: Staging(main branch) / Production

## Type of change

- [x] Bug fix (non-breaking change which fixes an issue)
- [ ] Chore (refactoring code, technical debt, workflow improvements)
- [ ] New feature (non-breaking change which adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] This change requires a documentation update
