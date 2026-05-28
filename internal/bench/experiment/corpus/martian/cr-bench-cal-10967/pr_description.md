## fix: handle collective multiple host on destinationCalendar

## What does this PR do?

- Converts destinationCalendar on CalendarEvent to be an array.
- Patches and fixes so every sentence of code that was using evet.destinationCalendar can support the new type of array.
- The logic here is to fallback to normal behaviour has having one main destinationCalendar but also supports multiple so host can get invited correctly.

Fixes #7754 
## Loom 
Creating event and rescheduling:
https://www.loom.com/share/3cb3af513d274ed2bdfed44e48edddf6?sid=d81ce3de-2893-4a30-b085-ff67c31b7dfb

Canceling event:
https://www.loom.com/share/7ef505d4665e4d4982d2bb2d7b118085?sid=cd43c97c-aa86-4667-979f-f6492adf3365

## Requirement/Documentation

## Type of change

<!-- Please delete bullets that are not relevant. -->

- [X] Bug fix (non-breaking change which fixes an issue)

## How should this be tested?

- Are there environment variables that should be set? No
- What are the minimal test data to have? Setup a collective event using teampro and teamfree users. Add any app calendar to those accounts and add them as host for collective event.
- What is expected (happy path) to have (input and output)? Collective event with 2 host and every user with any calendar app installed should receive proper calendar events.
- Any other important info that could help to test that PR

## Mandatory Tasks

- [X] Make sure you have self-reviewed the code. A decent size PR without self-review might be rejected.

## Checklist

- I haven't added tests that prove my fix is effective or that my feature works
