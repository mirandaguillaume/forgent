## OAuth credential sync and app integration enhancements

## What does this PR do?

<!-- Please include a summary of the change and which issue is fixed. Please also include relevant motivation and context. List any dependencies that are required for this change. -->

This PR allows self-hosters that have their own app store, to sync their user's 3rd party credentials with their instance of Cal.com. This allows their app store to be the source of truth for apps.

Fixes # (issue)

<!-- Please provide a loom video for visual changes to speed up reviews
 Loom Video: https://www.loom.com/
-->

## Requirement/Documentation

<!-- Please provide all documents that are important to understand the reason of that PR. -->

- If there is a requirement document, please, share it here.
- If there is ab UI/UX design document, please, share it here.

## Type of change

<!-- Please delete bullets that are not relevant. -->

- [X] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [X] This change requires a documentation update

## How should this be tested?

<!-- Please describe the tests that you ran to verify your changes. Provide instructions so we can reproduce. Please also list any relevant details for your test configuration. Write details that help to start the tests -->

- Set up the required .env variables that were added in the PR
- Hit the `api/webhook/app-credential` endpoint with the correct webhook secret and encrypted payload
- The new credentials should be created in the DB
- For the affected apps, force a refresh of the tokens. The endpoint they are hitting should simulate what a self-hoster would have made
- Force a refresh of the tokens but this time do not enable app credential syncing

## Mandatory Tasks

- [x] Make sure you have self-reviewed the code. A decent size PR without self-review might be rejected.

## Checklist

<!-- Please remove all the irrelevant bullets to your PR -->

- I haven't added tests that prove my fix is effective or that my feature works
- I haven't checked if new and existing unit tests pass locally with my changes
