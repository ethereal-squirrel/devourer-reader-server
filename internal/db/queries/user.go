package queries

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/devourer/server/internal/db"
)

func GetUserByID(d *sql.DB, id int64) (*db.User, error) {
	row := d.QueryRow(`SELECT id, email, password, api_key, roles, metadata, created_at FROM User WHERE id=?`, id)
	return scanUser(row)
}

func GetUserByEmail(d *sql.DB, email string) (*db.User, error) {
	row := d.QueryRow(`SELECT id, email, password, api_key, roles, metadata, created_at FROM User WHERE email=?`, email)
	return scanUser(row)
}

func ListUsers(d *sql.DB) ([]*db.User, error) {
	rows, err := d.Query(`SELECT id, email, password, api_key, roles, metadata, created_at FROM User`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*db.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func CreateUser(d *sql.DB, email, password, apiKey string, roles []string, metadata json.RawMessage) (*db.User, error) {
	rolesJSON, err := json.Marshal(roles)
	if err != nil {
		return nil, err
	}

	res, err := d.Exec(
		`INSERT INTO User (email, password, api_key, roles, metadata, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		email, password, apiKey, string(rolesJSON), string(metadata),
		time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	id, _ := res.LastInsertId()
	return GetUserByID(d, id)
}

func UpdateUserPassword(d *sql.DB, id int64, hashedPassword string) error {
	_, err := d.Exec(`UPDATE User SET password=? WHERE id=?`, hashedPassword, id)
	return err
}

func UpdateUserRole(d *sql.DB, id int64, roles []string) error {
	rolesJSON, err := json.Marshal(roles)
	if err != nil {
		return err
	}
	_, err = d.Exec(`UPDATE User SET roles=? WHERE id=?`, string(rolesJSON), id)
	return err
}

func UpdateUserSettings(d *sql.DB, id int64, metadata json.RawMessage) error {
	_, err := d.Exec(`UPDATE User SET metadata=? WHERE id=?`, string(metadata), id)
	return err
}

func DeleteUser(d *sql.DB, id int64) error {
	_, err := d.Exec(`DELETE FROM User WHERE id=?`, id)
	return err
}

func GetRoleByTitle(d *sql.DB, title string) (*db.Roles, error) {
	row := d.QueryRow(`SELECT id, title, is_admin, add_file, delete_file, edit_metadata, manage_collections, manage_library, create_user FROM Roles WHERE title=?`, title)
	var r db.Roles
	if err := row.Scan(&r.ID, &r.Title, &r.IsAdmin, &r.AddFile, &r.DeleteFile, &r.EditMetadata, &r.ManageCollections, &r.ManageLibrary, &r.CreateUser); err != nil {
		return nil, err
	}
	return &r, nil
}

func ListRoles(d *sql.DB) ([]*db.Roles, error) {
	rows, err := d.Query(`SELECT id, title, is_admin, add_file, delete_file, edit_metadata, manage_collections, manage_library, create_user FROM Roles`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []*db.Roles
	for rows.Next() {
		var r db.Roles
		if err := rows.Scan(&r.ID, &r.Title, &r.IsAdmin, &r.AddFile, &r.DeleteFile, &r.EditMetadata, &r.ManageCollections, &r.ManageLibrary, &r.CreateUser); err != nil {
			return nil, err
		}
		roles = append(roles, &r)
	}
	return roles, rows.Err()
}

func scanUser(s rowScanner) (*db.User, error) {
	var u db.User
	var roles, metadata string
	if err := s.Scan(&u.ID, &u.Email, &u.Password, &u.APIKey, &roles, &metadata, &u.CreatedAt); err != nil {
		return nil, err
	}
	u.Roles = json.RawMessage(roles)
	u.Metadata = json.RawMessage(metadata)
	return &u, nil
}
