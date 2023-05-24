# Drone Vault GPG signing plugin
This plugin allows to sign artifacts inside a Drone CI pipeline using the custom [Vault GPG secret backend](https://github.com/LeSuisse/vault-gpg-plugin).

## Configuration
### Plugin settings

* `key`: Named Vault GPG key to sign the artifacts with. Defaults to `drone`.
* `armor`: Write ASCII-armored detached signatures (`.asc`) instead of binary ones (`.sig`). Defaults to `false`.
* `mount`: Mount the GPG secret backend is mounted at. Defaults to `gpg`.
* `algo`: Specifies the hash algorithm to use. Defaults to `sha2-256`.
* `auth`: Specifies the auth method to use with Vault. Defaults to `token`. Valid: `token`, `approle`.
* `authmount`: Specifies the mount the AppRole auth backend is mounted at. Defaults to `approle`.
* `roleid`: Specifies the Vault AppRole RoleID to authenticate with.
* `secretid`: Specifies the Vault AppRole SecretID to authenticate with.
* `wrapped_secret`: Indicates that the authentication secret is passed as a wrapping token. Defaults to `false`.
* `files`: List of files to sign.
* `exclude`: List of exclude patterns.

### Vault settings
This plugin respects all Vault client environment variables. For a comprehensive list, see the [client source](https://github.com/hashicorp/vault/blob/main/api/client.go).

#### Important
1. Specify the server URI in `VAULT_ADDR`. This is required.
2. If `token` auth is in use, you will need to specify `VAULT_TOKEN` as well.

#### Possibly relevant
* `VAULT_CACERT_BYTES`: A PEM-encoded certificate to accept as the only root CA.
* `VAULT_SKIP_VERIFY`: Skip TLS certificate verification.

## Example pipeline (rough)

```yaml
---
kind: pipeline
type: docker
name: build_publish

platform:
  os: linux
  arch: amd64

steps:
  - name: build
    image: golang
    environment:
      CGO_ENABLED: 0
    commands:
      - GOOS=linux GOARCH=amd64 go build -o dist/gpg-secret-plugin_${DRONE_TAG}_linux_amd64 .
      - cd ./dist; find * -type f -name 'gpg-secret-plugin*' -exec shasum -a 256 {} \; > SHA256

  - name: sign
    image: git.example.name/drone/vault-gpg-sign
    environment:
      VAULT_ADDR:
        from_secret: vault_addr
      VAULT_CACERT_BYTES:
        from_secret: vault_cacert
    settings:
      files:
        - dist/SHA256
      auth: approle
      roleid:
        from_secret: vault_roleid
      secretid:
        from_secret: vault_secretid

  - name: publish-artifacts
    image: plugins/gitea-release
    settings:
      api_key:
        from_secret: gitea_api_key
      base_url:
        from_secret: gitea_api_url
      files: dist/*

trigger:
  event:
    - tag
---
kind: secret
name: gitea_api_key
get:
  path: secrets/drone/gitea_api
  name: key
---
kind: secret
name: gitea_api_url
get:
  path: secrets/drone/gitea_api
  name: url

# TODO: Extend Vault secret plugin to issue creds
# to avoid having to write the following data to Drone secrets
# (especially secret ID, this should be issued automatically and
# done via response wrapping)
---
kind: secret
name: vault_addr
get:
  path: secrets/drone/vault
  name: url
---
kind: secret
name: vault_cacert
get:
  path: secrets/drone/vault
  name: cacert
---
kind: secret
name: vault_roleid
get:
  path: secrets/drone/vault
  name: roleid
---
kind: secret
name: vault_secretid
get:
  path: secrets/drone/vault
  name: secretid
```

## Notes
* This plugin is in very early development.
* Getting Vault auth credentials for a pipeline step is a bit difficult at the moment, but I plan to adapt the official `drone-vault` secret plugin to issue wrapped secret IDs as secrets as well as to expose other required connection configurations.

## Related
* https://github.com/drone-plugins/drone-gpgsign
