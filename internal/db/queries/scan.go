package queries

// rowScanner is satisfied by both *sql.Row and *sql.Rows, allowing the
// scan* helpers to accept either without duplicating the interface in every file.
type rowScanner interface {
	Scan(dest ...any) error
}
