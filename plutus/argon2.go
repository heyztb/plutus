package plutus

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"runtime"
	"strings"

	"github.com/awnumar/memguard"
	"golang.org/x/crypto/argon2"
)

type params struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
	saltLength  uint32
	keyLength   uint32
}

func hashFromString(s string) (string, error) {

	// the number of logical cpus available on the machine running this code
	threadCount := uint8(runtime.NumCPU())

	p := &params{
		memory:      64 * 1024,
		iterations:  1,
		parallelism: threadCount,
		saltLength:  16,
		keyLength:   32,
	}

	salt, err := generateRandomBytes(p.saltLength)
	if err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(s), salt, p.iterations, p.memory, p.parallelism, p.keyLength)

	encodedSalt := base64.RawStdEncoding.EncodeToString(salt)
	encodedHash := base64.RawStdEncoding.EncodeToString(hash)

	formattedHash := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, p.memory, p.iterations, p.parallelism, encodedSalt, encodedHash)

	return formattedHash, nil
}

func compareStringAndHash(s string, encodedHash string) (bool, error) {
	match := false
	p, salt, hash, err := decodeHash(encodedHash)
	if err != nil {
		return match, err
	}

	// Generate a new hash to compare to the one we've got with the same exact parameters
	comparableHash := argon2.IDKey([]byte(s), salt, p.iterations, p.memory, p.parallelism, p.keyLength)

	// Check that the contents of the two hashes are identical. We are using a constant time comparison function to prevent timing attacks :)
	if subtle.ConstantTimeCompare(hash, comparableHash) == 1 {
		match = true
	}

	memguard.WipeBytes(salt)
	memguard.WipeBytes(hash)
	memguard.WipeBytes(comparableHash)

	return match, nil
}

func decodeHash(encodedHash string) (*params, []byte, []byte, error) {
	vals := strings.Split(encodedHash, "$")
	if len(vals) != 6 {
		return nil, nil, nil, errors.New("Invalid hash: incorrect format")
	}

	var version int
	_, err := fmt.Sscanf(vals[2], "v=%d", &version)
	if err != nil {
		return nil, nil, nil, err
	}

	if version != argon2.Version {
		return nil, nil, nil, errors.New("Invalid hash: wrong version")
	}

	p := &params{}

	_, err = fmt.Sscanf(vals[3], "m=%d,t=%d,p=%d", &p.memory, &p.iterations, &p.parallelism)
	if err != nil {
		return nil, nil, nil, err
	}

	salt, err := base64.RawStdEncoding.DecodeString(vals[4])
	if err != nil {
		return nil, nil, nil, err
	}

	p.saltLength = uint32(len(salt))

	hash, err := base64.RawStdEncoding.DecodeString(vals[5])
	if err != nil {
		return nil, nil, nil, err
	}

	p.keyLength = uint32(len(hash))

	return p, salt, hash, nil
}

// generate random bytes to use as a salt
func generateRandomBytes(length uint32) ([]byte, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}
