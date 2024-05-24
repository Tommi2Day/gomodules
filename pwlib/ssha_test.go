package pwlib

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var sshaPlain = []byte("password")

func TestSSHA(t *testing.T) {
	t.Run("TestSSHAEncoder_Encode", func(t *testing.T) {
		enc := SSHAEncoder{}
		encoded, err := enc.Encode(sshaPlain, SSHAPrefix)
		require.NoError(t, err)
		assert.Contains(t, string(encoded), SSHAPrefix)
		assert.Greater(t, len(encoded), 6)
		t.Log(string(encoded))
	})

	t.Run("TestSSHAEncoder_Matches", func(t *testing.T) {
		enc := SSHAEncoder{}
		encoded, err := enc.Encode(sshaPlain, SSHAPrefix)
		require.NoError(t, err)
		assert.True(t, enc.Matches(encoded, sshaPlain))
	})

	t.Run("TestMakeSSHAHash", func(t *testing.T) {
		salt, err := makeSSHASalt()
		require.NoError(t, err)
		hash := makeSSHAHash(sshaPlain, salt)
		assert.Equal(t, 24, len(hash))
		t.Log(hash)
	})

	t.Run("TestMakeSalt", func(t *testing.T) {
		salt, err := makeSSHASalt()
		require.NoError(t, err)
		assert.Equal(t, 4, len(salt))
	})
}
