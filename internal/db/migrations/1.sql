-- CreateTable
CREATE TABLE "Config" (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "key" TEXT NOT NULL,
    "value" TEXT NOT NULL
);

-- CreateTable
CREATE TABLE "User" (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "email" TEXT NOT NULL,
    "password" TEXT NOT NULL,
    "api_key" TEXT NOT NULL,
    "roles" JSONB NOT NULL,
    "metadata" JSONB NOT NULL,
    "created_at" DATETIME NOT NULL
);

-- CreateTable
CREATE TABLE "Library" (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "name" TEXT NOT NULL,
    "path" TEXT NOT NULL,
    "type" TEXT NOT NULL,
    "metadata" JSONB NOT NULL
);

-- CreateTable
CREATE TABLE "BookFile" (
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
    CONSTRAINT "BookFile_library_id_fkey" FOREIGN KEY ("library_id") REFERENCES "Library" ("id") ON DELETE RESTRICT ON UPDATE CASCADE
);

-- CreateTable
CREATE TABLE "MangaSeries" (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "title" TEXT NOT NULL,
    "path" TEXT NOT NULL,
    "cover" TEXT NOT NULL,
    "library_id" INTEGER NOT NULL,
    "manga_data" JSONB NOT NULL,
    CONSTRAINT "MangaSeries_library_id_fkey" FOREIGN KEY ("library_id") REFERENCES "Library" ("id") ON DELETE RESTRICT ON UPDATE CASCADE
);

-- CreateTable
CREATE TABLE "MangaFile" (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "path" TEXT NOT NULL,
    "file_name" TEXT NOT NULL,
    "file_format" TEXT NOT NULL,
    "volume" INTEGER NOT NULL,
    "chapter" INTEGER NOT NULL,
    "total_pages" INTEGER NOT NULL,
    "current_page" INTEGER NOT NULL,
    "is_read" BOOLEAN NOT NULL,
    "series_id" INTEGER NOT NULL,
    "metadata" JSONB NOT NULL,
    CONSTRAINT "MangaFile_series_id_fkey" FOREIGN KEY ("series_id") REFERENCES "MangaSeries" ("id") ON DELETE RESTRICT ON UPDATE CASCADE
);

-- CreateTable
CREATE TABLE "RecentlyRead" (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "is_local" BOOLEAN NOT NULL,
    "library_id" INTEGER NOT NULL,
    "series_id" INTEGER NOT NULL,
    "file_id" INTEGER NOT NULL,
    "current_page" INTEGER NOT NULL,
    "total_pages" INTEGER NOT NULL,
    "volume" INTEGER NOT NULL,
    "chapter" INTEGER NOT NULL,
    "user_id" INTEGER NOT NULL
);

-- CreateTable
CREATE TABLE "Collection" (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "library_id" INTEGER NOT NULL,
    "name" TEXT NOT NULL,
    "series" JSONB NOT NULL,
    "user_id" INTEGER NOT NULL
);

-- CreateIndex
CREATE UNIQUE INDEX "Config_key_key" ON "Config"("key");

-- CreateIndex
CREATE UNIQUE INDEX "User_email_key" ON "User"("email");

-- CreateIndex
CREATE UNIQUE INDEX "Library_name_key" ON "Library"("name");

-- CreateIndex
CREATE UNIQUE INDEX "Library_path_key" ON "Library"("path");

-- CreateIndex
CREATE UNIQUE INDEX "BookFile_path_key" ON "BookFile"("path");

-- CreateIndex
CREATE UNIQUE INDEX "MangaSeries_path_key" ON "MangaSeries"("path");

-- CreateIndex
CREATE UNIQUE INDEX "MangaFile_path_key" ON "MangaFile"("path");
