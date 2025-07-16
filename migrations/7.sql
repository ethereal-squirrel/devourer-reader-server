CREATE TABLE "ReadingStatus" (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "user_id" INTEGER NOT NULL,
    "file_type" TEXT NOT NULL,
    "file_id" INTEGER NOT NULL,
    "current_page" TEXT NOT NULL
);