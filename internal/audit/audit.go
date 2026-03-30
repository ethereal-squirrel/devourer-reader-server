package audit

import "log/slog"

func Login(ip, email string) {
	slog.Info("audit", "event", "login", "ip", ip, "email", email)
}

func LoginFailed(ip, email string) {
	slog.Warn("audit", "event", "login_failed", "ip", ip, "email", email)
}

func UserCreated(actorIP, email, role string) {
	slog.Info("audit", "event", "user_created", "ip", actorIP, "email", email, "role", role)
}

func UserDeleted(actorIP string, targetID int64) {
	slog.Info("audit", "event", "user_deleted", "ip", actorIP, "target_id", targetID)
}

func UserEdited(actorIP string, targetID int64, newRole string) {
	slog.Info("audit", "event", "user_edited", "ip", actorIP, "target_id", targetID, "new_role", newRole)
}

func PasswordReset(ip, email string) {
	slog.Info("audit", "event", "password_reset", "ip", ip, "email", email)
}

func FileUploaded(ip string, libraryID int64, filename string) {
	slog.Info("audit", "event", "file_uploaded", "ip", ip, "library_id", libraryID, "filename", filename)
}

func LibraryCreated(ip, name, path string) {
	slog.Info("audit", "event", "library_created", "ip", ip, "name", name, "path", path)
}

func LibraryDeleted(ip string, libraryID int64) {
	slog.Info("audit", "event", "library_deleted", "ip", ip, "library_id", libraryID)
}

func LibraryScanned(ip string, libraryID int64) {
	slog.Info("audit", "event", "library_scanned", "ip", ip, "library_id", libraryID)
}
