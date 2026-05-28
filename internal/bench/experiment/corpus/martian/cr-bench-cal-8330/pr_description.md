## Advanced date override handling and timezone compatibility improvements

## What does this PR do?

This PR fixes that date overrides of fixed hosts are taken into account for round-robin events. This prevents round-robin events being booked where the fixed hosts are unavailable. 

This PR also fixes that date overrides didn't adjust to different timezones. If the organizer has a date override from 10-11 UTC it always showed the slots from 10-11 open no matter what timezone the attendee has.

Fixes #8207
Fixes #8329

Fixes: #8273

**Environment**: Staging(main branch) / Production

## Type of change

- Bug fix (non-breaking change which fixes an issue)

## How should this be tested?

**Date overrides for fixed hosts (round-robin):** 
- Create a round-robin event with one fixed host and host round-robin host
- Create a date override for fixed host (round-robin host should also be available at that time)
- Check available slots of event on the day of the date override, it should only show the slots where fixed host is available
- Test that with different time zones on booking page and on the availabilities of fixed and round-robin host

**Date override timezones issue:** 
- Create a date override 
- Go to the booking page of a personal event 
- Change the timezone to a different one that the availability with the date override is 
- See the available slots of the day with the date override change accordingly 
- Also test with different timezones here 

