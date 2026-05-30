# Go Library

## [v1.25.0 - 2026-05-30]
### New
- maillib: mail signature support with multiple signing methods
  - New `mail_signing.go` with `SignMailContent()` and `VerifyMailSignature()` functions
  - Supported signing methods: RSA, ECDSA, GPG, and S/MIME
  - `MailSignatureConfig` struct for flexible signing configuration
  - `MailType.SignMail()` method to sign mail content
  - `MailType.VerifyMailSignature()` method to verify mail signatures
  - Helper functions: `IsValidSigningMethod()`, `GetSupportedSigningMethods()`
  - Comprehensive test coverage with `mail_sign_test.go`
  - GPG signing uses temporary files to work with existing GPG infrastructure
  - S/MIME CMS/PKCS#7 detached signature support
  - New `BuildSMIMEMultipartSigned()` and `VerifySMIMEMultipartSigned()` helpers
  - `SendMail` now auto-builds `multipart/signed` payloads when S/MIME config is set
### Changed
- maillib: extended `MailType` struct with signature fields:
  - `Signature` field for storing signature data
  - `SignatureConfig` field for signature configuration
  - `IsSigned` and `SignatureVerified` boolean flags
- Updated Readme.md with new mail signature capabilities
- update dependencies to solve CVEs
- fix linter issues by fixing linter version
- update openldap container to 2.6.13
### Security
- maillib/smime: added `VerifyWithChain` to vendored pkcs7 library — S/MIME verification
  now validates the actual signer certificate against a caller-supplied trust pool
  (expiry + chain-of-trust), replacing the previous check that only tested certificate
  presence in the attacker-controlled certificate bag
- maillib/smime: `VerifySMIMEMultipartSigned` (no-pinned-cert path) now rejects signatures
  whose embedded certificates are outside their validity window
- maillib: `EnableSSL` / `EnableTLS` now correctly set `InsecureSkipVerify` from the
  `insecure` parameter instead of unconditionally hardcoding `true`
### Fixed
- maillib: IMAP docker-test connections updated to pass `insecure=true` for self-signed
  test-server certificates after the `InsecureSkipVerify` fix above

## [v1.24.3 - 2026-04-09]
### Changed
- pwlib: migrate gopass config parsing to native config format, remove YAML support, and update tests accordingly

## [v1.24.2 - 2026-03-27]
### Changed
- update dependencies to latest versions
- pwlib/gopass: add more tests
### Fixed
- pwlib/age: fix age decrypt with passphrase fallback

## [v1.24.0 - 2026-03-20]
### New
- pwlib/gpg: GPG agent integration library (`gpg_agent.go`) with Assuan protocol client for decryption via gpg-agent
- pwlib/gpg: `GPGDecryptFileAuto` — decrypt using all secret keys from the system keyring (secring.gpg or gpg binary export)
- pwlib/gpg: `GPGSystemSecretKeys`, `GPGSecretKeyRingPath`, `GPGReadSecretKeyRing`, `GPGExportSecretKeysArmored`
- pwlib/gpg: `GPGDetectRecipients` and `GPGFindDecryptKey` — inspect encrypted file headers without decryption
- pwlib/gopass: `GopassMounts` — enumerate all gopass stores (root + mounts) from config
- pwlib/gopass: `GOPASS_HOMEDIR` env var override for home directory resolution
- pwlib/gopass: `GOPASS_AGE_PASSWORD` env var support for age passphrase fallback
- pwlib/gopass: `GopassStoreDir` now also checks root path from gopass config and respects `XDG_DATA_HOME`
- pwlib/age: `AgeDecryptFileAuto` extended tests; improved resource handling in age functions
- common: `PromptPassword` function with tests
### Changed
- update dependencies to latest versions
### Fixed
- pwlib/gopass: `GopassRead` with empty `keyFile` now uses system GPG keyring (`GPGDecryptFileAuto`) instead of failing with a path error on Windows
- pwlib/gpg_agent: clear `GPG_AGENT_INFO` and `XDG_RUNTIME_DIR` in agent socket/decrypt tests to prevent CI environment's live gpg-agent from interfering

## [v1.23.0 - 2026-03-17]
### New
- pwlib/gopass: native Go functions for reading, writing, and listing gopass secrets (GPG and age)
- pwlib/gopass: auto-detect store encryption type from marker files and gopass config
- pwlib/gopass: support gopass method in GetPassword / DecryptFile
- pwlib/age: passphrase-encrypted age identity file support (ExportAgeKeyPairEncrypted, AgeLoadEncryptedIdentity)
- pwlib/age: passphrase-based file encryption/decryption (AgeEncryptFileWithPassphrase, AgeDecryptFileWithPassphrase)
- pwlib/age: auto-detect plaintext vs passphrase-protected identity (AgeDecryptFileAuto)
- pwlib/age: detect matching identity for an encrypted file (AgeDetectIdentity, AgeDetectIdentityWithPassphrase)
- pwlib/gpg: detect recipient key IDs from encrypted file header without decryption (GPGDetectRecipients)
- pwlib/gpg: find matching private key file for an encrypted file (GPGFindDecryptKey)
### Changed
- ldaplib/dblib: update ldap container to 2.6.12
### Fixed
- linter: suppress gosec false positives on path/env-var constants in gopass.go
- pwlib/vault: vault provision

## [v1.22.0 - 2026-02-15]
### New
- pwlib: add ECDSA key generation and signing functions
- pwlib: add signing functions and tests
- pwlib: add function to detect key type from file
- pwlib: enhance key type detection with GPG and Age
- pwlib: update openssl.go to handle RSA and ECDSA keys seamlessly

### Changed
- pwlib: pwconfig type handling
### Fixed
- pwlib: fix linter issues in rsa_test.go and ecdsa_test.go

## [v1.20.0 - 2026-02-13]
### Changed
- pwlib: make vault list recursive, return only path
- update dependencies

## [v1.19.2 - 2025-12-26]
### New
- ldaplib: add ldif functions
- common: add GetDockerHost, GetVersionedDockerPool and GetDockerAPIVersion functions and tests
- common: add FindCommand function and test
- pwlib: add age function

### Changed
- ldaplib, dblib: use clearstart/openldap as ldap test container
- reenable ldap test
- maillib: use dockerhost as mail test container
- symcon: always load stable container

### Fixed
- netlib,dblib: api version error when building docker image
- pwlib: panic in vault test

## [v1.18.1 - 2025-12-22]
### Changed
- update dependencies
- use go 1.25
- update mail and oracle test docker image
- disable ldap test as bitnami images are not available any more
- update workflow actions
- update golangci config for v2
### Fixed
- symcon: change TestWithoutProfile Test
- linter issues

## [v1.18.0 - 2025-03-08]
### New
- common: add StructToJSON function
### Changed
- pwlib: use $HOME/.pwcli as default dir
- update dependencies
- update test docker images

## [v1.17.1 - 2025-02-27]
### New
- common: add MergeMap and FindFileInPath functions and tests
- pwlib: add password profileset functions
- pwlib: case-insensitive match in get_password
### Changed
- vault: update vault test image

## [v1.16.0 - 2025-02-21]
### New
- common: add GetGitlabJobURL and GetGitlabPipelineURL functions
### Changed
- require Go1.23
- update dependencies

# [v1.15.0 - 2024-11-25]
### New
- common: add ReadFileToStruct function and tests
### Changed
- update dependencies

## [v1.14.11 - 2024-11-08]
### New
- common: add ReadStdinToString and ReadStdinByLine functions and tests
- dblib: add ssl tns 
### Changed
- dblib: modify tns ldap tests
- update dependencies

## [v1.14.10 - 2024-09-04]
### Changed
- update dns tests

## [v1.14.9 - 2024-08-31]
### Changed
- dns test docker creation error handling
### Fixed
- panic in docker_helper

## [v1.14.8 - 2024-08-23]
### New
- common: add StructToMap and StructToString functions
### Changed
- pwlib: refactor gpg gopass test files
- update linter github-action
- mail: setTimeout uses int64
- update test docker image versions

## [v1.14.7 - 2024-08-18]
### Changed
- netlib,dblib: change docker network range
- use common.WriteStringToFile/ReadFileToString instead of os.WriteFile and os.ReadFile
- update dependencies
### Fixed
- fix new linter issues

## [v1.14.4 - 2024-08-13]
### New
- common: add RandomString function and tests
- common: add WriteStringToFile function and tests
### Changed
- symcon: change no profile output
- update dependencies
### Fixed
- symcon: fix linter issues
- symcon: fix test


## [v1.14.3 - 2024-06-24]
### New
- symcon: add IsReady and IPSObject.String function
- symcon: add path to object and variable types
### Changed
- symcon: restruct GetObjectPath function
- symcon: refactor tests
- all: replace test init function

## [v1.14.2 - 2024-06-23]
### New
- symcon: add functions and tests
### Changed
- symcon: renamed functions to avoid name collision with native symcon function names

## [v1.14.1 - 2024-06-22]
### New
- add symcon module for [Symcon Json Api](https://www.symcon.de/service/dokumentation/datenaustausch/)
- add symcon tests
- common: add HTTPGet function
- common: add ReverseMap function
- common: add GetHexInt64Val function
- hmlib: add SetDebug function
### Changed
- update dependencies
- fix linter issues

## [v1.13.3 - 2024-05-25]
### Changed
- use Go1.22
- update dependencies

## [v1.13.2 - 2024-05-24]
### New
- pwlib: add SSHA functions
### Changed
- update dependencies

## [v1.13.1 - 2024-04-23]
### New
- maillib: add SetAuthMethod function
### Changed
- update dependencies

## [v1.13.0 - 2024-04-11]
### New
- common: add CommandExists function
- common: add DefaultPorts map
- test: add InitTestDirs function
- new module netlib for IP/DNS related functions
### Changed
- use go v1.21
- update dependencies
- update linter
- dblib: replace rename dns vars to dblibDNS
- dblib: replace local ip dns functions with netlib
- dblib: add more tests
- dblib: rename dns docker dir to oracle-dns
### Fixed
- GetHostPort function when no port supplied
- linter issues


## [v1.12.1 - 2024-04-01]
### Changed
- update dependencies

## [v1.12.0 - 2024-03-17]
### New
- pwlib: add amazon kms encryption methods
### Changed
- update dependencies

## [v1.11.5 - 2024-03-01]
### New
- pwlib: add scram hash method
### Changed
- update dependencies

## [v1.11.4 - 2024-02-18]
### Changed
- hmlib: use plain url insead of httpclient query params encoded strings
- hmlib: change sysvar structure and output

## [v1.11.3 - 2024-02-16]
### Changed
- dblib: use bitnami/openldap as test container
- maillib: use mailserver:13.2.0 and refactor tests to fit there

## [v1.11.2 - 2024-02-13]
### New
- ldaplib: add new function RetrieveEntry
- ldaplib: add new function HasObjectClass
- ldaplib: add new function HasAttribute
- ldaplib: add schemas
- ldaplib: use bitnami/openldap as test container
### Changed
- update dependencies
- update GitHub workflows to v4
### Fixed
- linter issues

## [v1.11.1 - 2024-01-25]
### New
- common: add IsNumeric function
- common: add FormatUnixtsString function
- hmlib: add GetDeviceIDofChannel function
- hmlib: add GetChannelIDofDatapoint function
### Changed
- hmlib: modify object string format
- hmlib: add more tests

## [v1.11.0 - 2024-01-20]
### New
- add hmlib module for using Homematic Devices using [XMLAPI-Addon](https://github.com/homematic-community/XML-API)
### changed
- move docker resources to separate folder
- update dependencies
- maillib: use mailserver 13.2.0 as test container
- pwlib: use vault 1.15.4 test container
### fixed
- NPE on dblib container purge

## [v1.10.4 - 2024-01-12]
### New
- dblib: add wait till the init_done table indicates db is ready
- dblib: add DBLogout function
### Changed
- update dependencies
- dblib: rename variables
- dblib: change oracle port
- ldaplib: increase time for provisioning to 15s
- remove tools.go

## [v1.10.3 - 2023-11-13]
### New
- pwlib: add gpg and gopass method and tests
- add codecov.yml
### Changed
- update dependencies

## [v1.10.2 - 2023-11-03]
### Changed
- dblib: use Oracle-Free 23.3 as test container, which causes to replace XEPDB1 with FREEPDB1
### Fixed
- dblib: oracle startup wait retry function

## [v1.10.1 - 2023-11-01]
### New
- common: add GetHostname function
- maillib: add setHELO function and set Helo in Connect
- dblib: add GetJDBCUrl function with ModifyJDBCTransportConnectTimeout flag (default true) 
to update build jdbc url from tns entry and replace TRANSPORT_CONNECT_TIMEOUT in ms if <1000

# [v1.10.0 - 2023-10-27]
### New
- use go 1.21
- pwlib: expose GenerateRandomString function
### Fixed
- linter issues

# [v1.9.6 - 2023-10-19]
### New
- common: add Git functions
- common: add InArray function
- common: add isDir and isFile functions
### Changed
- common: update tests

# [v1.9.5 - 2023-10-10]
### New
- common: add FileExists function
- common: add CanRead function
- common: add more tests
### Changed
- common: split common.go and tests into separate files for net, file and type functions

# [v1.9.4 - 2023-08-10]
### New
- dblib: add more ExecSQL functions and tests

# [v1.9.3 - 2023-08-09]
### New
- common: add isNil function and tests
- common: add CheckType function and tests
### Changed
- dblib: refactor type checks
- dblib: move checkType to common

# [v1.9.2 - 2023-08-08]
### New
- common: add Cobra Command helper
- dblib: add more sql functions and tests
### Changed
- dblib: use github.com/jmoiron/sqlx instead of database/sql

# [v1.9.1 - 2023-07-22]
### New
- add line number to location
### Changed
- rename TNSEntry.Filename to TNSEntry.Location
- update tests

# [v1.9.0 - 2023-07-16]
### New
- common: add URL and Host parsing functions
- common:add more tests
- common: add dockertest helper
- dblib: add RACInfo Lookup per INI and DNS SRV record
### Changed
- use go 1.20
- update dependencies
- use docker_helper for tests
### Fixed
- dblib: fix tns server parsing RegExp

# [v1.8.1 - 2023-06-22]
### Changed
- dblib: enhance ldap functions and test

# [v1.8.0 - 2023-06-19]
### Changed
- dblib: move write tns ldap functions to tnscli
- dblib: add/remove tests

## [v1.7.4 - 2023-05-19]
### Fixed
- pwlib: parsing PKCS1 (openssl traditional) RSA keys

## [v1.7.3 - 2023-05-17]
### New
- common: type converter functions

## [v1.7.2 - 2023-05-16]
### Fixed
- dblib: fix SID parsing

## [v1.7.1 - 2023-05-10]
### New
- pwlib: add vault method to get_password

## [v1.7.0 - 2023-04-24]
### New
- pwlib: add [Hashicorp Vault](https://developer.hashicorp.com/vault) KV2 and Logical API functions

## [v1.6.0 - 2023-04-09]
### New
- maillib: add Imap functions
- ldaplib: refactor types and functions, add timeout
- pwlib: add totp generator
- pwlib: add plain and base64 encode password methods
### Changed
- align test init
- make some interfaces type based

## [v1.5.0 - 2023-02-24]
### New
- ldaplib: add write functions and tests
- dblib: add TNS Ldap read and write functions and tests

## [v1.4.0 - 2023-02-21]
### New
- add maillib module
- add ldaplib module
- add ExecuteOsCommand to common

## [v1.3.0 - 2023-02-08]
### New
- add encryption method option to config for go and openssl
- add dblib module

## [v1.1.0 - 2023-02-07]
### New
- add main to track version
### Changed
- use gitlab prefix

## [v1.0.0 - 2023-02-06]
initial load
### New
- common functions
- pwlib functions
- tests
