package security

import (
	"encoding/hex"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/sha3"
	"math/rand"
	"regexp"
	"time"
)

const CharsetAlpha = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const Charset = CharsetAlpha + "0123456789_-"

var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))

func HashBcrypt(plain string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(plain), 14)
	return string(bytes), err
}

func CheckBcryptHash(plain, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
	return err == nil
}

func IsBcryptHash(s string) bool {
	bcryptRegex := regexp.MustCompile(`^\$2[abxy]\$\d\d\$[./A-Za-z0-9]{53}$`)
	return bcryptRegex.MatchString(s)
}

func HashSha3256(plain string) string {
	h := sha3.New256()
	h.Write([]byte(plain))
	return hex.EncodeToString(h.Sum(nil))
}

func CheckSha3256Hash(plain, hash string) bool {
	return hash == HashSha3256(plain)
}

func RandomString(n int) string {
	return RandomStringWithCharset(n, Charset)
}

func RandomStringWithCharset(n int, letters string) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[seededRand.Intn(len(letters))]
	}
	return string(b)
}
