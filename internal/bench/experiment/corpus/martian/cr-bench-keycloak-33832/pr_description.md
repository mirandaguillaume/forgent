## Add AuthzClientCryptoProvider for authorization client cryptographic operations

…tory

closes #33831

This is the Keycloak main part for solving the https://github.com/keycloak/keycloak/issues/32962 and adding the related changes in the authz-client on the Keycloak side, which can then be later synced to the keycloak-client repository. 

In the end, I am using approach (2) from https://github.com/keycloak/keycloak/issues/32962 with adding the method `order()` to `CryptoProvider`, which allows to have multiple instances of `CryptoProvider` on classpath (Keycloak testsuite still uses the `DefaultCryptoProvider` as before this PR).