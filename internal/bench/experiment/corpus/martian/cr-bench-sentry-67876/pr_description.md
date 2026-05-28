## GitHub OAuth Security Enhancement

We're adding one more step in the GitHub integration installation pipeline, namely GitHub OAuth2 authorize. This is transparent from the UX perspective as the data exchange happens without user interaction.

The pipeline will now fail in these cases:
- If there is a mismatch between currently authenticated GitHub user (derived from OAuth2 authorize step) and the user who installed the GitHub app (https://github.com/apps/sentry-io)
- If there is a mismatch between `state` parameter supplied by user and pipeline signature
- If GitHub could not generate correct `access_token` from the `code` (wrong or attempt of re-use of `code`).

In all those cases, this error is shown:
![image](https://github.com/getsentry/sentry/assets/1127549/18923861-2ead-4cf5-adda-7738aef801d7)
