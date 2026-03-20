package pwlib

import (
	"bufio"
	"bytes"
	"crypto/rsa"
	"crypto/sha1" //nolint:gosec // used to verify gpgKeygrip which requires SHA-1
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tommi2day/gomodules/common"
	"github.com/tommi2day/gomodules/test"
)

// --------------------------------------------------------------------------
// GPGAgentSocket
// --------------------------------------------------------------------------

func TestGPGAgentSocket(t *testing.T) {
	test.InitTestDirs()

	t.Run("GPG_AGENT_INFO takes priority", func(t *testing.T) {
		t.Setenv("GPG_AGENT_INFO", "/tmp/agent.sock:12345:1")
		sock, err := GPGAgentSocket()
		require.NoError(t, err)
		assert.Equal(t, "/tmp/agent.sock", sock)
	})

	t.Run("GPG_AGENT_INFO with empty socket part is ignored", func(t *testing.T) {
		t.Setenv("GPG_AGENT_INFO", ":12345:1")
		// Clear all fallback env vars so no real agent socket can be found.
		t.Setenv(gpgEnvHome, filepath.Join(test.TestData, "gpg-agent-no-socket-info"))
		t.Setenv("XDG_RUNTIME_DIR", "")
		require.NoError(t, os.MkdirAll(filepath.Join(test.TestData, "gpg-agent-no-socket-info"), 0700))
		_, err := GPGAgentSocket()
		assert.Error(t, err)
	})

	t.Run("GNUPGHOME/S.gpg-agent is used when file exists", func(t *testing.T) {
		// Clear GPG_AGENT_INFO so it doesn't take priority over GNUPGHOME.
		t.Setenv("GPG_AGENT_INFO", "")
		home := filepath.Join(test.TestData, "gpg-agent-socket-test")
		require.NoError(t, os.MkdirAll(home, 0700))
		sockPath := filepath.Join(home, "S.gpg-agent")
		require.NoError(t, os.WriteFile(sockPath, []byte("fake"), 0600))
		defer func() { _ = os.Remove(sockPath) }()

		t.Setenv(gpgEnvHome, home)
		sock, err := GPGAgentSocket()
		require.NoError(t, err)
		assert.Equal(t, sockPath, sock)
	})

	t.Run("returns error when no socket found", func(t *testing.T) {
		emptyHome := filepath.Join(test.TestData, "gpg-agent-no-socket")
		require.NoError(t, os.MkdirAll(emptyHome, 0700))
		t.Setenv("GPG_AGENT_INFO", "")
		t.Setenv(gpgEnvHome, emptyHome)
		t.Setenv("XDG_RUNTIME_DIR", "")
		_, err := GPGAgentSocket()
		assert.Error(t, err)
	})
}

// --------------------------------------------------------------------------
// gpgKeygrip
// --------------------------------------------------------------------------

func TestGPGKeygrip(t *testing.T) {
	test.InitTestDirs()

	entity, _, err := CreateGPGEntity(testGPGName, "KeygripTest", testGPGEmail, testGPGPass)
	require.NoError(t, err)

	t.Run("RSA primary key keygrip equals SHA-1 of modulus", func(t *testing.T) {
		rsaKey, ok := entity.PrimaryKey.PublicKey.(*rsa.PublicKey)
		require.True(t, ok, "expected RSA public key")

		h := sha1.New() //nolint:gosec // verifying gpgKeygrip which requires SHA-1
		_, _ = h.Write(rsaKey.N.Bytes())
		expected := h.Sum(nil)

		grip, gripErr := gpgKeygrip(entity.PrimaryKey)
		require.NoError(t, gripErr)
		assert.Equal(t, expected, grip)
		assert.Len(t, grip, 20)
	})

	t.Run("keygrip is stable across calls", func(t *testing.T) {
		grip1, err1 := gpgKeygrip(entity.PrimaryKey)
		grip2, err2 := gpgKeygrip(entity.PrimaryKey)
		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.Equal(t, grip1, grip2)
	})

	t.Run("unsupported algorithm returns error", func(t *testing.T) {
		fakePub := &packet.PublicKey{PubKeyAlgo: packet.PubKeyAlgoECDH}
		_, err := gpgKeygrip(fakePub)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not supported")
	})
}

// --------------------------------------------------------------------------
// assuanPercentEncode / assuanPercentDecode
// --------------------------------------------------------------------------

func TestAssuanPercentEncode(t *testing.T) {
	t.Run("plain ASCII passes through unchanged", func(t *testing.T) {
		assert.Equal(t, "(enc-val (rsa))", assuanPercentEncode("(enc-val (rsa))"))
	})

	t.Run("percent sign is encoded", func(t *testing.T) {
		assert.Equal(t, "50%25", assuanPercentEncode("50%"))
	})

	t.Run("CR, LF, and NUL are encoded", func(t *testing.T) {
		encoded := assuanPercentEncode("a\r\nb\x00c")
		assert.Equal(t, "a%0D%0Ab%00c", encoded)
	})
}

func TestAssuanPercentDecode(t *testing.T) {
	t.Run("encoded percent sign round-trips", func(t *testing.T) {
		out, err := assuanPercentDecode("50%25")
		require.NoError(t, err)
		assert.Equal(t, []byte("50%"), out)
	})

	t.Run("plain bytes pass through", func(t *testing.T) {
		out, err := assuanPercentDecode("hello")
		require.NoError(t, err)
		assert.Equal(t, []byte("hello"), out)
	})

	t.Run("invalid hex escape returns error", func(t *testing.T) {
		_, err := assuanPercentDecode("%ZZ")
		assert.Error(t, err)
	})

	t.Run("encode then decode round-trip", func(t *testing.T) {
		original := "(enc-val\n (rsa (a #AB%CD#)))"
		decoded, err := assuanPercentDecode(assuanPercentEncode(original))
		require.NoError(t, err)
		assert.Equal(t, []byte(original), decoded)
	})
}

// --------------------------------------------------------------------------
// gpgSkipPacketHeader
// --------------------------------------------------------------------------

func TestGPGSkipPacketHeader(t *testing.T) {
	t.Run("1-byte length: body starts at offset 2", func(t *testing.T) {
		// tag=0xC1, length=0x0A (10), body = 10 "x" bytes
		data := append([]byte{0xC1, 0x0A}, bytes.Repeat([]byte("x"), 10)...)
		body, err := gpgSkipPacketHeader(data)
		require.NoError(t, err)
		assert.Equal(t, bytes.Repeat([]byte("x"), 10), body)
	})

	t.Run("2-byte length: body starts at offset 3", func(t *testing.T) {
		// first length byte = 0xC0 (192) → 2-byte length
		data := []byte{0xC1, 0xC0, 0x00, 0x01, 0x02}
		body, err := gpgSkipPacketHeader(data)
		require.NoError(t, err)
		assert.Equal(t, []byte{0x01, 0x02}, body)
	})

	t.Run("5-byte length: body starts at offset 6", func(t *testing.T) {
		// first length byte = 0xFF → 5-byte length
		data := []byte{0xC1, 0xFF, 0x00, 0x00, 0x00, 0x03, 0xAA, 0xBB, 0xCC}
		body, err := gpgSkipPacketHeader(data)
		require.NoError(t, err)
		assert.Equal(t, []byte{0xAA, 0xBB, 0xCC}, body)
	})

	t.Run("packet too short returns error", func(t *testing.T) {
		_, err := gpgSkipPacketHeader([]byte{0xC1})
		assert.Error(t, err)
	})

	t.Run("partial-body length returns error", func(t *testing.T) {
		// first length byte in range 224-254 → partial body
		data := []byte{0xC1, 0xE0, 0x01}
		_, err := gpgSkipPacketHeader(data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "partial-body")
	})
}

// --------------------------------------------------------------------------
// gpgPKESKtoSexp
// --------------------------------------------------------------------------

func TestGPGPKESKtoSexp(t *testing.T) {
	test.InitTestDirs()

	entity, _, err := CreateGPGEntity(testGPGName, "PKESKSexpTest", testGPGEmail, testGPGPass)
	require.NoError(t, err)

	pubFile := filepath.Join(test.TestData, "pkesk_sexp"+pubGPGExt)
	privFile := filepath.Join(test.TestData, "pkesk_sexp"+privGPGExt)
	require.NoError(t, ExportGPGKeyPair(entity, pubFile, privFile))

	plaintextFile := filepath.Join(test.TestData, "pkesk_sexp.txt")
	cryptedFile := filepath.Join(test.TestData, "pkesk_sexp.gpg")
	require.NoError(t, common.WriteStringToFile(plaintextFile, "sexp-test-content"))
	require.NoError(t, GPGEncryptFile(plaintextFile, cryptedFile, pubFile))

	fileData, readErr := os.ReadFile(cryptedFile)
	require.NoError(t, readErr)
	pksks := gpgCollectPKESK(fileData)
	require.NotEmpty(t, pksks, "should find at least one PKESK in encrypted file")

	t.Run("RSA PKESK produces valid S-expression", func(t *testing.T) {
		sexp, sexpErr := gpgPKESKtoSexp(pksks[0])
		require.NoError(t, sexpErr)
		assert.True(t, strings.HasPrefix(sexp, "(enc-val (flags pkcs1) (rsa (a #"),
			"unexpected sexp prefix: %s", sexp)
		assert.True(t, strings.HasSuffix(sexp, "#)))"),
			"unexpected sexp suffix: %s", sexp)
	})

	t.Run("S-expression hex section is non-empty", func(t *testing.T) {
		sexp, sexpErr := gpgPKESKtoSexp(pksks[0])
		require.NoError(t, sexpErr)
		start := strings.Index(sexp, "#")
		end := strings.LastIndex(sexp, "#")
		require.True(t, end > start+1, "hex section should be non-empty")
		hexPart := sexp[start+1 : end]
		_, hexErr := hex.DecodeString(hexPart)
		assert.NoError(t, hexErr, "sexp hex section should be valid hex")
	})
}

// --------------------------------------------------------------------------
// gpgParseAgentSessionKey
// --------------------------------------------------------------------------

func TestGPGParseAgentSessionKey(t *testing.T) {
	makeResponse := func(cipherByte byte, key []byte) []byte {
		var sum uint16
		for _, b := range key {
			sum += uint16(b)
		}
		payload := append([]byte{cipherByte}, key...)
		payload = append(payload, byte(sum>>8), byte(sum))
		return []byte(fmt.Sprintf("(value #%s#)", strings.ToUpper(hex.EncodeToString(payload))))
	}

	t.Run("valid AES-256 session key is parsed correctly", func(t *testing.T) {
		key := bytes.Repeat([]byte{0x42}, 32) // 32-byte AES-256 key
		resp := makeResponse(9, key)          // cipher 9 = AES-256

		cf, sk, err := gpgParseAgentSessionKey(resp)
		require.NoError(t, err)
		assert.Equal(t, packet.CipherFunction(9), cf)
		assert.Equal(t, key, sk)
	})

	t.Run("checksum mismatch returns error", func(t *testing.T) {
		key := []byte{0x01, 0x02, 0x03, 0x04}
		// build with correct checksum then corrupt it
		var sum uint16
		for _, b := range key {
			sum += uint16(b)
		}
		payload := append([]byte{0x07}, key...)
		payload = append(payload, byte(sum>>8), byte((sum^0xFF)&0xFF)) // corrupt low byte
		resp := []byte(fmt.Sprintf("(value #%s#)", hex.EncodeToString(payload)))

		_, _, err := gpgParseAgentSessionKey(resp)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "checksum")
	})

	t.Run("response without hex block returns error", func(t *testing.T) {
		_, _, err := gpgParseAgentSessionKey([]byte("(value notvalid)"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "hex block")
	})

	t.Run("payload too short returns error", func(t *testing.T) {
		// only 3 bytes: cipher + 0 key bytes + 2 checksum → too short (need ≥ 4)
		resp := []byte("(value #010203#)")
		_, _, err := gpgParseAgentSessionKey(resp)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "too short")
	})
}

// --------------------------------------------------------------------------
// gpgBuildKeyByID
// --------------------------------------------------------------------------

func TestGPGBuildKeyByID(t *testing.T) {
	test.InitTestDirs()

	entity1, _, err := CreateGPGEntity(testGPGName, "BuildMapTest1", testGPGEmail, testGPGPass)
	require.NoError(t, err)
	entity2, _, err := CreateGPGEntity(testGPGName, "BuildMapTest2", testGPGEmail, testGPGPass)
	require.NoError(t, err)

	t.Run("primary key ID is in map", func(t *testing.T) {
		m := gpgBuildKeyByID(openpgp.EntityList{entity1})
		assert.Contains(t, m, entity1.PrimaryKey.KeyId)
	})

	t.Run("subkey IDs are in map", func(t *testing.T) {
		m := gpgBuildKeyByID(openpgp.EntityList{entity1})
		for _, sk := range entity1.Subkeys {
			assert.Contains(t, m, sk.PublicKey.KeyId)
		}
	})

	t.Run("multiple entities are all indexed", func(t *testing.T) {
		m := gpgBuildKeyByID(openpgp.EntityList{entity1, entity2})
		assert.Contains(t, m, entity1.PrimaryKey.KeyId)
		assert.Contains(t, m, entity2.PrimaryKey.KeyId)
	})

	t.Run("empty list returns empty map", func(t *testing.T) {
		m := gpgBuildKeyByID(openpgp.EntityList{})
		assert.Empty(t, m)
	})
}

// --------------------------------------------------------------------------
// gpgCollectPKESK
// --------------------------------------------------------------------------

func TestGPGCollectPKESK(t *testing.T) {
	test.InitTestDirs()

	entity, _, err := CreateGPGEntity(testGPGName, "CollectPKESKTest", testGPGEmail, testGPGPass)
	require.NoError(t, err)

	pubFile := filepath.Join(test.TestData, "collect_pkesk"+pubGPGExt)
	privFile := filepath.Join(test.TestData, "collect_pkesk"+privGPGExt)
	require.NoError(t, ExportGPGKeyPair(entity, pubFile, privFile))

	plaintextFile := filepath.Join(test.TestData, "collect_pkesk.txt")
	cryptedFile := filepath.Join(test.TestData, "collect_pkesk.gpg")
	require.NoError(t, common.WriteStringToFile(plaintextFile, "collect-test"))
	require.NoError(t, GPGEncryptFile(plaintextFile, cryptedFile, pubFile))

	fileData, readErr := os.ReadFile(cryptedFile)
	require.NoError(t, readErr)

	t.Run("finds PKESK in encrypted file", func(t *testing.T) {
		pksks := gpgCollectPKESK(fileData)
		assert.NotEmpty(t, pksks)
	})

	t.Run("PKESK key ID matches the encryption key", func(t *testing.T) {
		pksks := gpgCollectPKESK(fileData)
		require.NotEmpty(t, pksks)
		// The key ID in the PKESK should match a key in the entity
		keyByID := gpgBuildKeyByID(openpgp.EntityList{entity})
		found := false
		for _, ek := range pksks {
			if _, ok := keyByID[ek.KeyId]; ok {
				found = true
				break
			}
		}
		assert.True(t, found, "at least one PKESK key ID should match the encryption entity")
	})

	t.Run("empty data returns empty slice", func(t *testing.T) {
		pksks := gpgCollectPKESK([]byte{})
		assert.Empty(t, pksks)
	})
}

// --------------------------------------------------------------------------
// assuanClient.pkdecrypt (mock server)
// --------------------------------------------------------------------------

func TestAssuanClientPkdecrypt(t *testing.T) {
	// Build a well-formed session key response.
	cipherByte := byte(9) // AES-256
	sessionKey := make([]byte, 32)
	for i := range sessionKey {
		sessionKey[i] = byte(i)
	}
	var sum uint16
	for _, b := range sessionKey {
		sum += uint16(b)
	}
	payload := append([]byte{cipherByte}, sessionKey...)
	payload = append(payload, byte(sum>>8), byte(sum))
	responseHex := strings.ToUpper(hex.EncodeToString(payload))

	runMockServer := func(serverConn net.Conn, serverResponse string) {
		defer func() { _ = serverConn.Close() }()
		r := bufio.NewReader(serverConn)
		// Read PKDECRYPT command line
		_, _ = r.ReadString('\n')
		// Send INQUIRE CIPHERTEXT
		_, _ = fmt.Fprintln(serverConn, "INQUIRE CIPHERTEXT")
		// Read D line
		_, _ = r.ReadString('\n')
		// Read END
		_, _ = r.ReadString('\n')
		// Send response
		_, _ = fmt.Fprintln(serverConn, serverResponse)
		_, _ = fmt.Fprintln(serverConn, "OK")
	}

	t.Run("valid response returns session key data", func(t *testing.T) {
		serverConn, clientConn := net.Pipe()
		go runMockServer(serverConn, fmt.Sprintf("D (value #%s#)", responseHex))

		c := &assuanClient{conn: clientConn, reader: bufio.NewReader(clientConn)}
		resp, err := c.pkdecrypt("DEADBEEF01234567", "(enc-val (rsa (a #AABB#)))")
		_ = clientConn.Close()
		require.NoError(t, err)
		assert.Contains(t, string(resp), "value")
	})

	t.Run("ERR response returns error", func(t *testing.T) {
		serverConn, clientConn := net.Pipe()
		go runMockServer(serverConn, "ERR 67108949 No such key")

		c := &assuanClient{conn: clientConn, reader: bufio.NewReader(clientConn)}
		_, err := c.pkdecrypt("DEADBEEF01234567", "(enc-val (rsa (a #AABB#)))")
		_ = clientConn.Close()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "No such key")
	})

	t.Run("unexpected first response returns error", func(t *testing.T) {
		serverConn, clientConn := net.Pipe()
		go func() {
			defer func() { _ = serverConn.Close() }()
			r := bufio.NewReader(serverConn)
			_, _ = r.ReadString('\n')
			_, _ = fmt.Fprintln(serverConn, "OK unexpected")
		}()

		c := &assuanClient{conn: clientConn, reader: bufio.NewReader(clientConn)}
		_, err := c.pkdecrypt("DEADBEEF01234567", "(enc-val (rsa (a #AABB#)))")
		_ = clientConn.Close()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "INQUIRE CIPHERTEXT")
	})
}

// --------------------------------------------------------------------------
// GPGAgentDecrypt (error paths without a running agent)
// --------------------------------------------------------------------------

func TestGPGAgentDecrypt(t *testing.T) {
	test.InitTestDirs()

	entity, _, err := CreateGPGEntity(testGPGName, "AgentDecryptTest", testGPGEmail, testGPGPass)
	require.NoError(t, err)

	pubFile := filepath.Join(test.TestData, "agent_decrypt"+pubGPGExt)
	privFile := filepath.Join(test.TestData, "agent_decrypt"+privGPGExt)
	require.NoError(t, ExportGPGKeyPair(entity, pubFile, privFile))

	plaintextFile := filepath.Join(test.TestData, "agent_decrypt.txt")
	cryptedFile := filepath.Join(test.TestData, "agent_decrypt.gpg")
	require.NoError(t, common.WriteStringToFile(plaintextFile, "agent-test-content"))
	require.NoError(t, GPGEncryptFile(plaintextFile, cryptedFile, pubFile))

	t.Run("returns error when gpg-agent socket is not found", func(t *testing.T) {
		emptyHome := filepath.Join(test.TestData, "gpg-agent-decrypt-empty")
		require.NoError(t, os.MkdirAll(emptyHome, 0700))
		t.Setenv("GPG_AGENT_INFO", "")
		t.Setenv(gpgEnvHome, emptyHome)
		t.Setenv("XDG_RUNTIME_DIR", "")

		_, agentErr := GPGAgentDecrypt(cryptedFile, openpgp.EntityList{entity})
		assert.Error(t, agentErr)
		assert.Contains(t, agentErr.Error(), "socket not found")
	})

	t.Run("returns error for nonexistent encrypted file", func(t *testing.T) {
		// Point to a real socket so we get past the socket lookup
		socketDir := filepath.Join(test.TestData, "gpg-agent-fake-socket")
		require.NoError(t, os.MkdirAll(socketDir, 0700))
		sockPath := filepath.Join(socketDir, "S.gpg-agent")
		require.NoError(t, os.WriteFile(sockPath, []byte("fake"), 0600))
		t.Setenv(gpgEnvHome, socketDir)

		_, agentErr := GPGAgentDecrypt(filepath.Join(test.TestData, "nonexistent.gpg"), openpgp.EntityList{entity})
		assert.Error(t, agentErr)
	})

	t.Run("returns error when no PKESK packets in file", func(t *testing.T) {
		// A plain text file has no PKESK packets
		emptyFile := filepath.Join(test.TestData, "agent_decrypt_empty.gpg")
		require.NoError(t, os.WriteFile(emptyFile, []byte("not a gpg file"), 0600))

		socketDir := filepath.Join(test.TestData, "gpg-agent-fake-socket2")
		require.NoError(t, os.MkdirAll(socketDir, 0700))
		sockPath := filepath.Join(socketDir, "S.gpg-agent")
		require.NoError(t, os.WriteFile(sockPath, []byte("fake"), 0600))
		t.Setenv(gpgEnvHome, socketDir)

		_, agentErr := GPGAgentDecrypt(emptyFile, openpgp.EntityList{entity})
		assert.Error(t, agentErr)
		assert.Contains(t, agentErr.Error(), "no PKESK")
	})
}

// --------------------------------------------------------------------------
// gpgAgentTryKey (via mock Assuan server — full round-trip with real session key)
// --------------------------------------------------------------------------

func TestGPGAgentTryKey(t *testing.T) {
	test.InitTestDirs()

	// Create entity, encrypt a file, and collect the PKESK.
	entity, _, err := CreateGPGEntity(testGPGName, "TryKeyTest", testGPGEmail, testGPGPass)
	require.NoError(t, err)

	pubFile := filepath.Join(test.TestData, "try_key"+pubGPGExt)
	privFile := filepath.Join(test.TestData, "try_key"+privGPGExt)
	require.NoError(t, ExportGPGKeyPair(entity, pubFile, privFile))

	plaintextFile := filepath.Join(test.TestData, "try_key.txt")
	cryptedFile := filepath.Join(test.TestData, "try_key.gpg")
	require.NoError(t, common.WriteStringToFile(plaintextFile, "try-key-content"))
	require.NoError(t, GPGEncryptFile(plaintextFile, cryptedFile, pubFile))

	fileData, readErr := os.ReadFile(cryptedFile)
	require.NoError(t, readErr)
	pksks := gpgCollectPKESK(fileData)
	require.NotEmpty(t, pksks)

	keyByID := gpgBuildKeyByID(openpgp.EntityList{entity})

	t.Run("unrecognised key ID returns error", func(t *testing.T) {
		serverConn, clientConn := net.Pipe()
		go func() { _ = serverConn.Close() }()

		c := &assuanClient{conn: clientConn, reader: bufio.NewReader(clientConn)}
		emptyMap := make(map[uint64]*packet.PublicKey)
		_, tryErr := gpgAgentTryKey(c, pksks[0], emptyMap, fileData)
		_ = clientConn.Close()
		assert.Error(t, tryErr)
		assert.Contains(t, tryErr.Error(), "not in entity list")
	})

	t.Run("agent connection error propagates", func(t *testing.T) {
		serverConn, clientConn := net.Pipe()
		// Server immediately closes → client reads EOF
		go func() {
			defer func() { _ = serverConn.Close() }()
			r := bufio.NewReader(serverConn)
			_, _ = r.ReadString('\n') // read PKDECRYPT command, then EOF on close
		}()

		c := &assuanClient{conn: clientConn, reader: bufio.NewReader(clientConn)}
		_, tryErr := gpgAgentTryKey(c, pksks[0], keyByID, fileData)
		_ = clientConn.Close()
		assert.Error(t, tryErr)
	})
}
