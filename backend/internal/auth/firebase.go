package auth

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"firebase.google.com/go/v4/auth"
	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/option"
)

type contextKey string

const UserIDKey contextKey = "userID"

type AuthService struct {
	authClient *auth.Client
}

func NewAuthService(ctx context.Context) (*AuthService, error) {
	credsPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if credsPath == "" {
		log.Println("WARN: GOOGLE_APPLICATION_CREDENTIALS environment variable not set.")
	}

	var opt option.ClientOption
	if credsPath != "" {
		opt = option.WithCredentialsFile(credsPath)
	} else {
		log.Println("INFO: Attempting to use Application Default Credentials for Firebase.")
	}

	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		return nil, fmt.Errorf("error initializing Firebase app: %w", err)
	}

	authClient, err := app.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting Firebase Auth client: %w", err)
	}

	log.Println("Firebase Auth service initialized successfully.")
	return &AuthService{authClient: authClient}, nil
}

func (s *AuthService) VerifyIDToken(ctx context.Context, idToken string) (*auth.Token, error) {
	token, err := s.authClient.VerifyIDToken(ctx, idToken)
	if err != nil {
		return nil, fmt.Errorf("error verifying Firebase ID token: %w", err)
	}
	return token, nil
}

func (s *AuthService) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header required", http.StatusUnauthorized)
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				http.Error(w, "Invalid Authorization header format (Bearer token required)", http.StatusUnauthorized)
				return
			}
			idToken := parts[1]

			verifiedToken, err := s.VerifyIDToken(r.Context(), idToken)
			if err != nil {
				log.Printf("WARN: Failed to verify token: %v", err)
				http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, verifiedToken.UID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserIDKey).(string)
	return userID, ok
}


