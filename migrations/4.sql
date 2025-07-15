PRAGMA defer_foreign_keys=ON;
PRAGMA foreign_keys=OFF;
-- AlterTable
CREATE TABLE "new_BookFile" (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "title" TEXT NOT NULL,
    "path" TEXT NOT NULL,
    "file_name" TEXT NOT NULL,
    "file_format" TEXT NOT NULL,
    "total_pages" INTEGER NOT NULL,
    "current_page" TEXT NOT NULL,
    "is_read" BOOLEAN NOT NULL,
    "library_id" INTEGER NOT NULL,
    "metadata" JSONB NOT NULL,
    "formats" JSONB NOT NULL,
    "tags" JSONB NOT NULL,
    CONSTRAINT "BookFile_library_id_fkey" FOREIGN KEY ("library_id") REFERENCES "Library" ("id") ON DELETE RESTRICT ON UPDATE CASCADE
);
INSERT INTO "new_BookFile" ("current_page", "file_format", "file_name", "id", "is_read", "library_id", "metadata", "path", "title", "total_pages", "formats", "tags") SELECT "current_page", "file_format", "file_name", "id", "is_read", "library_id", "metadata", "path", "title", "total_pages", "formats", "tags" FROM "BookFile";
DROP TABLE "BookFile";
ALTER TABLE "new_BookFile" RENAME TO "BookFile";

-- AlterTable
CREATE TABLE "new_RecentlyRead" (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "is_local" BOOLEAN NOT NULL,
    "library_id" INTEGER NOT NULL,
    "series_id" INTEGER NOT NULL,
    "file_id" INTEGER NOT NULL,
    "current_page" TEXT NOT NULL,
    "total_pages" INTEGER NOT NULL,
    "volume" INTEGER NOT NULL,
    "chapter" INTEGER NOT NULL,
    "user_id" INTEGER NOT NULL
);
INSERT INTO "new_RecentlyRead" ("chapter", "current_page", "file_id", "id", "is_local", "library_id", "series_id", "total_pages", "user_id", "volume") SELECT "chapter", "current_page", "file_id", "id", "is_local", "library_id", "series_id", "total_pages", "user_id", "volume" FROM "RecentlyRead";
DROP TABLE "RecentlyRead";
ALTER TABLE "new_RecentlyRead" RENAME TO "RecentlyRead";
PRAGMA foreign_keys=ON;
PRAGMA defer_foreign_keys=OFF;