package db

import (
	"encoding/json"
	"path/filepath"
)

type Config struct {
	ID    int64  `json:"id"`
	Key   string `json:"key"`
	Value string `json:"value"`
}

type User struct {
	ID        int64           `json:"id"`
	Email     string          `json:"email"`
	Password  string          `json:"-"`
	APIKey    string          `json:"-"`
	Roles     json.RawMessage `json:"roles"`
	Metadata  json.RawMessage `json:"metadata"`
	CreatedAt string          `json:"created_at"`
}

type Roles struct {
	ID                int64  `json:"id"`
	Title             string `json:"title"`
	IsAdmin           bool   `json:"is_admin"`
	AddFile           bool   `json:"add_file"`
	DeleteFile        bool   `json:"delete_file"`
	EditMetadata      bool   `json:"edit_metadata"`
	ManageCollections bool   `json:"manage_collections"`
	ManageLibrary     bool   `json:"manage_library"`
	CreateUser        bool   `json:"create_user"`
}

type RolePermissions struct {
	IsAdmin           bool `json:"is_admin"`
	AddFile           bool `json:"add_file"`
	DeleteFile        bool `json:"delete_file"`
	EditMetadata      bool `json:"edit_metadata"`
	ManageCollections bool `json:"manage_collections"`
	ManageLibrary     bool `json:"manage_library"`
	CreateUser        bool `json:"create_user"`
}

type Library struct {
	ID       int64           `json:"id"`
	Name     string          `json:"name"`
	Path     string          `json:"path"`
	Type     string          `json:"type"`
	Metadata json.RawMessage `json:"metadata"`
}

func (l *Library) MetaBase() string {
	var meta map[string]any
	if json.Unmarshal(l.Metadata, &meta) == nil {
		if p, ok := meta["metadataPath"].(string); ok && p != "" {
			return p
		}
	}
	return filepath.Join(l.Path, ".devourer")
}

type BookFile struct {
	ID          int64           `json:"id"`
	Title       string          `json:"title"`
	Path        string          `json:"path"`
	FileName    string          `json:"file_name"`
	FileFormat  string          `json:"file_format"`
	TotalPages  int             `json:"total_pages"`
	CurrentPage string          `json:"current_page"`
	IsRead      bool            `json:"is_read"`
	LibraryID   int64           `json:"library_id"`
	Metadata    json.RawMessage `json:"metadata"`
	Formats     json.RawMessage `json:"formats"`
	Tags        json.RawMessage `json:"tags"`
}

type MangaSeries struct {
	ID        int64           `json:"id"`
	Title     string          `json:"title"`
	Path      string          `json:"path"`
	Cover     string          `json:"cover"`
	LibraryID int64           `json:"library_id"`
	MangaData json.RawMessage `json:"manga_data"`
}

type MangaFile struct {
	ID          int64           `json:"id"`
	Path        string          `json:"path"`
	FileName    string          `json:"file_name"`
	FileFormat  string          `json:"file_format"`
	Volume      int             `json:"volume"`
	Chapter     int             `json:"chapter"`
	TotalPages  int             `json:"total_pages"`
	CurrentPage int             `json:"current_page"`
	IsRead      bool            `json:"is_read"`
	SeriesID    int64           `json:"series_id"`
	Metadata    json.RawMessage `json:"metadata"`
}

type RecentlyRead struct {
	ID          int64  `json:"id"`
	IsLocal     bool   `json:"is_local"`
	LibraryID   int64  `json:"library_id"`
	SeriesID    int64  `json:"series_id"`
	FileID      int64  `json:"file_id"`
	CurrentPage string `json:"current_page"`
	TotalPages  int    `json:"total_pages"`
	Volume      int    `json:"volume"`
	Chapter     int    `json:"chapter"`
	UserID      int64  `json:"user_id"`
}

type ReadingStatus struct {
	ID          int64  `json:"id"`
	UserID      int64  `json:"user_id"`
	FileType    string `json:"file_type"`
	FileID      int64  `json:"file_id"`
	CurrentPage string `json:"current_page"`
}

type UserRating struct {
	ID       int64  `json:"id"`
	UserID   int64  `json:"user_id"`
	FileType string `json:"file_type"`
	FileID   int64  `json:"file_id"`
	Rating   int    `json:"rating"`
}

type UserTag struct {
	ID       int64  `json:"id"`
	UserID   int64  `json:"user_id"`
	FileType string `json:"file_type"`
	FileID   int64  `json:"file_id"`
	Tag      string `json:"tag"`
}

type Collection struct {
	ID        int64           `json:"id"`
	LibraryID int64           `json:"library_id"`
	Name      string          `json:"name"`
	Series    json.RawMessage `json:"series"`
	UserID    int64           `json:"user_id"`
}
