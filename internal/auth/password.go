package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const saltLength = 16
const memory = 64 * 1024
const iterations = 3
const parallelism = 2
const keyLength = 32

// Hash takes a string and returns a build string in argon2id format
func Hash(plain string) (string, error) {
	salt := make([]byte, saltLength)
	_, err := rand.Read(salt)
	if err != nil {
		return "", fmt.Errorf("auth_Hash: error reading rand salt bytes: %w", err)
	}

	hash := argon2.IDKey([]byte(plain), salt, iterations, memory, parallelism, keyLength)

	b64salt := base64.RawStdEncoding.EncodeToString(salt)
	b64hash := base64.RawStdEncoding.EncodeToString(hash)

	bhash := fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", memory, iterations, parallelism, b64salt, b64hash)

	return bhash, nil
}

// Verify gets a string and saved hash and verifies them against each other
func Verify(plain string, phcString string) (bool, error) {
	dbArray := strings.Split(phcString, "$")

	if len(dbArray) != 6 {
		return false, fmt.Errorf("auth_Verify: array length not correct; expected: 6; got: %d", len(dbArray))
	}

	switch dbArray[1] {
	case "argon2id":
		// do nothing on correct case
	case "argon2i":
		return false, fmt.Errorf("auth_Verify: argon2i is not allowed")
	case "argon2d":
		return false, fmt.Errorf("auth_Verify: argon2d is not allowed")
	default:
		return false, fmt.Errorf("auth_Verify: not a recognized algorithm; expected: argon2id; got: %s", dbArray[1])
	}

	if dbArray[2] != "v=19" {
		return false, fmt.Errorf("auth_Verify: not a recognized version; expected: v=19; got: %s", dbArray[2])
	}

	paramArray := strings.Split(dbArray[3], ",")

	var memInt, iterInt, paraInt int
	var err error
	for _, param := range paramArray {
		kv := strings.Split(param, "=")
		if kv[0] == "" || kv[1] == "" {
			return false, fmt.Errorf("auth_Verify: parameters malformed: %s;%s", kv[0], kv[1])
		}
		switch kv[0] {
		case "m":
			memInt, err = strconv.Atoi(kv[1])
			if err != nil {
				return false, fmt.Errorf("auth_Verify: memory param not convertable to int: %w", err)
			}
			if memInt != memory {
				return false, fmt.Errorf("auth_Verify: memory param wrong; expected: %d; got: %d", memory, memInt)
			}
		case "t":
			iterInt, err = strconv.Atoi(kv[1])
			if err != nil {
				return false, fmt.Errorf("auth_Verify: iterations param not convertable to int: %w", err)
			}
			if iterInt != iterations {
				return false, fmt.Errorf("auth_Verify: iterations wrong; expected: %d; got: %d", iterations, iterInt)
			}
		case "p":
			paraInt, err = strconv.Atoi(kv[1])
			if err != nil {
				return false, fmt.Errorf("auth_Verify: paralellism param not convertable to int: %w", err)
			}
			if paraInt != parallelism {
				return false, fmt.Errorf("auth_Verify: paralellism param wrong; expected: %d; got: %d", parallelism, paraInt)
			}
		default:
			return false, fmt.Errorf("auth_Verify: parameter not recognized; expected: (m,t,p); got: %s", kv[0])
		}
	}

	if memInt == 0 {
		return false, fmt.Errorf("auth_Verify: memory param not present")
	}
	if iterInt == 0 {
		return false, fmt.Errorf("auth_Verify: iter param not present")
	}
	if paraInt == 0 {
		return false, fmt.Errorf("auth_Verify: paralellism param not present")
	}

	var decSalt, decHash []byte
	if dbArray[4] != "" {
		decSalt, err = base64.RawStdEncoding.DecodeString(dbArray[4])
		if err != nil {
			return false, fmt.Errorf("auth_Verify: salt cant be decoded: %w", err)
		}
	} else {
		return false, fmt.Errorf("auth_Verify: salt is empty")
	}

	if dbArray[5] != "" {
		decHash, err = base64.RawStdEncoding.DecodeString(dbArray[5])
		if err != nil {
			return false, fmt.Errorf("auth_Verify: hash cant be decoded: %w", err)
		}
		if len(decHash) != keyLength {
			return false, fmt.Errorf("auth_Verify: hash hash wrong length")
		}
	} else {
		return false, fmt.Errorf("auth_Verify: hash is empty")
	}

	reKey := argon2.IDKey([]byte(plain), decSalt, uint32(iterInt), uint32(memInt), uint8(paraInt), keyLength)

	same := subtle.ConstantTimeCompare(decHash, reKey)
	if same == 1 {
		return true, nil
	}
	return false, nil
}
