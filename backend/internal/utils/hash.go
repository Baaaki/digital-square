// backend/internal/utils/hash.go
package utils

import (
    "crypto/rand"
    "crypto/subtle"
    "encoding/base64"
    "errors"
    "fmt"
    "strings"
    
    "golang.org/x/crypto/argon2"
)

// Argon2 parameters
const (
    SaltLength  = 16
    Memory      = 64 * 1024  // 64 MB
    Iterations  = 1
    Parallelism = 4
    KeyLength   = 32
)

var (
    ErrInvalidHash         = errors.New("invalid hash format")
    ErrIncompatibleVersion = errors.New("incompatible argon2 version")
)

// HashPassword generates Argon2id hash
// Output format: $argon2id$v=19$m=65536,t=1,p=4$salt$hash
func HashPassword(password string) (string, error) {
    // Generate random salt
    salt := make([]byte, SaltLength)
    if _, err := rand.Read(salt); err != nil {
        return "", err
    }
    
    // Generate hash
    hash := argon2.IDKey(
        []byte(password),
        salt,
        Iterations,
        Memory,
        Parallelism,
        KeyLength,
    )
    
    // Encode: $argon2id$v=19$m=65536,t=1,p=4$salt$hash
    encoded := fmt.Sprintf(
        "$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
        argon2.Version,
        Memory,
        Iterations,
        Parallelism,
        base64.RawStdEncoding.EncodeToString(salt),
        base64.RawStdEncoding.EncodeToString(hash),
    )
    
    return encoded, nil
}

// VerifyPassword checks if password matches hash
func VerifyPassword(password, encodedHash string) (bool, error) {
    // Parse encoded hash
    salt, hash, params, err := decodeHash(encodedHash)
    if err != nil {
        return false, err
    }
    
    // Generate hash with same params
    testHash := argon2.IDKey(
        []byte(password),
        salt,
        params.iterations,
        params.memory,
        params.parallelism,
        params.keyLength,
    )
    
    // Constant-time comparison (prevent timing attacks)
    if subtle.ConstantTimeCompare(hash, testHash) == 1 {
        return true, nil
    }
    
    return false, nil
}

// hashParams holds Argon2 parameters
type hashParams struct {
    memory      uint32
    iterations  uint32
    parallelism uint8
    saltLength  uint32
    keyLength   uint32
}

// decodeHash parses encoded hash string
// Format: $argon2id$v=19$m=65536,t=1,p=4$salt$hash
func decodeHash(encodedHash string) (salt, hash []byte, params *hashParams, err error) {
    parts := strings.Split(encodedHash, "$")
    if len(parts) != 6 {
        return nil, nil, nil, ErrInvalidHash
    }
    
    // Check version
    var version int
    if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
        return nil, nil, nil, err
    }
    if version != argon2.Version {
        return nil, nil, nil, ErrIncompatibleVersion
    }
    
    // Parse parameters
    params = &hashParams{}
    _, err = fmt.Sscanf(
        parts[3],
        "m=%d,t=%d,p=%d",
        &params.memory,
        &params.iterations,
        &params.parallelism,
    )
    if err != nil {
        return nil, nil, nil, err
    }
    
    // Decode salt
    salt, err = base64.RawStdEncoding.DecodeString(parts[4])
    if err != nil {
        return nil, nil, nil, err
    }
    params.saltLength = uint32(len(salt))
    
    // Decode hash
    hash, err = base64.RawStdEncoding.DecodeString(parts[5])
    if err != nil {
        return nil, nil, nil, err
    }
    params.keyLength = uint32(len(hash))
    
    return salt, hash, params, nil
}