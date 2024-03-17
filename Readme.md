# Tommi2Day Go Modules

[![Go Report Card](https://goreportcard.com/badge/github.com/tommi2day/gomodules)](https://goreportcard.com/report/github.com/tommi2day/gomodules) 
![CI](https://github.com/tommi2day/gomodules/actions/workflows/main.yml/badge.svg)
[![codecov](https://codecov.io/gh/Tommi2Day/gomodules/branch/main/graph/badge.svg?token=4KLVC3TT6A)](https://codecov.io/gh/Tommi2Day/gomodules)
![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/tommi2day/gomodules)
[![Go Reference](https://pkg.go.dev/badge/github.com/tommi2day/gomodules.svg)](https://pkg.go.dev/github.com/tommi2day/gomodules)

this is a collection of my often used functions

- common: Common functions used in modules and implementations
- pwlib: 
  - password generation, 
  - password storing and handling with RSA, Openssl, GoPass, Amazon KMS and Hashicorp Vault
  - totp generation
  - scram hashing
- tools: collection of often used tools as build loader
- dblib: db related functions, esp. for oracle and tns handling
- maillib: function to send Mails
- ldaplib: base ldap functions
- hmlib: handle access to homematic devices using [XMLAPI-Addon](https://github.com/homematic-community/XML-API)

### usage
for usage see the provided test cases and the implemenations as is:

- [tnscli](https://github.com/tommi2day/tnscli)
- [pwcli](https://github.com/tommi2day/pwcli)
- [check_hm](https://github.com/Tommi2Day/check_hm)

### API
see [godoc](https://pkg.go.dev/github.com/tommi2day/gomodules)
