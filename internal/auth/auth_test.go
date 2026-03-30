package auth

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/devourer/server/internal/db"
	"github.com/devourer/server/internal/testutil"
)

var testBcryptHash string

func TestMain(m *testing.M) {
	h, err := bcrypt.GenerateFromPassword([]byte("password123"), bcryptCost)
	if err != nil {
		panic(err)
	}
	testBcryptHash = string(h)
	os.Exit(m.Run())
}

// ---------------------------------------------------------------------------
// ResolveRoles
// ---------------------------------------------------------------------------

func TestResolveRoles(t *testing.T) {
	cases := []struct {
		name      string
		rolesJSON string
		want      db.RolePermissions
	}{
		{
			name:      "admin gets all permissions",
			rolesJSON: `["admin"]`,
			want:      db.RolePermissions{IsAdmin: true, AddFile: true, DeleteFile: true, EditMetadata: true, ManageCollections: true, ManageLibrary: true, CreateUser: true},
		},
		{
			name:      "user gets no permissions",
			rolesJSON: `["user"]`,
			want:      db.RolePermissions{},
		},
		{
			name:      "moderator cannot manage library or create users",
			rolesJSON: `["moderator"]`,
			want:      db.RolePermissions{AddFile: true, DeleteFile: true, EditMetadata: true, ManageCollections: true},
		},
		{
			name:      "upload gets only AddFile",
			rolesJSON: `["upload"]`,
			want:      db.RolePermissions{AddFile: true},
		},
		{
			name:      "upload+moderator union merges permissions",
			rolesJSON: `["upload","moderator"]`,
			want:      db.RolePermissions{AddFile: true, DeleteFile: true, EditMetadata: true, ManageCollections: true},
		},
		{
			name:      "unknown role name is ignored",
			rolesJSON: `["superadmin"]`,
			want:      db.RolePermissions{},
		},
		{
			name:      "empty array gives no permissions",
			rolesJSON: `[]`,
			want:      db.RolePermissions{},
		},
		{
			name:      "invalid JSON falls back to user (all false)",
			rolesJSON: `not-valid-json`,
			want:      db.RolePermissions{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveRoles(json.RawMessage(tc.rolesJSON))
			assert.Equal(t, tc.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// GetJWTSecret / SignToken / VerifyToken
// ---------------------------------------------------------------------------

func TestGetJWTSecret_Found(t *testing.T) {
	d := testutil.NewDB(t)
	secret, err := GetJWTSecret(d)
	require.NoError(t, err)
	assert.NotEmpty(t, secret)
}

func TestGetJWTSecret_Missing(t *testing.T) {
	d := testutil.NewDBNoSecret(t)
	_, err := GetJWTSecret(d)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "jwt_secret not found")
}

func TestSignToken_Valid(t *testing.T) {
	d := testutil.NewDB(t)
	tok, err := SignToken(d, 42, "user@example.com", []string{"user"})
	require.NoError(t, err)
	assert.NotEmpty(t, tok)
}

func TestVerifyToken_RoundTrip(t *testing.T) {
	d := testutil.NewDB(t)
	roles := []string{"admin"}
	tok, err := SignToken(d, 7, "admin@example.com", roles)
	require.NoError(t, err)

	claims, err := VerifyToken(d, tok)
	require.NoError(t, err)
	assert.Equal(t, int64(7), claims.UserID)
	assert.Equal(t, "admin@example.com", claims.Email)
	assert.Equal(t, roles, claims.Roles)
}

func TestVerifyToken_WrongSecret(t *testing.T) {
	d1 := testutil.NewDB(t)
	d2 := testutil.NewDBNoSecret(t)

	_, err := d2.Exec(`INSERT INTO Config (key, value) VALUES ('jwt_secret', 'completely-different-secret')`)
	require.NoError(t, err)

	tok, err := SignToken(d1, 1, "a@b.com", []string{"user"})
	require.NoError(t, err)

	_, err = VerifyToken(d2, tok)
	assert.Error(t, err)
}

func TestVerifyToken_Tampered(t *testing.T) {
	d := testutil.NewDB(t)
	tok, err := SignToken(d, 1, "a@b.com", []string{"user"})
	require.NoError(t, err)

	_, err = VerifyToken(d, tok+"x")
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// HandleLogin
// ---------------------------------------------------------------------------

func TestHandleLogin_Success(t *testing.T) {
	d := testutil.NewDB(t)
	testutil.MustInsertUser(t, d, "alice@example.com", testBcryptHash, []string{"user"})

	result, err := HandleLogin(d, "alice@example.com", "password123")
	require.NoError(t, err)
	assert.True(t, result["status"].(bool))
	assert.NotEmpty(t, result["token"])
}

func TestHandleLogin_WrongPassword(t *testing.T) {
	d := testutil.NewDB(t)
	testutil.MustInsertUser(t, d, "bob@example.com", testBcryptHash, []string{"user"})

	result, err := HandleLogin(d, "bob@example.com", "wrongpassword")
	require.NoError(t, err)
	assert.False(t, result["status"].(bool))
	assert.Equal(t, "Invalid credentials.", result["message"])
}

func TestHandleLogin_NonexistentUser(t *testing.T) {
	d := testutil.NewDB(t)

	result, err := HandleLogin(d, "nobody@example.com", "password123")
	require.NoError(t, err)
	assert.False(t, result["status"].(bool))
	assert.Equal(t, "Invalid credentials.", result["message"])
}

// ---------------------------------------------------------------------------
// HandleRegister (these tests are slow due to bcrypt at cost 12)
// ---------------------------------------------------------------------------

func TestHandleRegister_EmptyEmail(t *testing.T) {
	d := testutil.NewDB(t)
	result, err := HandleRegister(d, "", "password123", "user")
	require.NoError(t, err)
	assert.False(t, result["status"].(bool))
	assert.Equal(t, "All fields are required.", result["message"])
}

func TestHandleRegister_ShortPassword(t *testing.T) {
	d := testutil.NewDB(t)
	result, err := HandleRegister(d, "new@example.com", "short", "user")
	require.NoError(t, err)
	assert.False(t, result["status"].(bool))
	assert.Equal(t, "Password must be at least 8 characters long.", result["message"])
}

func TestHandleRegister_Success(t *testing.T) {
	d := testutil.NewDB(t)
	result, err := HandleRegister(d, "newuser@example.com", "securepassword", "user")
	require.NoError(t, err)
	assert.True(t, result["status"].(bool))
	assert.NotEmpty(t, result["token"])
	userMap := result["user"].(map[string]any)
	assert.NotEmpty(t, userMap["api_key"])
}

func TestHandleRegister_DuplicateEmail(t *testing.T) {
	d := testutil.NewDB(t)
	testutil.MustInsertUser(t, d, "existing@example.com", testBcryptHash, []string{"user"})

	result, err := HandleRegister(d, "existing@example.com", "password123", "user")
	require.NoError(t, err)
	assert.False(t, result["status"].(bool))
	assert.Equal(t, "Registration failed.", result["message"])
}

// ---------------------------------------------------------------------------
// HandleDeleteUser
// ---------------------------------------------------------------------------

func TestHandleDeleteUser_NotFound(t *testing.T) {
	d := testutil.NewDB(t)
	result, err := HandleDeleteUser(d, 9999)
	require.NoError(t, err)
	assert.False(t, result["status"].(bool))
	assert.Equal(t, "User not found.", result["message"])
}

func TestHandleDeleteUser_AdminBlocked(t *testing.T) {
	d := testutil.NewDB(t)
	id := testutil.MustInsertUser(t, d, "admin@example.com", testBcryptHash, []string{"admin"})

	result, err := HandleDeleteUser(d, id)
	require.NoError(t, err)
	assert.False(t, result["status"].(bool))
	assert.Equal(t, "Cannot delete admin user.", result["message"])
}

func TestHandleDeleteUser_Success(t *testing.T) {
	d := testutil.NewDB(t)
	id := testutil.MustInsertUser(t, d, "regular@example.com", testBcryptHash, []string{"user"})

	result, err := HandleDeleteUser(d, id)
	require.NoError(t, err)
	assert.True(t, result["status"].(bool))

	var count int
	err = d.QueryRow(`SELECT COUNT(*) FROM User WHERE id=?`, id).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

// ---------------------------------------------------------------------------
// HandleEditUser
// ---------------------------------------------------------------------------

func TestHandleEditUser_NotFound(t *testing.T) {
	d := testutil.NewDB(t)
	result, err := HandleEditUser(d, 9999, "user", nil)
	require.NoError(t, err)
	assert.False(t, result["status"].(bool))
	assert.Equal(t, "User not found.", result["message"])
}

func TestHandleEditUser_ProtectsUserID1(t *testing.T) {
	d := testutil.NewDB(t)

	_, err := d.Exec(`INSERT INTO User (id, email, password, api_key, roles, metadata, created_at) VALUES (1, 'admin@x.com', 'hash', 'key', '["admin"]', '{}', '2024-01-01T00:00:00Z')`)
	require.NoError(t, err)

	result, err := HandleEditUser(d, 1, "user", nil)
	require.NoError(t, err)
	assert.False(t, result["status"].(bool))
	assert.Equal(t, "Cannot remove admin role from admin user.", result["message"])
}

func TestHandleEditUser_UpdatesRole(t *testing.T) {
	d := testutil.NewDB(t)

	testutil.MustInsertUser(t, d, "placeholder@example.com", testBcryptHash, []string{"admin"})
	id := testutil.MustInsertUser(t, d, "editor@example.com", testBcryptHash, []string{"user"})

	result, err := HandleEditUser(d, id, "moderator", nil)
	require.NoError(t, err)
	assert.True(t, result["status"].(bool))

	var rolesStr string
	err = d.QueryRow(`SELECT roles FROM User WHERE id=?`, id).Scan(&rolesStr)
	require.NoError(t, err)
	assert.Equal(t, `["moderator"]`, rolesStr)
}

// ---------------------------------------------------------------------------
// ResetPassword
// ---------------------------------------------------------------------------

func TestResetPassword_NonexistentEmail(t *testing.T) {
	d := testutil.NewDB(t)

	result, err := ResetPassword(d, "ghost@example.com", "newpassword")
	require.NoError(t, err)
	assert.True(t, result["status"].(bool))
}
