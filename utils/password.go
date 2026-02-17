package utils

import "golang.org/x/crypto/bcrypt"

// HashAuthorityPassword hashes a plaintext password for storage. Never store plaintext.
func HashAuthorityPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// CheckAuthorityPassword returns nil if plain matches stored bcrypt hash.
func CheckAuthorityPassword(plain, hashed string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain))
}
