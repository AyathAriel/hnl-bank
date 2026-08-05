package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

type contextKey string

const (
	userIDContextKey contextKey = "user_id"
	emailContextKey  contextKey = "email"
	jtiContextKey    contextKey = "jti"
)

// IsRevokedFunc consulta si un jti fue invalidado (logout).
type IsRevokedFunc func(ctx context.Context, jti string) (bool, error)

// Middleware valida el header Authorization: Bearer <token>, lo verifica contra
// la blacklist de revocación y añade el usuario autenticado al contexto.
func Middleware(secret string, isRevoked IsRevokedFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				writeUnauthorized(w, "Falta el encabezado de autenticación.")
				return
			}
			tokenString := strings.TrimPrefix(header, "Bearer ")

			claims, err := ParseToken(secret, tokenString)
			if err != nil {
				writeUnauthorized(w, "Sesión inválida o expirada. Inicia sesión de nuevo.")
				return
			}
			if claims.Purpose != "" {
				// Token intermedio (ej. pendiente de 2FA): nunca es una sesión válida.
				writeUnauthorized(w, "Sesión inválida o expirada. Inicia sesión de nuevo.")
				return
			}

			revoked, err := isRevoked(r.Context(), claims.ID)
			if err != nil {
				writeUnauthorized(w, "No se pudo verificar la sesión.")
				return
			}
			if revoked {
				writeUnauthorized(w, "La sesión fue cerrada. Inicia sesión de nuevo.")
				return
			}

			ctx := context.WithValue(r.Context(), userIDContextKey, claims.UserID)
			ctx = context.WithValue(ctx, emailContextKey, claims.Email)
			ctx = context.WithValue(ctx, jtiContextKey, claims.ID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(userIDContextKey).(string)
	return v
}

func EmailFromContext(ctx context.Context) string {
	v, _ := ctx.Value(emailContextKey).(string)
	return v
}

func JTIFromContext(ctx context.Context) string {
	v, _ := ctx.Value(jtiContextKey).(string)
	return v
}

func writeUnauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
