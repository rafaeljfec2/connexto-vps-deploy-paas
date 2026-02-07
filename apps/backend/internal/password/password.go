package password

import "golang.org/x/crypto/bcrypt"

const defaultCost = 12

// Hash returns a bcrypt hash of the plain-text password.
func Hash(plain string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(plain), defaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// Verify compares a bcrypt hashed password with a plain-text candidate.
// Returns nil on success or an error if they do not match.
func Verify(hash, plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
}
