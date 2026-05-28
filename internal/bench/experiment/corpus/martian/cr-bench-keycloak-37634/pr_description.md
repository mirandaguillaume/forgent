## Implement access token context encoding framework

closes #37118

PR is adding some context information to be available in the `access-token` without a need to introduce any additional custom claims. So far, it provides info about:

- session type (online, offline, transient), which was used to create the token
- token type (regular or lightweight token)
- grant type

These informations can be useful when token is later sent to some Keycloak endpoints (for example introspection, userInfo, admin/account REST API) to optimize something. For example:
- session-type will allow to lookup only correct session type based on the type of the session, which token was created from. As for example today, we need to lookup "online" session and if it is not found, then fallback to "offline" session. So possible unnecessary lookup. Related follow-up issue for optimize that: #37662

- Grant type knowledge can be useful for example for the token-exchange use-case: #37117

- Lightweight access token: We will be able to recognize more reliably if access-token is lightweight or not and do something based on that (EG. lookup roles from model instead of lookup roles from the token)

In this PR, I've made token ID to look something like `onrtac:1f65761e-1c69-40a2-82da-2d8aa5e39a8a` .
  - The first 2 characters is shortcut for "session type" - `on` is shortcut for "online" session
  - The next 2 characters is shortcut for "token type" - `rt` is shortcut for regular token (not lightweight token)
  - The next 2 characters is shortcut for grant type - `ac` is shortcut for "authorization code" grant

Shortcuts are used, so the token ID contains important info, but at the same time, it is limited in size and does not introduce much new characters to the token.

The encoding into ID is used only for access-tokens. Not used for refresh tokens or ID tokens or any others.

Introduced dedicated SPI in the end. Wanted to "cache" some objects (EG. shortcuts for grant types etc) and it seems clearer to me to cache them at the level of factory, rather than in the static fields, which is not very pretty. Moreover, provider adds ability for someone to introduce his own provider (and eventually encode some more info or encode stuff in the different way etc).

#### Note

I've used 2 commits in this PR. Originally, in the 1st commit, I've made the encoding a bit longer with token ID to look something like `st.on_tt.rt_gt.ac:1f65761e-1c69-40a2-82da-2d8aa5e39a8a` .
  - The `st` is shortcut for "session type" and `on` is shortcut for "online" session
  - The `tt` is shortcut for "token type" and `rt` is shortcut for regular token (not lightweight token)
  - The `gt` is shortcut for grant type and `ac` is shortcut for "authorization code" grant

But assuming that every "type" has always 2 characters long shortcut, I've rather updated it in the 2nd commit to have only 7 new characters instead of 18 characters. I planned to squash before merge (unless you think that we should rather use longer format for some reason, in which case the 2nd commit can be removed from the PR).

