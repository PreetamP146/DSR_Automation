package jwt

import (
	"errors"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

type TokenClaims struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	jwtlib.RegisteredClaims
}

type Service interface {
	GenerateAccessToken(userID, email string) (string, error)
	GenerateRefreshToken(userID, email string) (string, error)
	ValidateAccessToken(tokenString string) (*TokenClaims, error)
	ValidateRefreshToken(tokenString string) (*TokenClaims, error)
}

type jwtService struct {
	accessTokenSecret  string
	refreshTokenSecret string
	accessTokenExpiry  time.Duration
	refreshTokenExpiry time.Duration
}

func NewService(accessTokenSecret, refreshTokenSecret string) Service {
	return &jwtService{
		accessTokenSecret:  accessTokenSecret,
		refreshTokenSecret: refreshTokenSecret,
		accessTokenExpiry:  time.Hour,
		refreshTokenExpiry: 7 * 24 * time.Hour,
	}
}

func (s *jwtService) GenerateAccessToken(userID, email string) (string, error) {
	return s.generateToken(userID, email, s.accessTokenSecret, s.accessTokenExpiry)
}

func (s *jwtService) GenerateRefreshToken(userID, email string) (string, error) {
	return s.generateToken(userID, email, s.refreshTokenSecret, s.refreshTokenExpiry)
}

func (s *jwtService) ValidateAccessToken(tokenString string) (*TokenClaims, error) {
	return s.validateToken(tokenString, s.accessTokenSecret)
}

func (s *jwtService) ValidateRefreshToken(tokenString string) (*TokenClaims, error) {
	return s.validateToken(tokenString, s.refreshTokenSecret)
}

func (s *jwtService) generateToken(userID, email, secret string, expiry time.Duration) (string, error) {
	claims := TokenClaims{
		ID:    userID,
		Email: email,
		RegisteredClaims: jwtlib.RegisteredClaims{
			ExpiresAt: jwtlib.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwtlib.NewNumericDate(time.Now()),
			NotBefore: jwtlib.NewNumericDate(time.Now()),
		},
	}

	token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func (s *jwtService) validateToken(tokenString, secret string) (*TokenClaims, error) {
	claims := &TokenClaims{}
	token, err := jwtlib.ParseWithClaims(tokenString, claims, func(token *jwtlib.Token) (any, error) {
		if _, ok := token.Method.(*jwtlib.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
