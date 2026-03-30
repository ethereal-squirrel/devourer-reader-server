package queries

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpsertReadingStatus_CreatesNew(t *testing.T) {
	d := newTestDB(t)

	err := UpsertReadingStatus(d, 1, 10, "book", "42")
	require.NoError(t, err)

	rs, err := GetReadingStatus(d, 1, 10, "book")
	require.NoError(t, err)
	assert.Equal(t, int64(1), rs.UserID)
	assert.Equal(t, int64(10), rs.FileID)
	assert.Equal(t, "book", rs.FileType)
	assert.Equal(t, "42", rs.CurrentPage)
}

func TestUpsertReadingStatus_UpdatesExisting(t *testing.T) {
	d := newTestDB(t)

	require.NoError(t, UpsertReadingStatus(d, 1, 10, "book", "1"))
	require.NoError(t, UpsertReadingStatus(d, 1, 10, "book", "99"))

	rs, err := GetReadingStatus(d, 1, 10, "book")
	require.NoError(t, err)
	assert.Equal(t, "99", rs.CurrentPage)

	var count int
	err = d.QueryRow(`SELECT COUNT(*) FROM ReadingStatus WHERE user_id=1 AND file_id=10 AND file_type='book'`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestGetReadingStatus_NotFound(t *testing.T) {
	d := newTestDB(t)
	_, err := GetReadingStatus(d, 99, 99, "book")
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestDeleteReadingStatus(t *testing.T) {
	d := newTestDB(t)
	require.NoError(t, UpsertReadingStatus(d, 1, 5, "manga", "3"))

	err := DeleteReadingStatus(d, 1, 5, "manga")
	require.NoError(t, err)

	_, err = GetReadingStatus(d, 1, 5, "manga")
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestDeleteReadingStatusByUserID(t *testing.T) {
	d := newTestDB(t)
	require.NoError(t, UpsertReadingStatus(d, 7, 1, "book", "10"))
	require.NoError(t, UpsertReadingStatus(d, 7, 2, "book", "20"))
	require.NoError(t, UpsertReadingStatus(d, 8, 1, "book", "5"))

	err := DeleteReadingStatusByUserID(d, 7)
	require.NoError(t, err)

	var count int
	err = d.QueryRow(`SELECT COUNT(*) FROM ReadingStatus WHERE user_id=7`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	err = d.QueryRow(`SELECT COUNT(*) FROM ReadingStatus WHERE user_id=8`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestDeleteReadingStatusByFileType(t *testing.T) {
	d := newTestDB(t)
	require.NoError(t, UpsertReadingStatus(d, 1, 10, "book", "5"))
	require.NoError(t, UpsertReadingStatus(d, 1, 20, "book", "8"))
	require.NoError(t, UpsertReadingStatus(d, 1, 30, "manga", "1"))

	err := DeleteReadingStatusByFileType(d, "book", []int64{10, 20})
	require.NoError(t, err)

	var count int
	err = d.QueryRow(`SELECT COUNT(*) FROM ReadingStatus WHERE file_type='book'`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	err = d.QueryRow(`SELECT COUNT(*) FROM ReadingStatus WHERE file_type='manga'`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestDeleteReadingStatusByFileType_EmptyIDs(t *testing.T) {
	d := newTestDB(t)
	require.NoError(t, UpsertReadingStatus(d, 1, 10, "book", "5"))

	err := DeleteReadingStatusByFileType(d, "book", []int64{})
	require.NoError(t, err)

	var count int
	err = d.QueryRow(`SELECT COUNT(*) FROM ReadingStatus`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestDeleteReadingStatusByFileID(t *testing.T) {
	d := newTestDB(t)
	require.NoError(t, UpsertReadingStatus(d, 1, 55, "book", "3"))
	require.NoError(t, UpsertReadingStatus(d, 2, 55, "book", "7"))
	require.NoError(t, UpsertReadingStatus(d, 1, 66, "book", "1"))

	err := DeleteReadingStatusByFileID(d, 55)
	require.NoError(t, err)

	var count int
	err = d.QueryRow(`SELECT COUNT(*) FROM ReadingStatus WHERE file_id=55`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	err = d.QueryRow(`SELECT COUNT(*) FROM ReadingStatus WHERE file_id=66`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}
