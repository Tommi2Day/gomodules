package maillib

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"mime"
	"mime/multipart"
	"net/textproto"
	"os"
	"strings"
	"time"

	"github.com/fullsailor/pkcs7"
	"github.com/tommi2day/gomodules/common"
)

const (
	smimeProtocolPKCS7 = "application/pkcs7-signature"
	smimeMicalgSHA256  = "sha-256"
)

// BuildSMIMEMultipartSigned builds a multipart/signed body and returns body, content-type and signature.
// The signature is detached CMS/PKCS#7 (DER encoded, Base64 transport representation).
func BuildSMIMEMultipartSigned(content string, config *MailSignatureConfig) (body string, contentType string, signature string, err error) {
	if config == nil {
		return "", "", "", fmt.Errorf("signature config is nil")
	}
	if config.Method != SigningMethodSMIME {
		return "", "", "", fmt.Errorf("signing method must be smime")
	}

	privKey, cert, err := loadSMIMEPrivateKeyAndCert(config.PrivateKeyFile, config.KeyPassphrase)
	if err != nil {
		return "", "", "", err
	}

	chain, err := loadSMIMECertChain(config.CertificateChain)
	if err != nil {
		return "", "", "", err
	}

	signedPartBytes := buildSMIMESignedTextPart(content)
	sigDER, err := createSMIMEPKCS7DetachedSignature(signedPartBytes, cert, privKey, chain, config.IncludeChain)
	if err != nil {
		return "", "", "", err
	}

	sigB64 := base64.StdEncoding.EncodeToString(sigDER)
	sigWrapped := wrapBase64(sigB64, 76)

	boundary := "smime-" + common.RandString(24)
	var b strings.Builder
	b.WriteString("--" + boundary + "\r\n")
	b.Write(signedPartBytes)
	if !strings.HasSuffix(b.String(), "\r\n") {
		b.WriteString("\r\n")
	}
	b.WriteString("--" + boundary + "\r\n")
	b.WriteString("Content-Type: application/pkcs7-signature; name=\"smime.p7s\"\r\n")
	b.WriteString("Content-Transfer-Encoding: base64\r\n")
	b.WriteString("Content-Disposition: attachment; filename=\"smime.p7s\"\r\n\r\n")
	b.WriteString(sigWrapped)
	if !strings.HasSuffix(sigWrapped, "\r\n") {
		b.WriteString("\r\n")
	}
	b.WriteString("--" + boundary + "--\r\n")

	ct := fmt.Sprintf("multipart/signed; protocol=\"%s\"; micalg=%s; boundary=\"%s\"", smimeProtocolPKCS7, smimeMicalgSHA256, boundary)
	return b.String(), ct, sigB64, nil
}

// VerifySMIMEMultipartSigned verifies a multipart/signed S/MIME body and returns the signed text part and verification result.
func VerifySMIMEMultipartSigned(body string, contentType string) (signedContent string, valid bool, err error) {
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return "", false, fmt.Errorf("invalid content-type: %w", err)
	}
	if strings.ToLower(mediaType) != "multipart/signed" {
		return "", false, fmt.Errorf("content-type is not multipart/signed")
	}
	protocol := strings.Trim(params["protocol"], "\"")
	if strings.ToLower(protocol) != smimeProtocolPKCS7 {
		return "", false, fmt.Errorf("unsupported multipart/signed protocol: %s", protocol)
	}
	boundary := params["boundary"]
	if boundary == "" {
		return "", false, fmt.Errorf("missing multipart boundary")
	}

	mr := multipart.NewReader(strings.NewReader(body), boundary)
	firstPart, err := mr.NextPart()
	if err != nil {
		return "", false, fmt.Errorf("failed to read signed part: %w", err)
	}
	firstHeaders := firstPart.Header
	firstBodyBytes, err := readAllPart(firstPart)
	if err != nil {
		return "", false, fmt.Errorf("failed to read signed content: %w", err)
	}

	sigPart, err := mr.NextPart()
	if err != nil {
		return "", false, fmt.Errorf("failed to read signature part: %w", err)
	}
	sigBodyBytes, err := readAllPart(sigPart)
	if err != nil {
		return "", false, fmt.Errorf("failed to read signature content: %w", err)
	}
	sigB64 := stripWhitespace(string(sigBodyBytes))
	sigDER, err := base64.StdEncoding.DecodeString(sigB64)
	if err != nil {
		return "", false, fmt.Errorf("invalid base64 signature: %w", err)
	}

	signedEntity := rebuildSignedPartForVerification(firstHeaders, firstBodyBytes)
	valid, err = verifySMIMEPKCS7DetachedSignature(signedEntity, sigDER)
	if err == nil {
		return string(firstBodyBytes), valid, nil
	}
	return string(firstBodyBytes), false, nil
}

func createSMIMEPKCS7DetachedSignature(data []byte, cert *x509.Certificate, privKey *rsa.PrivateKey, chain []*x509.Certificate, includeChain bool) ([]byte, error) {
	sd, err := pkcs7.NewSignedData(data)
	if err != nil {
		return nil, fmt.Errorf("create signed data failed: %w", err)
	}
	sd.Detach()
	if err := sd.AddSigner(cert, privKey, pkcs7.SignerInfoConfig{}); err != nil {
		return nil, fmt.Errorf("add signer failed: %w", err)
	}
	if includeChain {
		for _, c := range chain {
			sd.AddCertificate(c)
		}
	}
	out, err := sd.Finish()
	if err != nil {
		return nil, fmt.Errorf("finish signed data failed: %w", err)
	}
	return out, nil
}

func verifySMIMEPKCS7DetachedSignature(data []byte, sigDER []byte) (bool, error) {
	return verifySMIMEPKCS7DetachedSignatureWithCert(data, sigDER, nil)
}

func verifySMIMEPKCS7DetachedSignatureWithCert(data []byte, sigDER []byte, expected *x509.Certificate) (bool, error) {
	p7, err := pkcs7.Parse(sigDER)
	if err != nil {
		return false, fmt.Errorf("parse pkcs7 failed: %w", err)
	}
	p7.Content = data

	if expected != nil {
		// Build a single-cert trust pool so VerifyWithChain confirms the actual
		// signer cert (identified by issuer+serial inside the PKCS#7) chains to
		// the expected cert — not just that the expected cert appears anywhere in
		// the attacker-controlled certificate bag.
		trustPool := x509.NewCertPool()
		trustPool.AddCert(expected)
		if err := p7.VerifyWithChain(trustPool); err != nil {
			return false, fmt.Errorf("verify pkcs7 with chain failed: %w", err)
		}
	} else {
		// No pinned cert: verify the cryptographic signature only, then check
		// that no embedded certificate is outside its validity window.
		if err := p7.Verify(); err != nil {
			return false, fmt.Errorf("verify pkcs7 failed: %w", err)
		}
		now := time.Now()
		for _, cert := range p7.Certificates {
			if now.Before(cert.NotBefore) || now.After(cert.NotAfter) {
				return false, fmt.Errorf("signer certificate validity period check failed: %s", cert.Subject)
			}
		}
	}
	return true, nil
}

func loadSMIMECertChain(files []string) ([]*x509.Certificate, error) {
	if len(files) == 0 {
		return nil, nil
	}
	var out []*x509.Certificate
	for _, fn := range files {
		pemData, err := osReadFile(fn)
		if err != nil {
			return nil, fmt.Errorf("read certificate chain file %s failed: %w", fn, err)
		}
		certs, err := parsePEMCertificates(pemData)
		if err != nil {
			return nil, err
		}
		out = append(out, certs...)
	}
	return out, nil
}

func parsePEMCertificates(pemData []byte) ([]*x509.Certificate, error) {
	var certs []*x509.Certificate
	for len(pemData) > 0 {
		block, rest := pemDecode(pemData)
		pemData = rest
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse certificate failed: %w", err)
		}
		certs = append(certs, cert)
	}
	if len(certs) == 0 {
		return nil, fmt.Errorf("no certificate in chain file")
	}
	return certs, nil
}

func buildSMIMESignedTextPart(content string) []byte {
	canonical := normalizeCRLF(content)
	if !strings.HasSuffix(canonical, "\r\n") {
		canonical += "\r\n"
	}
	var b strings.Builder
	b.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	b.WriteString("Content-Transfer-Encoding: 7bit\r\n\r\n")
	b.WriteString(canonical)
	return []byte(b.String())
}

func rebuildSignedPartForVerification(h textproto.MIMEHeader, body []byte) []byte {
	ct := firstHeaderValue(h, "Content-Type", "text/plain; charset=utf-8")
	cte := firstHeaderValue(h, "Content-Transfer-Encoding", "7bit")
	canonicalBody := normalizeCRLF(string(body))
	if !strings.HasSuffix(canonicalBody, "\r\n") {
		canonicalBody += "\r\n"
	}
	var b strings.Builder
	b.WriteString("Content-Type: ")
	b.WriteString(ct)
	b.WriteString("\r\n")
	b.WriteString("Content-Transfer-Encoding: ")
	b.WriteString(cte)
	b.WriteString("\r\n\r\n")
	b.WriteString(canonicalBody)
	return []byte(b.String())
}

func normalizeCRLF(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.ReplaceAll(s, "\n", "\r\n")
}

func wrapBase64(s string, lineLen int) string {
	if lineLen <= 0 || len(s) <= lineLen {
		return s
	}
	var b strings.Builder
	for len(s) > lineLen {
		b.WriteString(s[:lineLen])
		b.WriteString("\r\n")
		s = s[lineLen:]
	}
	if len(s) > 0 {
		b.WriteString(s)
	}
	return b.String()
}

func stripWhitespace(s string) string {
	r := strings.NewReplacer("\r", "", "\n", "", "\t", "", " ", "")
	return r.Replace(s)
}

func readAllPart(p *multipart.Part) ([]byte, error) {
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(p); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func firstHeaderValue(h textproto.MIMEHeader, key string, fallback string) string {
	v := h.Get(key)
	if v == "" {
		return fallback
	}
	return v
}

// Thin wrappers to keep testability without stubbing os/encoding packages directly.
var (
	osReadFile = os.ReadFile
	pemDecode  = pem.Decode
)
