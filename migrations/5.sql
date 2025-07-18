-- CreateTable
CREATE TABLE "Roles" (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "title" TEXT NOT NULL,
    "is_admin" BOOLEAN NOT NULL,
    "add_file" BOOLEAN NOT NULL,
    "delete_file" BOOLEAN NOT NULL,
    "edit_metadata" BOOLEAN NOT NULL,
    "manage_collections" BOOLEAN NOT NULL,
    "manage_library" BOOLEAN NOT NULL,
    "create_user" BOOLEAN NOT NULL
);

INSERT INTO "Roles" ("title", "is_admin", "add_file", "delete_file", "edit_metadata", "manage_collections", "manage_library", "create_user") VALUES
('admin', 1, 1, 1, 1, 1, 1, 1),
('moderator', 0, 1, 1, 1, 1, 0, 0),
('upload', 0, 1, 0, 0, 0, 0, 0),
('user', 0, 0, 0, 0, 0, 0, 0);