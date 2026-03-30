package queries

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetConfig_Exists(t *testing.T) {
	d := newTestDB(t)
	cfg, err := GetConfig(d, "jwt_secret")
	require.NoError(t, err)
	assert.Equal(t, "jwt_secret", cfg.Key)
	assert.Equal(t, "test-secret", cfg.Value)
}

func TestGetConfig_Missing(t *testing.T) {
	d := newTestDB(t)
	_, err := GetConfig(d, "nonexistent_key")
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestUpsertConfig_Insert(t *testing.T) {
	d := newTestDB(t)
	err := UpsertConfig(d, "new_key", "new_value")
	require.NoError(t, err)

	cfg, err := GetConfig(d, "new_key")
	require.NoError(t, err)
	assert.Equal(t, "new_value", cfg.Value)
}

func TestUpsertConfig_Update(t *testing.T) {
	d := newTestDB(t)

	err := UpsertConfig(d, "jwt_secret", "updated-secret")
	require.NoError(t, err)

	cfg, err := GetConfig(d, "jwt_secret")
	require.NoError(t, err)
	assert.Equal(t, "updated-secret", cfg.Value)

	var count int
	err = d.QueryRow(`SELECT COUNT(*) FROM Config WHERE key='jwt_secret'`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestGetAllConfig(t *testing.T) {
	d := newTestDB(t)

	all, err := GetAllConfig(d)
	require.NoError(t, err)
	assert.Equal(t, "test-secret", all["jwt_secret"])
	assert.Equal(t, "0", all["allow_public"])
	assert.Equal(t, "0", all["allow_register"])
}

func TestGetAllConfig_Empty(t *testing.T) {
	d, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { d.Close() })
	_, err = d.Exec(testSchema)
	require.NoError(t, err)

	all, err := GetAllConfig(d)
	require.NoError(t, err)
	assert.Empty(t, all)
}
