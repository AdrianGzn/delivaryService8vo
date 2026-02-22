package middleware

import (
	"context"
	"net/http"
	"strings"

)

type AuthMiddleware struct {
	// Aquí podrías tener una instancia de tu manejador de usuarios
}

func (m *AuthMiddleware) ValidateToken(token string) (int, string, error) {
	// Formato: "user_id:role"
	parts := strings.Split(token, ":")
	if len(parts) == 2 {
		return 1, parts[1], nil
	}
	return 0, "", nil
}

func (m *AuthMiddleware) Authenticate(next http.HandlerFunc, allowedRoles ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			http.Error(w, "Token no proporcionado", http.StatusUnauthorized)
			return
		}

		
		token = strings.TrimPrefix(token, "Bearer ")

		userId, role, err := m.ValidateToken(token)
		if err != nil || userId == 0 {
			http.Error(w, "Token inválido", http.StatusUnauthorized)
			return
		}

		
		if len(allowedRoles) > 0 {
			roleAllowed := false
			for _, allowedRole := range allowedRoles {
				if role == allowedRole {
					roleAllowed = true
					break
				}
			}
			if !roleAllowed {
				http.Error(w, "No tiene permisos para acceder", http.StatusForbidden)
				return
			}
		}

		
		ctx := context.WithValue(r.Context(), "user_id", userId)
		ctx = context.WithValue(ctx, "user_role", role)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}