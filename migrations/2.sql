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
    "formats" JSONB NOT NULL DEFAULT '[]',
    CONSTRAINT "BookFile_library_id_fkey" FOREIGN KEY ("library_id") REFERENCES "Library" ("id") ON DELETE RESTRICT ON UPDATE CASCADE
);
INSERT INTO "new_BookFile" ("current_page", "file_format", "file_name", "id", "is_read", "library_id", "metadata", "path", "title", "total_pages") SELECT "current_page", "file_format", "file_name", "id", "is_read", "library_id", "metadata", "path", "title", "total_pages" FROM "BookFile";
DROP TABLE "BookFile";
ALTER TABLE "new_BookFile" RENAME TO "BookFile";