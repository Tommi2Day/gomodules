# pwlib Go Library

this is a collection of often used encryption and password related functions, see tests for usage

- password generation
- password storing and handling with Go Crypt, RSA, Openssl, Age, Amazon KMS, Amazon Secrets Manager and Hashicorp Vault
- totp generation
- scram(e.g.for postgresql) and ssha(e.g for LDAP userPassword) hashing

## Usage
for usage see the provided test cases

## AWS Profile and MFA

By default KMS and Secrets Manager use the AWS SDK default credential chain. Optionally a profile from
the shared config (`~/.aws/config`, `~/.aws/credentials`) and an MFA token can be set before connecting:

```go
// profile without MFA
pwlib.SetAWSProfile("myprofile", "")
// profile with mfa_serial, token code from the authenticator app
pwlib.SetAWSProfile("myprofile", "123456")
// or ask for the token on demand
pwlib.SetAWSProfileWithTokenProvider("myprofile", stscreds.StdinTokenProvider)
svc := pwlib.ConnectToKMS()
```

Profiles with `role_arn` and `mfa_serial` pass the token to STS `AssumeRole`, profiles with only
`mfa_serial` (IAM user) get temporary credentials via STS `GetSessionToken`. Because an MFA token can
be used only once, the resulting AWS config is cached and shared by KMS and Secrets Manager until
`SetAWSProfile`/`ResetAWSConfig` is called again. The principal additionally needs `sts:AssumeRole`
on the role or `sts:GetSessionToken` respectively.

## AWS RDS IAM Authentication

`GetRDSAuthToken(endpoint, region, dbUser)` returns an IAM auth token for an RDS database (MySQL,
MariaDB, PostgreSQL), which is used as password and is valid for 15 minutes. `endpoint` is
`host[:port]` (default port `RDSDefaultPort`, 5432); the region is taken from the parameter, the env var
`RDS_REGION`, `RDSRegion` or the AWS config. Credentials follow the AWS profile/MFA settings above.

With method `rds` `PassConfig.GetPassword(system, account)` returns such a token, where `system` is the
RDS endpoint and `account` the database user:

```go
pc := pwlib.NewConfig("myapp", "", "", "", "rds")
token, err := pc.GetPassword("mydb.xxx.eu-central-1.rds.amazonaws.com:5432", "dbuser")
```

The principal needs `rds-db:connect` on
`arn:aws:rds-db:<region>:<account-id>:dbuser:<db-resource-id>/<db-user>`, and the database user must
be enabled for IAM authentication (e.g. `GRANT rds_iam TO dbuser;` in PostgreSQL).

## AWS IAM Permissions

pwlib talks to AWS KMS and AWS Secrets Manager via the standard AWS SDK default credential chain
(env vars, shared config/credentials file, EC2/ECS/EKS instance role, ...). The IAM principal used
to run an application built on pwlib needs the permissions below, split into the actions actually
called by each function group. Replace `<region>`, `<account-id>` and the resource identifiers with
your own values, and scope `Resource` down as far as your setup allows instead of using `*`.

### KMS (`kms.go`)

Runtime usage - `KMSEncryptString`/`KMSDecryptString`, `KMSSignString`/`KMSVerifyString`,
`KMSEncryptFile`/`KMSDecryptFile` and `PassConfig.GetPassword`/`EncryptFile`/`DecryptFile`/
`SignFile`/`VerifyFile` when `Method` is `kms` - only needs access to the specific key(s) in use:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "PwlibKMSUsage",
      "Effect": "Allow",
      "Action": [
        "kms:Encrypt",
        "kms:Decrypt",
        "kms:Sign",
        "kms:Verify"
      ],
      "Resource": "arn:aws:kms:<region>:<account-id>:key/<key-id>"
    }
  ]
}
```

Administrative usage - `GenKMSKey`, `DescribeKMSKey`, `ListKMSKeys`, `CreateKMSAlias`,
`DeleteKMSAlias`, `ListKMSAliases`, `DescribeKMSAlias` - typically only needed by provisioning
tooling, not by the runtime application. `kms:CreateKey`, `kms:ListKeys` and `kms:ListAliases` are
account-level actions and do not support resource-level restriction, so they require
`Resource: "*"`; `kms:TagResource` is required because `GenKMSKey` tags the key on creation:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "PwlibKMSAdmin",
      "Effect": "Allow",
      "Action": [
        "kms:CreateKey",
        "kms:TagResource",
        "kms:ListKeys"
      ],
      "Resource": "*"
    },
    {
      "Sid": "PwlibKMSAdminOnKey",
      "Effect": "Allow",
      "Action": [
        "kms:DescribeKey",
        "kms:CreateAlias",
        "kms:DeleteAlias"
      ],
      "Resource": "arn:aws:kms:<region>:<account-id>:key/*"
    },
    {
      "Sid": "PwlibKMSListAliases",
      "Effect": "Allow",
      "Action": "kms:ListAliases",
      "Resource": "*"
    }
  ]
}
```

### Secrets Manager (`secretsmanager.go`)

Read-only usage - `SecretsManagerRead`/`SecretsManagerReadJSON`, `GetAWSSMSecret` and
`PassConfig.GetPassword`/`DecryptFile` when `Method` is `awssm`:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "PwlibSecretsManagerRead",
      "Effect": "Allow",
      "Action": "secretsmanager:GetSecretValue",
      "Resource": "arn:aws:secretsmanager:<region>:<account-id>:secret:<name-prefix>*"
    }
  ]
}
```

Write usage - `SecretsManagerWrite` calls `PutSecretValue` and falls back to `CreateSecret` when the
secret does not exist yet; `SecretsManagerList` calls `ListSecrets`, which is an account-level action
and requires `Resource: "*"`. If a customer managed KMS key is set via `SecretsManagerKMSKeyID` or the
env var `SECRETSMANAGER_KMS_KEY_ID` (key ID, ARN or alias), `SecretsManagerWrite` uses `UpdateSecret`
instead of `PutSecretValue`, so existing secrets are switched to that key, and new secrets are created
with it; this additionally requires `secretsmanager:UpdateSecret`:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "PwlibSecretsManagerWrite",
      "Effect": "Allow",
      "Action": [
        "secretsmanager:PutSecretValue",
        "secretsmanager:CreateSecret",
        "secretsmanager:UpdateSecret"
      ],
      "Resource": "arn:aws:secretsmanager:<region>:<account-id>:secret:<name-prefix>*"
    },
    {
      "Sid": "PwlibSecretsManagerList",
      "Effect": "Allow",
      "Action": "secretsmanager:ListSecrets",
      "Resource": "*"
    }
  ]
}
```

Note: if a secret is encrypted with a customer-managed KMS key instead of the default
`aws/secretsmanager` key, the same principal also needs `kms:GenerateDataKey` (for writes) and
`kms:Decrypt` (for reads) on that KMS key, since Secrets Manager uses envelope encryption via KMS.

### used by:
- [pwcli](https://git.hv.devk.de/dba-cloud/goproj/pwcli)
- [coupa_ldap_abgleich](https://git.hv.devk.de/dba-cloud/goproj/coupa_ldap_abgleich)
