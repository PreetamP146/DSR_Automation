package passwordhashing

import "golang.org/x/crypto/bcrypt"

// Hasher hashes passwords and can verify them.
type Hasher interface {
	HashPassword(password string) (string, error)
	ComparePassword(hash, password string) error
}

type bcryptHasher struct {
	cost int
}

func NewBcryptHasher(cost int) Hasher {
	return &bcryptHasher{cost: cost}
}

func (h *bcryptHasher) HashPassword(password string) (string, error) {
	cost := h.cost
	if cost == 0 {
		cost = bcrypt.DefaultCost
	}
	b, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (h *bcryptHasher) ComparePassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
