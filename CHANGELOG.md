# Go Library

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
