package queries

import (
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateUser_RoundTrip(t *testing.T) {
	d := newTestDB(t)
	meta := json.RawMessage(`{"settings":{}}`)

	u, err := CreateUser(d, "alice@example.com", "hashed-pw", "hashed-api-key", []string{"user"}, meta)
	require.NoError(t, err)
	require.NotNil(t, u)

	assert.Equal(t, "alice@example.com", u.Email)
	assert.Equal(t, "hashed-pw", u.Password)

	byID, err := GetUserByID(d, u.ID)
	require.NoError(t, err)
	assert.Equal(t, u.ID, byID.ID)
	assert.Equal(t, "alice@example.com", byID.Email)

	byEmail, err := GetUserByEmail(d, "alice@example.com")
	require.NoError(t, err)
	assert.Equal(t, u.ID, byEmail.ID)
}

func TestCreateUser_DuplicateEmail(t *testing.T) {
	d := newTestDB(t)
	meta := json.RawMessage(`{"settings":{}}`)

	_, err := CreateUser(d, "dup@example.com", "pw1", "key1", []string{"user"}, meta)
	require.NoError(t, err)

	_, err = CreateUser(d, "dup@example.com", "pw2", "key2", []string{"user"}, meta)
	assert.Error(t, err, "inserting duplicate email should fail")
}

func TestGetUserByID_NotFound(t *testing.T) {
	d := newTestDB(t)
	_, err := GetUserByID(d, 99999)
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestGetUserByEmail_NotFound(t *testing.T) {
	d := newTestDB(t)
	_, err := GetUserByEmail(d, "nobody@example.com")
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestListUsers_Empty(t *testing.T) {
	d := newTestDB(t)
	users, err := ListUsers(d)
	require.NoError(t, err)
	assert.Empty(t, users)
}

func TestListUsers_Multiple(t *testing.T) {
	d := newTestDB(t)
	meta := json.RawMessage(`{"settings":{}}`)

	_, err := CreateUser(d, "user1@example.com", "pw", "key1", []string{"user"}, meta)
	require.NoError(t, err)
	_, err = CreateUser(d, "user2@example.com", "pw", "key2", []string{"user"}, meta)
	require.NoError(t, err)

	users, err := ListUsers(d)
	require.NoError(t, err)
	assert.Len(t, users, 2)
}

func TestUpdateUserPassword(t *testing.T) {
	d := newTestDB(t)
	id := insertUser(t, d, "pw-user@example.com", "original-hash", []string{"user"})

	err := UpdateUserPassword(d, id, "new-hash")
	require.NoError(t, err)

	u, err := GetUserByID(d, id)
	require.NoError(t, err)
	assert.Equal(t, "new-hash", u.Password)
}

func TestUpdateUserRole_Roundtrip(t *testing.T) {
	d := newTestDB(t)
	id := insertUser(t, d, "role-user@example.com", "hash", []string{"user"})

	err := UpdateUserRole(d, id, []string{"moderator"})
	require.NoError(t, err)

	u, err := GetUserByID(d, id)
	require.NoError(t, err)

	var roles []string
	require.NoError(t, json.Unmarshal(u.Roles, &roles))
	assert.Equal(t, []string{"moderator"}, roles)
}

func TestDeleteUser(t *testing.T) {
	d := newTestDB(t)
	id := insertUser(t, d, "to-delete@example.com", "hash", []string{"user"})

	err := DeleteUser(d, id)
	require.NoError(t, err)

	_, err = GetUserByID(d, id)
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestGetRoleByTitle_Found(t *testing.T) {
	d := newTestDB(t)

	_, err := d.Exec(`INSERT INTO Roles (title,is_admin,add_file,delete_file,edit_metadata,manage_collections,manage_library,create_user) VALUES ('admin',1,1,1,1,1,1,1)`)
	require.NoError(t, err)

	role, err := GetRoleByTitle(d, "admin")
	require.NoError(t, err)
	assert.Equal(t, "admin", role.Title)
	assert.True(t, role.IsAdmin)
}

func TestGetRoleByTitle_NotFound(t *testing.T) {
	d := newTestDB(t)
	_, err := GetRoleByTitle(d, "nonexistent")
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestListRoles(t *testing.T) {
	d := newTestDB(t)
	_, err := d.Exec(`INSERT INTO Roles (title,is_admin,add_file,delete_file,edit_metadata,manage_collections,manage_library,create_user) VALUES
		('admin',1,1,1,1,1,1,1),
		('user',0,0,0,0,0,0,0)`)
	require.NoError(t, err)

	roles, err := ListRoles(d)
	require.NoError(t, err)
	assert.Len(t, roles, 2)
}
