package auth

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/devourer/server/internal/db"
	"github.com/devourer/server/internal/db/queries"
)

const (
	jwtExpiry  = 14 * 24 * time.Hour
	bcryptCost = 12
)

var RolesData = map[string]db.RolePermissions{
	"admin": {
		IsAdmin: true, AddFile: true, DeleteFile: true,
		EditMetadata: true, ManageCollections: true,
		ManageLibrary: true, CreateUser: true,
	},
	"moderator": {
		IsAdmin: false, AddFile: true, DeleteFile: true,
		EditMetadata: true, ManageCollections: true,
		ManageLibrary: false, CreateUser: false,
	},
	"upload": {
		IsAdmin: false, AddFile: true, DeleteFile: false,
		EditMetadata: false, ManageCollections: false,
		ManageLibrary: false, CreateUser: false,
	},
	"user": {
		IsAdmin: false, AddFile: false, DeleteFile: false,
		EditMetadata: false, ManageCollections: false,
		ManageLibrary: false, CreateUser: false,
	},
}

func GetJWTSecret(d *sql.DB) (string, error) {
	cfg, err := queries.GetConfig(d, "jwt_secret")
	if err != nil {
		return "", fmt.Errorf("jwt_secret not found: %w", err)
	}
	return cfg.Value, nil
}

type TokenClaims struct {
	UserID int64    `json:"userId"`
	Email  string   `json:"email"`
	Roles  []string `json:"roles"`
	jwt.RegisteredClaims
}

func SignToken(d *sql.DB, userID int64, email string, roles []string) (string, error) {
	secret, err := GetJWTSecret(d)
	if err != nil {
		return "", err
	}

	claims := TokenClaims{
		UserID: userID,
		Email:  email,
		Roles:  roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(jwtExpiry)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func VerifyToken(d *sql.DB, tokenStr string) (*TokenClaims, error) {
	secret, err := GetJWTSecret(d)
	if err != nil {
		return nil, err
	}

	token, err := jwt.ParseWithClaims(tokenStr, &TokenClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*TokenClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

func ResolveRoles(rolesJSON json.RawMessage) db.RolePermissions {
	var roleNames []string
	if err := json.Unmarshal(rolesJSON, &roleNames); err != nil {
		return RolesData["user"]
	}

	perms := db.RolePermissions{}
	for _, name := range roleNames {
		if p, ok := RolesData[name]; ok {
			if p.IsAdmin {
				perms.IsAdmin = true
			}
			if p.AddFile {
				perms.AddFile = true
			}
			if p.DeleteFile {
				perms.DeleteFile = true
			}
			if p.EditMetadata {
				perms.EditMetadata = true
			}
			if p.ManageCollections {
				perms.ManageCollections = true
			}
			if p.ManageLibrary {
				perms.ManageLibrary = true
			}
			if p.CreateUser {
				perms.CreateUser = true
			}
		}
	}
	return perms
}

func HandleLogin(d *sql.DB, email, password string) (map[string]any, error) {
	user, err := queries.GetUserByEmail(d, email)
	if err != nil {
		return map[string]any{"status": false, "message": "Invalid credentials."}, nil
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return map[string]any{"status": false, "message": "Invalid credentials."}, nil
	}

	var roleNames []string
	json.Unmarshal(user.Roles, &roleNames)

	token, err := SignToken(d, user.ID, user.Email, roleNames)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"status": true,
		"token":  token,
		"user": map[string]any{
			"id":    user.ID,
			"email": user.Email,
			"roles": roleNames,
		},
		"message": "Login successful.",
	}, nil
}

func HandleRegister(d *sql.DB, email, password, role string) (map[string]any, error) {
	if email == "" || password == "" {
		return map[string]any{"status": false, "message": "All fields are required."}, nil
	}
	if len(password) < 8 {
		return map[string]any{"status": false, "message": "Password must be at least 8 characters long."}, nil
	}

	hashedPw, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return nil, err
	}

	apiKey := uuid.New().String()
	hashedAPIKey, err := bcrypt.GenerateFromPassword([]byte(apiKey), bcryptCost)
	if err != nil {
		return nil, err
	}

	defaultSettings := json.RawMessage(`{"settings":{"book_pagemode":"single","book_font":"default","book_background":"#000000","manga_direction":"ltr","manga_pagemode":"single","manga_resizemode":"fit","manga_background":"#000000"}}`)

	user, err := queries.CreateUser(d, email, string(hashedPw), string(hashedAPIKey), []string{role}, defaultSettings)
	if err != nil {
		log.Printf("[auth] create user error: %v", err)
		return map[string]any{"status": false, "message": "Registration failed."}, nil
	}

	token, err := SignToken(d, user.ID, user.Email, []string{role})
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"status": true,
		"token":  token,
		"user": map[string]any{
			"id":      user.ID,
			"email":   user.Email,
			"api_key": apiKey,
			"roles":   []string{role},
		},
		"message": "User created successfully.",
	}, nil
}

func HandleDeleteUser(d *sql.DB, id int64) (map[string]any, error) {
	user, err := queries.GetUserByID(d, id)
	if err != nil {
		return map[string]any{"status": false, "message": "User not found."}, nil
	}

	var roles []string
	json.Unmarshal(user.Roles, &roles)
	for _, r := range roles {
		if r == "admin" {
			return map[string]any{"status": false, "message": "Cannot delete admin user."}, nil
		}
	}

	queries.DeleteCollectionsByUserID(d, id)
	queries.DeleteRecentlyReadByUserID(d, id)
	queries.DeleteReadingStatusByUserID(d, id)
	queries.DeleteUser(d, id)

	return map[string]any{"status": true, "message": "User deleted successfully."}, nil
}

func HandleEditUser(d *sql.DB, id int64, role string, password *string) (map[string]any, error) {
	user, err := queries.GetUserByID(d, id)
	if err != nil {
		return map[string]any{"status": false, "message": "User not found."}, nil
	}

	if user.ID == 1 && role != "admin" {
		return map[string]any{"status": false, "message": "Cannot remove admin role from admin user."}, nil
	}

	if err := queries.UpdateUserRole(d, id, []string{role}); err != nil {
		return nil, err
	}

	if password != nil {
		hashed, err := bcrypt.GenerateFromPassword([]byte(*password), bcryptCost)
		if err != nil {
			return nil, err
		}
		if err := queries.UpdateUserPassword(d, id, string(hashed)); err != nil {
			return nil, err
		}
	}

	return map[string]any{"status": true, "message": "User updated successfully."}, nil
}

func ResetPassword(d *sql.DB, email, password string) (map[string]any, error) {
	user, err := queries.GetUserByEmail(d, email)
	if err != nil {
		return map[string]any{"status": true, "message": "If that email exists, the password has been reset."}, nil
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return nil, err
	}

	if err := queries.UpdateUserPassword(d, user.ID, string(hashed)); err != nil {
		return nil, err
	}

	return map[string]any{"status": true, "message": "Password reset successful."}, nil
}
