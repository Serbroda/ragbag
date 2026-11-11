package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// --- Two-factor codes (TOTP, RFC 6238: SHA-1, 6 digits, 30 seconds: what every authenticator app understands) ---

const (
	totpPeriod = 30
	totpDigits = 6
)

var totpEncoding = base32.StdEncoding.WithPadding(base32.NoPadding)

// NewTOTPSecret returns 160 random bits in the Base32 spelling authenticator apps take.
func NewTOTPSecret() string {
	secret := make([]byte, 20)
	_, _ = rand.Read(secret)
	return totpEncoding.EncodeToString(secret)
}

// TOTPURI is what the QR code holds: the secret with the names the app lists the account under.
func TOTPURI(issuer, account, secret string) string {
	query := url.Values{"secret": {secret}, "issuer": {issuer}}
	return "otpauth://totp/" + url.PathEscape(issuer+":"+account) + "?" + query.Encode()
}

// TOTPStep is the number of the 30 second window a time lies in.
func TOTPStep(at time.Time) int64 { return at.Unix() / totpPeriod }

func totpCode(secret string, step int64) (string, error) {
	key, err := totpEncoding.DecodeString(strings.ToUpper(strings.TrimSpace(secret)))
	if err != nil || len(key) == 0 {
		return "", errors.New("invalid two-factor secret")
	}
	mac := hmac.New(sha1.New, key)
	_ = binary.Write(mac, binary.BigEndian, uint64(step))
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	value := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff
	return fmt.Sprintf("%0*d", totpDigits, value%1_000_000), nil
}

// TOTPCode is the code of the secret at the given time.
func TOTPCode(secret string, at time.Time) (string, error) { return totpCode(secret, TOTPStep(at)) }

// CheckTOTP accepts the code of the current window and of the ones right before and after it (clocks differ,
// people type slowly), but none at or before lastStep: a code works once. It returns the window that matched.
func CheckTOTP(secret, code string, at time.Time, lastStep int64) (int64, bool) {
	code = strings.ReplaceAll(strings.TrimSpace(code), " ", "")
	if len(code) != totpDigits {
		return 0, false
	}
	for _, step := range []int64{TOTPStep(at), TOTPStep(at) - 1, TOTPStep(at) + 1} {
		expected, err := totpCode(secret, step)
		if err != nil {
			return 0, false
		}
		if step > lastStep && subtle.ConstantTimeCompare([]byte(expected), []byte(code)) == 1 {
			return step, true
		}
	}
	return 0, false
}

// NewRecoveryCode is a one-time replacement for a code of the app: 50 random bits as xxxxx-xxxxx.
func NewRecoveryCode() string {
	random := make([]byte, 7)
	_, _ = rand.Read(random)
	code := strings.ToLower(totpEncoding.EncodeToString(random))[:10]
	return code[:5] + "-" + code[5:]
}

// NormalizeRecoveryCode makes typed codes comparable: case, spaces and the dash do not matter.
func NormalizeRecoveryCode(code string) string {
	return strings.NewReplacer(" ", "", "-", "").Replace(strings.ToLower(strings.TrimSpace(code)))
}

// SecretBox encrypts values the server has to read again (two-factor secrets cannot be hashed) with a key
// derived from the server's secret, so a copy of the database alone does not reveal them.
type SecretBox struct{ aead cipher.AEAD }

func NewSecretBox(secret []byte, purpose string) *SecretBox {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(purpose))
	block, _ := aes.NewCipher(mac.Sum(nil))
	aead, _ := cipher.NewGCM(block)
	return &SecretBox{aead: aead}
}

func (b *SecretBox) Seal(plain string) string {
	nonce := make([]byte, b.aead.NonceSize())
	_, _ = rand.Read(nonce)
	return base64.RawStdEncoding.EncodeToString(b.aead.Seal(nonce, nonce, []byte(plain), nil))
}

// Open fails for values sealed with another server secret.
func (b *SecretBox) Open(sealed string) (string, error) {
	raw, err := base64.RawStdEncoding.DecodeString(sealed)
	if err != nil || len(raw) < b.aead.NonceSize() {
		return "", errors.New("invalid sealed value")
	}
	plain, err := b.aead.Open(nil, raw[:b.aead.NonceSize()], raw[b.aead.NonceSize():], nil)
	return string(plain), err
}
