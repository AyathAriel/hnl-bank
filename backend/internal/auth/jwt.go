package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// PurposePending2FA marca un token como intermedio: solo sirve para
// completar el segundo factor (POST /api/auth/2fa/verify), nunca para
// autenticar contra el resto de la API. auth.Middleware lo rechaza
// explícitamente para que un token pendiente nunca otorgue acceso real.
const PurposePending2FA = "2fa_pending"

type Claims struct {
	UserID  string `json:"user_id"`
	Email   string `json:"email"`
	Purpose string `json:"purpose,omitempty"`
	jwt.RegisteredClaims
}

// GenerateToken crea un JWT de sesión completo (HS256) para el usuario dado.
// Devuelve el token y el jti (usado para poder revocarlo en logout).
func GenerateToken(secret, userID, email string, expiryHours int) (token string, jti string, err error) {
	return generateToken(secret, userID, email, "", time.Duration(expiryHours)*time.Hour)
}

// GeneratePending2FAToken crea un token de muy corta duración (5 minutos)
// que solo prueba que el usuario ya pasó la verificación de contraseña;
// hace falta completar el segundo factor para canjearlo por una sesión real.
func GeneratePending2FAToken(secret, userID, email string) (token string, err error) {
	token, _, err = generateToken(secret, userID, email, PurposePending2FA, 5*time.Minute)
	return token, err
}

func generateToken(secret, userID, email, purpose string, ttl time.Duration) (token string, jti string, err error) {
	jti = uuid.NewString()
	now := time.Now()

	claims := Claims{
		UserID:  userID,
		Email:   email,
		Purpose: purpose,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := t.SignedString([]byte(secret))
	if err != nil {
		return "", "", fmt.Errorf("signing token: %w", err)
	}
	return signed, jti, nil
}

// ParseToken valida la firma y expiración de un JWT y devuelve sus claims.
func ParseToken(secret, tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}
