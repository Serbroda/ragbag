package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// The SHA-1 test vectors of RFC 6238 (8 digits there, the last 6 are ours); secret "12345678901234567890"
func TestTOTPMatchesTheRFC(t *testing.T) {
	secret := totpEncoding.EncodeToString([]byte("12345678901234567890"))
	for unix, expected := range map[int64]string{59: "287082", 1111111109: "081804", 1234567890: "005924", 20000000000: "353130"} {
		code, err := TOTPCode(secret, time.Unix(unix, 0))
		require.NoError(t, err)
		require.Equal(t, expected, code)
	}
}

func TestCheckTOTPAllowsNeighbouringWindowsButNoReuse(t *testing.T) {
	secret := NewTOTPSecret()
	now := time.Unix(1_700_000_000, 0)
	code, _ := TOTPCode(secret, now)

	step, ok := CheckTOTP(secret, " "+code[:3]+" "+code[3:], now, 0)
	require.True(t, ok)
	require.Equal(t, TOTPStep(now), step)
	_, ok = CheckTOTP(secret, code, now.Add(30*time.Second), 0)
	require.True(t, ok, "the code of the window before")
	_, ok = CheckTOTP(secret, code, now.Add(-30*time.Second), 0)
	require.True(t, ok, "the code of the window after")
	_, ok = CheckTOTP(secret, code, now.Add(90*time.Second), 0)
	require.False(t, ok, "too old")
	_, ok = CheckTOTP(secret, code, now, step)
	require.False(t, ok, "used already")
	_, ok = CheckTOTP(secret, "12345", now, 0)
	require.False(t, ok)
	_, ok = CheckTOTP("not base32!", code, now, 0)
	require.False(t, ok)
}

func TestSecretBoxOnlyOpensWithTheSameSecret(t *testing.T) {
	box := NewSecretBox([]byte("server secret"), "totp")
	sealed := box.Seal("JBSWY3DPEHPK3PXP")
	require.NotContains(t, sealed, "JBSWY3DPEHPK3PXP")
	require.NotEqual(t, sealed, box.Seal("JBSWY3DPEHPK3PXP"))
	plain, err := box.Open(sealed)
	require.NoError(t, err)
	require.Equal(t, "JBSWY3DPEHPK3PXP", plain)

	_, err = NewSecretBox([]byte("another secret"), "totp").Open(sealed)
	require.Error(t, err)
	_, err = NewSecretBox([]byte("server secret"), "other").Open(sealed)
	require.Error(t, err)
	_, err = box.Open("garbage")
	require.Error(t, err)
}

func TestRecoveryCodes(t *testing.T) {
	code := NewRecoveryCode()
	require.Regexp(t, `^[a-z2-7]{5}-[a-z2-7]{5}$`, code)
	require.NotEqual(t, code, NewRecoveryCode())
	require.Equal(t, "abcde23456", NormalizeRecoveryCode(" ABCDE-23456 "))
}
