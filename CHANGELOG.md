# Go Library

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
