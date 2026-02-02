package tokens

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
)

type Service struct {
	Secret          []byte
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

type AccessClaims struct {
	jwt.RegisteredClaims
}

func (s Service) NewAccessToken(userID string, now time.Time) (string, time.Time, error) {
	if len(s.Secret) == 0 {
		return "", time.Time{}, errors.New("missing jwt secret")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	exp := now.Add(s.AccessTokenTTL)

	claims := AccessClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}

	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString(s.Secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, exp, nil
}

func (s Service) ParseAccessToken(tokenString string) (*AccessClaims, error) {
	parsed, err := jwt.ParseWithClaims(tokenString, &AccessClaims{}, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		return s.Secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := parsed.Claims.(*AccessClaims)
	if !ok || !parsed.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

func NewRefreshToken() (raw string, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	raw = base64.RawURLEncoding.EncodeToString(b)
	sum := sha256.Sum256([]byte(raw))
	hash = hex.EncodeToString(sum[:])
	return raw, hash, nil
}
