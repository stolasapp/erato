package sec

import "golang.org/x/crypto/bcrypt"

// ComparePassword returns an error if the provided password does not resolve to
// the given hash.
func ComparePassword[T ~string | ~[]byte](password T, hash []byte) error {
	return bcrypt.CompareHashAndPassword(hash, []byte(password))
}

// HashPassword generates the hash for a given password. It errors if the
// password is longer than 72 bytes.
func HashPassword[T ~string | ~[]byte](password T) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
}
