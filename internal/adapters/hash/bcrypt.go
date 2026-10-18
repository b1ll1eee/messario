// Package hash implements app.PasswordHasher with bcrypt.
package hash

import "golang.org/x/crypto/bcrypt"

// Bcrypt hashes with the given cost (bcrypt.DefaultCost when zero).
type Bcrypt struct{ Cost int }

func (b Bcrypt) Hash(password string) (string, error) {
	cost := b.Cost
	if cost == 0 {
		cost = bcrypt.DefaultCost
	}
	out, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	return string(out), err
}

func (b Bcrypt) Compare(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
