package queries

type rowScanner interface {
	Scan(dest ...any) error
}
