package middleware

import (
	"crypto/sha256"
	"crypto/subtle"
	"log"
	"net/http"
	"strings"
)

type authenticationMiddleware struct {
	expectedToken string
}

func NewAuthMiddleware(expectedToken string) *authenticationMiddleware {
	return &authenticationMiddleware{
		expectedToken: expectedToken,
	}
}

func (amw *authenticationMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			log.Print("authorization header not provided")
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		authFields := strings.Fields(authHeader)
		authType := strings.ToLower(authFields[0]) //should be bearer
		authToken := strings.ToLower(authFields[1])

		if authType != "bearer" {
			log.Printf("unsupported authorization type %s", authType)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		tokenHash := sha256.Sum256([]byte(authToken))
		expectedTokenHash := sha256.Sum256([]byte(amw.expectedToken))
		tokenMatch := subtle.ConstantTimeCompare(tokenHash[:], expectedTokenHash[:]) == 1

		if tokenMatch {
			log.Printf("Authenticated")
			next.ServeHTTP(w, r)
		} else {
			log.Print("Request is unauthenticated")
			http.Error(w, "Forbidden", http.StatusForbidden)
		}
	})
}
