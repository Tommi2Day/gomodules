# Mail Signing and Signature Verification

This document describes the mail signing and verification functionality in the `maillib` package.

## Overview

The mail signing module provides support for digitally signing email content and verifying signatures using multiple cryptographic methods. This is essential for:

- Email authentication and non-repudiation
- Compliance with security standards
- Protection against email spoofing and tampering
- Enterprise email security

## Supported Signing Methods

### 1. RSA (Recommended for Most Use Cases)

**Characteristics:**
- Uses RSA-2048 bit keys with SHA-256 hashing
- Industry standard since decades
- Signatures are Base64-encoded
- Good performance and compatibility

**Key Features:**
- Private key encryption required for signing
- Public key verification
- Works with PEM-encoded key files
- Support for password-protected private keys

**Use Cases:**
- General-purpose email signing
- Systems without OpenPGP or S/MIME infrastructure
- Applications requiring simple key management

**Example:**
```go
config := &MailSignatureConfig{
    Method:         SigningMethodRSA,
    PrivateKeyFile: "~/.ssh/email.key",
    PublicKeyFile:  "~/.ssh/email.pub",
    KeyPassphrase:  "my-secret-password",
}
err := mail.SignMail(config)
```

### 2. ECDSA (Best for Performance)

**Characteristics:**
- Uses Elliptic Curve Cryptography (ECC)
- Smaller key sizes than RSA with equivalent strength
- Excellent performance characteristics
- Modern cryptographic approach

**Key Features:**
- P-256 or P-384 elliptic curve keys
- Shorter signatures than RSA
- Faster signing and verification
- Lower computational overhead

**Use Cases:**
- Mobile and IoT applications
- High-volume signing scenarios
- Performance-critical systems
- Resource-constrained environments

**Example:**
```go
config := &MailSignatureConfig{
    Method:         SigningMethodECDSA,
    PrivateKeyFile: "~/.ssh/email_ecdsa.key",
    PublicKeyFile:  "~/.ssh/email_ecdsa.pub",
    KeyPassphrase:  "my-secret-password",
}
err := mail.SignMail(config)
```

### 3. GPG (For OpenPGP Compliance)

**Characteristics:**
- Based on OpenPGP standard (RFC 4880)
- Compatible with GnuPG and other OpenPGP tools
- Widely used in Unix/Linux communities
- Text-based key format

**Key Features:**
- Requires GPG key export or local GPG installation
- Support for multiple signatures
- Trust-based key model
- Armor-formatted keys (human-readable)

**Use Cases:**
- Linux/Unix email systems
- Organizations using GnuPG infrastructure
- Email systems requiring OpenPGP compatibility
- Developer communities

**Example:**
```go
config := &MailSignatureConfig{
    Method:         SigningMethodGPG,
    PrivateKeyFile: "~/.gnupg/private.gpg",
    PublicKeyFile:  "~/.gnupg/public.gpg",
    KeyPassphrase:  "my-secret-password",
}
err := mail.SignMail(config)
```

**Note:** GPG signing uses temporary files internally and works through the pwlib GPG infrastructure.

### 4. S/MIME (For Enterprise Email)

**Characteristics:**
- Uses X.509 certificates (similar to HTTPS)
- Industry standard for enterprise email signing
- Certificate-based trust model
- Binary signature format

**Key Features:**
- X.509 certificate infrastructure
- Detached signature verification via certificate public key
- PEM-based key/certificate handling
- Integration with Certificate Authorities (CA)

**Use Cases:**
- Corporate email systems
- Financial institutions
- Healthcare email (HIPAA compliance)
- Systems integrated with PKI infrastructure

**Example:**
```go
config := &MailSignatureConfig{
    Method:         SigningMethodSMIME,
    PrivateKeyFile: "~/.certs/email_bundle.pem", // contains CERTIFICATE + RSA PRIVATE KEY
    PublicKeyFile:  "~/.certs/email.crt",        // CERTIFICATE for verification
    KeyPassphrase:  "my-secret-password",
}
err := mail.SignMail(config)
```

**Implementation:** CMS/PKCS#7 detached signatures with RSA PKCS#1 v1.5. Both a simple detached-signature API (`SignMail`/`VerifyMailSignature`) and a standards-conformant `multipart/signed` MIME builder (`BuildSMIMEMultipartSigned`/`VerifySMIMEMultipartSigned`) are provided. `SendMail` auto-builds the `multipart/signed` payload when an S/MIME config is present.

## API Reference

### Types

#### SigningMethod
```go
type SigningMethod string

const (
    SigningMethodRSA   SigningMethod = "rsa"
    SigningMethodECDSA SigningMethod = "ecdsa"
    SigningMethodGPG   SigningMethod = "gpg"
    SigningMethodSMIME SigningMethod = "smime"
)
```

#### MailSignatureConfig
```go
type MailSignatureConfig struct {
    Method           SigningMethod  // Signing method to use
    PrivateKeyFile   string         // Path to private key file
    PublicKeyFile    string         // Path to public key or certificate
    KeyPassphrase    string         // Password for encrypted private key
    SignMessageBody  bool           // S/MIME: sign only body or full message
    CertificateChain []string       // S/MIME: optional certificate chain files
    IncludeChain     bool           // S/MIME: include cert chain in signature
}
```

### Functions

#### BuildSMIMEMultipartSigned
```go
func BuildSMIMEMultipartSigned(content string, config *MailSignatureConfig) (body string, contentType string, signature string, err error)
```
Builds a CMS/PKCS#7 detached `multipart/signed` body for S/MIME.

- `body`: MIME multipart payload
- `contentType`: value for `Content-Type` header (includes boundary)
- `signature`: Base64 signature of the PKCS#7 DER data

#### VerifySMIMEMultipartSigned
```go
func VerifySMIMEMultipartSigned(body string, contentType string) (signedContent string, valid bool, err error)
```
Parses and verifies a `multipart/signed` S/MIME body.

#### SignMailContent
```go
func SignMailContent(content string, config *MailSignatureConfig) (signature string, err error)
```
Signs raw mail content and returns a Base64-encoded signature.

**Parameters:**
- `content`: The email content to sign
- `config`: Signing configuration

**Returns:**
- `signature`: Base64-encoded signature
- `err`: Error if signing fails

#### VerifyMailSignature
```go
func VerifyMailSignature(content string, signature string, config *MailSignatureConfig) (valid bool, err error)
```
Verifies a mail signature against the original content.

**Parameters:**
- `content`: The email content that was signed
- `signature`: The Base64-encoded signature to verify
- `config`: Verification configuration (same as signing config)

**Returns:**
- `valid`: True if signature is valid
- `err`: Error if verification fails

#### IsValidSigningMethod
```go
func IsValidSigningMethod(method string) bool
```
Checks if a signing method is supported.

#### GetSupportedSigningMethods
```go
func GetSupportedSigningMethods() []string
```
Returns a list of all supported signing methods.

### MailType Methods

#### SignMail
```go
func (mt *MailType) SignMail(config *MailSignatureConfig) error
```
Signs the current mail with the specified configuration.

**Behavior:**
- Joins TextParts with newlines
- Stores signature in Signature field
- Sets IsSigned to true
- Stores configuration for later verification

#### VerifyMailSignature
```go
func (mt *MailType) VerifyMailSignature() (bool, error)
```
Verifies the signature of the current mail.

**Requirements:**
- Mail must be marked as IsSigned
- Signature field must contain signature data
- SignatureConfig must be set

**Returns:**
- True if signature is valid
- Sets SignatureVerified flag
- False if signature is invalid

#### GetSignatureMethod
```go
func (mt *MailType) GetSignatureMethod() string
```
Returns the signing method used for the current mail.

### SMTP Integration

If `MailType.SignatureConfig` is set to `SigningMethodSMIME`, `SendMail` automatically:

1. Builds a `multipart/signed` S/MIME body,
2. Uses the generated multipart `Content-Type`,
3. Stores the generated signature in `MailType.Signature`.

This keeps the existing API intact while enabling S/MIME transport format for mail clients.

## Workflow Examples

### Simple RSA Signing and Verification

```go
package main

import (
    "fmt"
    "github.com/tommi2day/gomodules/maillib"
    "github.com/tommi2day/gomodules/pwlib"
)

func main() {
    // Generate RSA key pair
    pubKey, privKey, err := pwlib.GenRsaKey("email.pub", "email.key", "password")
    if err != nil {
        panic(err)
    }

    // Create mail
    mail := maillib.NewMail("sender@example.com", "recipient@example.com")
    mail.Subject = "Important Message"
    mail.TextParts = []string{"This is an important signed message."}

    // Sign mail
    config := &maillib.MailSignatureConfig{
        Method:         maillib.SigningMethodRSA,
        PrivateKeyFile: "email.key",
        PublicKeyFile:  "email.pub",
        KeyPassphrase:  "password",
    }

    err = mail.SignMail(config)
    if err != nil {
        panic(err)
    }

    // ... send mail ...

    // On receiving end: verify signature
    valid, err := mail.VerifyMailSignature()
    if err != nil {
        panic(err)
    }

    if valid {
        fmt.Println("✓ Signature verified - mail is authentic")
    } else {
        fmt.Println("✗ Signature invalid - mail may be tampered")
    }
}
```

### Multi-Method Support

```go
func signWithDifferentMethods(mail *maillib.MailType) error {
    methods := []maillib.SigningMethod{
        maillib.SigningMethodRSA,
        maillib.SigningMethodECDSA,
        maillib.SigningMethodGPG,
    }

    for _, method := range methods {
        config := &maillib.MailSignatureConfig{
            Method: method,
            // ... configure keys for each method ...
        }

        if err := mail.SignMail(config); err != nil {
            return err
        }

        validSignature := mail.IsSigned
        if validSignature {
            fmt.Printf("✓ Signed with %s\n", method)
        }
    }

    return nil
}
```

## Key Management Best Practices

1. **Key Storage:**
   - Keep private keys in secure locations (e.g., `~/.ssh`, `~/.gnupg`)
   - Use file permissions (600 or 0600) to restrict access
   - Consider hardware security modules (HSM) for critical systems

2. **Key Passphrases:**
   - Use strong passphrases (20+ characters with mixed case, numbers, symbols)
   - Store passphrases securely (never hardcode in source)
   - Use environment variables or secure credential wallets

3. **Key Rotation:**
   - Rotate keys periodically (annually or per security policy)
   - Maintain key history for verification of old signatures
   - Plan key transitions carefully

4. **Certificate Management (S/MIME):**
   - Use certificates from established Certificate Authorities
   - Monitor certificate expiration dates
   - Maintain certificate chains for chain of trust validation

## Error Handling

Common errors and how to handle them:

| Error | Cause | Resolution |
|-------|-------|-----------|
| `signature config is nil` | No config provided | Always provide MailSignatureConfig |
| `no content to sign` | Empty TextParts | Add content to TextParts before signing |
| `unsupported signing method` | Invalid method | Use one of the supported methods |
| `mail is not signed` | Attempting to verify unsigned mail | Check IsSigned flag first |
| `cannot read private key` | Key file not found or wrong format | Verify file path and format |
| `invalid signature` | Signature doesn't match content | Check that content wasn't modified |

## Testing

The implementation includes comprehensive tests covering happy paths, error cases, and security edge cases:

```bash
# Test RSA signing (sign/verify, cross-key failure, multi-part content)
go test -v ./maillib -run TestMailSigningRSA

# Test ECDSA signing
go test -v ./maillib -run TestMailSigningECDSA

# Test GPG signing
go test -v ./maillib -run TestMailSigningGPG

# Test S/MIME (sign/verify, tampered content, multipart/signed builder)
go test -v ./maillib -run TestMailSigningSMIME

# Test error cases (nil config, no content, unsigned mail, missing config)
go test -v ./maillib -run TestMailSigningErrors

# Test cross-key and invalid-signature rejection
go test -v ./maillib -run TestMailSigningCrossKey

# Test S/MIME multipart error paths (bad content-type, wrong protocol, invalid base64, etc.)
go test -v ./maillib -run TestSMIMEMultipartErrors

# Test S/MIME certificate chain, cert mismatch, and cert-only bundle
go test -v ./maillib -run TestSMIMECertChain

# Run all signing tests
go test -v ./maillib -run "TestMailSigning|TestSMIME|TestSign"
```

## Security Model

### S/MIME Certificate Validation

When `PublicKeyFile` is set, S/MIME verification uses a two-step approach:

1. `pkcs7.Verify()` checks the cryptographic signature and resolves the signer certificate by its signerInfo issuer+serial number.
2. `pkcs7.GetOnlySigner()` retrieves that resolved signer certificate — not just any certificate present in the attacker-controlled certificate bag. The signer cert is then validated against the expected certificate as a sole trust anchor via `x509.Certificate.Verify`, checking both chain-of-trust and expiry.

Simply embedding the expected certificate in the PKCS#7 certificate bag is **not** sufficient to pass verification; the certificate that produced the signature must chain to the trust anchor.

When no `PublicKeyFile` is set (standalone `VerifySMIMEMultipartSigned`), only the cryptographic signature and certificate validity period are checked. Full chain validation requires a trust anchor from the caller.

### TLS Transport Security

`EnableSSL(insecure bool)` and `EnableTLS(insecure bool)` control certificate verification for SMTP/IMAP connections. Pass `false` to enforce certificate validation (production); pass `true` only for development/test environments with self-signed certificates.

## Future Improvements

1. **Key Agility:** Support for additional algorithms (EdDSA, RSA-PSS)
2. **PKCS#11 Support:** Hardware security module integration
3. **Performance Optimization:** Caching and parallel signing

## References

- [RFC 3852 - Cryptographic Message Syntax (CMS)](https://tools.ietf.org/html/rfc3852)
- [RFC 3851 - S/MIME 3.1 Message Specification](https://tools.ietf.org/html/rfc3851)
- [RFC 4880 - OpenPGP Message Format](https://tools.ietf.org/html/rfc4880)
- [Go crypto package documentation](https://pkg.go.dev/crypto)
- [Go x509 package documentation](https://pkg.go.dev/crypto/x509)

## License

See main repository license file.



