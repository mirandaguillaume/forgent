## feat: 2fa backup codes

## What does this PR do?

- generate and display backup codes during 2fa setup
- enable their use during 2fa login flow
- fix 2fa setup test to include new backup code screen

[backup-codes-setup.webm](https://github.com/calcom/cal.com/assets/55853254/3bebfb21-f7ef-4717-9135-ac30387cfb18)

[backup-codes-login.webm](https://github.com/calcom/cal.com/assets/55853254/71ef61b1-af6f-4e32-bf1e-455e52560535)

Fixes #9910
/claim #9910

## Type of change

- [x] New feature (non-breaking change which adds functionality)

## How should this be tested?

1. Setup 2fa to see the generated backup codes
2. Try to login with them by clicking on "Lost access"

## Mandatory Tasks

- [x] Make sure you have self-reviewed the code. A decent size PR without self-review might be rejected.

## Checklist

- I haven't added tests that prove my fix is effective or that my feature works
