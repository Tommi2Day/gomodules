package pwlib

import (
	"bufio"
	"bytes"
	"crypto/rsa"
	"crypto/sha1" //nolint:gosec // SHA-1 is required by GnuPG keygrip specification
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	log "github.com/sirupsen/logrus"
)

// GPGAgentSocket returns the path to the gpg-agent Assuan socket.
// Checks (in order): $GPG_AGENT_INFO, $GNUPGHOME/S.gpg-agent, $XDG_RUNTIME_DIR/gnupg/S.gpg-agent.
func GPGAgentSocket() (string, error) {
	if info := os.Getenv("GPG_AGENT_INFO"); info != "" { //nolint:gosec // env-var name, not a credential
		if parts := strings.SplitN(info, ":", 2); parts[0] != "" {
			return parts[0], nil
		}
	}
	if gpgHome, err := GPGHomeDir(); err == nil {
		if sock := filepath.Join(gpgHome, "S.gpg-agent"); gpgFileExists(sock) {
			return sock, nil
		}
	}
	if xdg := os.Getenv("XDG_RUNTIME_DIR"); xdg != "" { //nolint:gosec // env-var name, not a credential
		if sock := filepath.Join(xdg, "gnupg", "S.gpg-agent"); gpgFileExists(sock) {
			return sock, nil
		}
	}
	return "", fmt.Errorf("gpg-agent socket not found (set GNUPGHOME or XDG_RUNTIME_DIR)")
}

func gpgFileExists(path string) bool {
	_, err := os.Stat(path) //nolint:gosec // path comes from trusted env-var or config
	return err == nil
}

// gpgAgentConnect opens a raw connection to the gpg-agent socket.
// On Unix this is a Unix domain socket. On Windows gpg-agent writes a file
// containing a TCP port number and 16-byte nonce; we connect via TCP and send
// the nonce first (GnuPG "Assuan socket emulation").
func gpgAgentConnect(socketPath string) (net.Conn, error) {
	if runtime.GOOS != "windows" {
		return net.Dial("unix", socketPath)
	}
	// Windows: socket file is "<decimal-port>\n<16-raw-nonce-bytes>"
	data, err := os.ReadFile(socketPath) //nolint:gosec // path comes from GPGAgentSocket
	if err != nil {
		return nil, fmt.Errorf("read gpg-agent socket file: %w", err)
	}
	nl := bytes.IndexByte(data, '\n')
	if nl < 0 {
		return nil, fmt.Errorf("gpg-agent socket file: missing newline")
	}
	port, err := strconv.Atoi(strings.TrimSpace(string(data[:nl])))
	if err != nil {
		return nil, fmt.Errorf("gpg-agent socket file port: %w", err)
	}
	nonce := data[nl+1:]
	if len(nonce) < 16 {
		return nil, fmt.Errorf("gpg-agent socket file: nonce too short (%d bytes)", len(nonce))
	}
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return nil, err
	}
	if _, err = conn.Write(nonce[:16]); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("send gpg-agent nonce: %w", err)
	}
	return conn, nil
}

// gpgKeygrip computes the GnuPG keygrip for a public key.
// For RSA the keygrip is SHA-1 of the raw big-endian modulus bytes, which is
// what libgcrypt's compute_keygrip produces. Other algorithms are not supported.
func gpgKeygrip(pub *packet.PublicKey) ([]byte, error) {
	switch pub.PubKeyAlgo { //nolint:exhaustive // only RSA is supported; default handles all others
	case packet.PubKeyAlgoRSA, packet.PubKeyAlgoRSAEncryptOnly, packet.PubKeyAlgoRSASignOnly:
		rsaKey, ok := pub.PublicKey.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("gpgKeygrip: not an RSA public key")
		}
		h := sha1.New() //nolint:gosec // required by GnuPG keygrip specification
		_, _ = h.Write(rsaKey.N.Bytes())
		return h.Sum(nil), nil
	default:
		return nil, fmt.Errorf("gpgKeygrip: algorithm %d not supported (only RSA)", pub.PubKeyAlgo)
	}
}

// assuanClient is a minimal Assuan protocol client for gpg-agent.
type assuanClient struct {
	conn   net.Conn
	reader *bufio.Reader
}

func newAssuanClient(socketPath string) (*assuanClient, error) {
	conn, err := gpgAgentConnect(socketPath)
	if err != nil {
		return nil, err
	}
	c := &assuanClient{conn: conn, reader: bufio.NewReader(conn)}
	line, err := c.readLine()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("gpg-agent greeting: %w", err)
	}
	if !strings.HasPrefix(line, "OK") {
		_ = conn.Close()
		return nil, fmt.Errorf("gpg-agent unexpected greeting: %q", line)
	}
	log.Debugf("gpg-agent connected: %s", line)
	return c, nil
}

func (c *assuanClient) Close() { _ = c.conn.Close() }

func (c *assuanClient) readLine() (string, error) {
	line, err := c.reader.ReadString('\n')
	return strings.TrimRight(line, "\r\n"), err
}

func (c *assuanClient) sendLine(cmd string) error {
	_, err := fmt.Fprintf(c.conn, "%s\n", cmd)
	return err
}

// pkdecrypt sends the PKDECRYPT command with the given keygrip and ciphertext
// S-expression, and returns the raw decrypted value bytes from the agent response.
func (c *assuanClient) pkdecrypt(keygrip, ciphertextSexp string) ([]byte, error) {
	if err := c.sendLine(fmt.Sprintf("PKDECRYPT --keygrip=%s", keygrip)); err != nil {
		return nil, err
	}
	line, err := c.readLine()
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(line, "INQUIRE CIPHERTEXT") {
		return nil, fmt.Errorf("expected INQUIRE CIPHERTEXT, got: %q", line)
	}
	if err = c.sendLine("D " + assuanPercentEncode(ciphertextSexp)); err != nil {
		return nil, err
	}
	if err = c.sendLine("END"); err != nil {
		return nil, err
	}
	var dataBuf bytes.Buffer
	for {
		line, err = c.readLine()
		if err != nil {
			return nil, err
		}
		switch {
		case strings.HasPrefix(line, "D "):
			decoded, decErr := assuanPercentDecode(line[2:])
			if decErr != nil {
				return nil, decErr
			}
			_, _ = dataBuf.Write(decoded)
		case strings.HasPrefix(line, "OK"):
			return dataBuf.Bytes(), nil
		case strings.HasPrefix(line, "ERR"):
			return nil, fmt.Errorf("gpg-agent PKDECRYPT: %s", line)
			// S (status) and # (comment) lines are silently ignored
		}
	}
}

// assuanPercentEncode encodes a string for use in an Assuan D command.
// NUL, CR, LF, and % must be encoded; all other bytes pass through.
func assuanPercentEncode(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch == '%' || ch == '\r' || ch == '\n' || ch == 0 {
			_, _ = fmt.Fprintf(&b, "%%%02X", ch)
		} else {
			_ = b.WriteByte(ch)
		}
	}
	return b.String()
}

// assuanPercentDecode decodes an Assuan percent-encoded data line.
func assuanPercentDecode(s string) ([]byte, error) {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '%' && i+2 < len(s) {
			b, err := strconv.ParseUint(s[i+1:i+3], 16, 8)
			if err != nil {
				return nil, fmt.Errorf("assuan decode offset %d: %w", i, err)
			}
			out = append(out, byte(b))
			i += 2
		} else {
			out = append(out, s[i])
		}
	}
	return out, nil
}

// gpgSkipPacketHeader returns the packet body by skipping the OpenPGP
// new-format packet framing (tag byte + variable-length length encoding).
func gpgSkipPacketHeader(data []byte) ([]byte, error) {
	if len(data) < 2 {
		return nil, fmt.Errorf("packet too short")
	}
	b1 := data[1] // first length byte (data[0] is the tag)
	switch {
	case b1 < 192:
		return data[2:], nil // 1-byte length
	case b1 < 224:
		if len(data) < 3 {
			return nil, fmt.Errorf("packet header truncated (2-byte length)")
		}
		return data[3:], nil // 2-byte length
	case b1 == 255:
		if len(data) < 6 {
			return nil, fmt.Errorf("packet header truncated (5-byte length)")
		}
		return data[6:], nil // 5-byte length
	default:
		return nil, fmt.Errorf("partial-body length not supported in PKESK")
	}
}

// gpgPKESKtoSexp serializes ek and re-parses the body to produce the
// S-expression string required by gpg-agent's PKDECRYPT INQUIRE CIPHERTEXT.
// Currently supports RSA (algorithms 1 and 2) only.
func gpgPKESKtoSexp(ek *packet.EncryptedKey) (string, error) {
	var buf bytes.Buffer
	if err := ek.Serialize(&buf); err != nil {
		return "", fmt.Errorf("serialize PKESK: %w", err)
	}
	body, err := gpgSkipPacketHeader(buf.Bytes())
	if err != nil {
		return "", err
	}
	// v3 body layout: version(1) + keyID(8) + algo(1) = 10-byte prefix
	if len(body) < 11 {
		return "", fmt.Errorf("PKESK body too short (%d bytes)", len(body))
	}
	if body[0] != 3 {
		return "", fmt.Errorf("PKESK version %d not supported (only v3)", body[0])
	}
	payload := body[10:] // algorithm-specific encrypted key material

	switch ek.Algo { //nolint:exhaustive // only RSA is supported; default handles all others
	case packet.PubKeyAlgoRSA, packet.PubKeyAlgoRSAEncryptOnly:
		if len(payload) < 3 {
			return "", fmt.Errorf("RSA PKESK payload too short")
		}
		bitLen := int(payload[0])<<8 | int(payload[1])
		byteLen := (bitLen + 7) / 8
		if 2+byteLen > len(payload) {
			return "", fmt.Errorf("RSA PKESK payload truncated")
		}
		return fmt.Sprintf("(enc-val (flags pkcs1) (rsa (a #%X#)))", payload[2:2+byteLen]), nil
	default:
		return "", fmt.Errorf("PKDECRYPT not implemented for algorithm %d; set GPG_PASSPHRASE instead", ek.Algo)
	}
}

// gpgParseAgentSessionKey parses the (value #...#) S-expression returned by
// gpg-agent PKDECRYPT for a v3 RSA/ECDH PKESK. The encoded format is:
// [1-byte cipher algo][session key bytes][2-byte checksum].
func gpgParseAgentSessionKey(resp []byte) (packet.CipherFunction, []byte, error) {
	start := bytes.IndexByte(resp, '#')
	end := bytes.LastIndexByte(resp, '#')
	if start < 0 || end <= start {
		return 0, nil, fmt.Errorf("gpg-agent response missing hex block: %q", resp)
	}
	decoded, err := hex.DecodeString(string(resp[start+1 : end]))
	if err != nil {
		return 0, nil, fmt.Errorf("gpg-agent session key hex: %w", err)
	}
	if len(decoded) < 4 {
		return 0, nil, fmt.Errorf("gpg-agent session key too short (%d bytes)", len(decoded))
	}
	cipherFunc := packet.CipherFunction(decoded[0])
	keyAndSum := decoded[1:]
	sessionKey := keyAndSum[:len(keyAndSum)-2]
	var sum uint16
	for _, b := range sessionKey {
		sum += uint16(b)
	}
	wantSum := uint16(keyAndSum[len(keyAndSum)-2])<<8 | uint16(keyAndSum[len(keyAndSum)-1])
	if sum != wantSum {
		return 0, nil, fmt.Errorf("gpg-agent session key checksum mismatch")
	}
	return cipherFunc, sessionKey, nil
}

// gpgReadLiteralData reads all LiteralData (and Compressed) packets from r
// and returns their concatenated plaintext.
func gpgReadLiteralData(r io.Reader) (string, error) {
	pr := packet.NewReader(r)
	var out strings.Builder
	for {
		p, err := pr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			// Non-EOF errors here are expected at the encrypted data boundary.
			return out.String(), nil //nolint:nilerr // boundary error, not a real failure
		}
		switch pt := p.(type) {
		case *packet.LiteralData:
			data, readErr := io.ReadAll(pt.Body)
			if readErr != nil {
				return "", fmt.Errorf("read literal data: %w", readErr)
			}
			_, _ = out.Write(data)
		case *packet.Compressed:
			nested, readErr := gpgReadLiteralData(pt.Body)
			if readErr != nil {
				return "", readErr
			}
			_, _ = out.WriteString(nested)
		}
	}
	return out.String(), nil
}

// gpgDecryptSEIPD re-parses fileData to find the SEIPD packet, decrypts it
// with the given cipher function and session key, and returns the plaintext.
func gpgDecryptSEIPD(fileData []byte, cipherFunc packet.CipherFunction, sessionKey []byte) (string, error) {
	pr := packet.NewReader(bytes.NewReader(fileData))
	for {
		p, err := pr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
		se, ok := p.(*packet.SymmetricallyEncrypted)
		if !ok {
			continue
		}
		plainRC, decErr := se.Decrypt(cipherFunc, sessionKey)
		if decErr != nil {
			return "", fmt.Errorf("SEIPD decrypt: %w", decErr)
		}
		return gpgReadLiteralData(plainRC)
	}
	return "", fmt.Errorf("no SEIPD packet found")
}

// gpgBuildKeyByID builds a key-ID → public-key map from an EntityList.
func gpgBuildKeyByID(entityList openpgp.EntityList) map[uint64]*packet.PublicKey {
	m := make(map[uint64]*packet.PublicKey)
	for _, e := range entityList {
		if e.PrimaryKey != nil {
			m[e.PrimaryKey.KeyId] = e.PrimaryKey
		}
		for _, sk := range e.Subkeys {
			if sk.PublicKey != nil {
				m[sk.PublicKey.KeyId] = sk.PublicKey
			}
		}
	}
	return m
}

// gpgCollectPKESK reads all PKESK packets from an already-read file buffer.
func gpgCollectPKESK(fileData []byte) []*packet.EncryptedKey {
	var pksks []*packet.EncryptedKey
	pr := packet.NewReader(bytes.NewReader(fileData))
	for {
		p, err := pr.Next()
		if err != nil {
			break
		}
		if ek, ok := p.(*packet.EncryptedKey); ok {
			pksks = append(pksks, ek)
		}
	}
	return pksks
}

// GPGAgentDecrypt decrypts filename using gpg-agent via the Assuan PKDECRYPT
// command. It matches PKESK key IDs against entityList, requests the agent to
// decrypt the session key (the agent supplies the passphrase from its cache),
// and decrypts the SEIPD packet using the returned session key. RSA keys only;
// other algorithms return an error suggesting GPG_PASSPHRASE.
func GPGAgentDecrypt(filename string, entityList openpgp.EntityList) (string, error) {
	socketPath, err := GPGAgentSocket()
	if err != nil {
		return "", fmt.Errorf("GPGAgentDecrypt: %w", err)
	}
	fileData, err := os.ReadFile(filename) //nolint:gosec // path comes from caller
	if err != nil {
		return "", fmt.Errorf("read %s: %w", filename, err)
	}
	pksks := gpgCollectPKESK(fileData)
	if len(pksks) == 0 {
		return "", fmt.Errorf("GPGAgentDecrypt: no PKESK packets in %s", filename)
	}
	keyByID := gpgBuildKeyByID(entityList)
	client, err := newAssuanClient(socketPath)
	if err != nil {
		return "", err
	}
	defer client.Close()

	for _, ek := range pksks {
		plain, tryErr := gpgAgentTryKey(client, ek, keyByID, fileData)
		if tryErr != nil {
			log.Debugf("GPGAgentDecrypt: key %016X: %v", ek.KeyId, tryErr)
			continue
		}
		log.Debugf("GPGAgentDecrypt: decrypted %s via gpg-agent (key %016X)", filename, ek.KeyId)
		return plain, nil
	}
	return "", fmt.Errorf("GPGAgentDecrypt: no matching RSA key found in gpg-agent for %s (only RSA PKESK supported; for other algorithms set GPG_PASSPHRASE)", filename)
}

// gpgAgentTryKey attempts to decrypt fileData using gpg-agent for one PKESK entry.
func gpgAgentTryKey(client *assuanClient, ek *packet.EncryptedKey, keyByID map[uint64]*packet.PublicKey, fileData []byte) (string, error) {
	pub, ok := keyByID[ek.KeyId]
	if !ok {
		return "", fmt.Errorf("key %016X not in entity list", ek.KeyId)
	}
	keygrip, err := gpgKeygrip(pub)
	if err != nil {
		return "", err
	}
	sexp, err := gpgPKESKtoSexp(ek)
	if err != nil {
		return "", err
	}
	kgHex := strings.ToUpper(hex.EncodeToString(keygrip))
	respBytes, err := client.pkdecrypt(kgHex, sexp)
	if err != nil {
		return "", err
	}
	cipherFunc, sessionKey, err := gpgParseAgentSessionKey(respBytes)
	if err != nil {
		return "", err
	}
	return gpgDecryptSEIPD(fileData, cipherFunc, sessionKey)
}
