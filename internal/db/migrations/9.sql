CREATE TABLE "AudiobookSeries" (
    "id"             INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "title"          TEXT    NOT NULL,
    "path"           TEXT    NOT NULL,
    "cover"          TEXT    NOT NULL DEFAULT '',
    "library_id"     INTEGER NOT NULL,
    "audiobook_data" JSONB   NOT NULL DEFAULT '{}',
    CONSTRAINT "AudiobookSeries_library_id_fkey"
        FOREIGN KEY ("library_id") REFERENCES "Library" ("id")
        ON DELETE RESTRICT ON UPDATE CASCADE
);
CREATE UNIQUE INDEX "AudiobookSeries_path_key" ON "AudiobookSeries"("path");

CREATE TABLE "AudiobookFile" (
    "id"                       INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "path"                     TEXT    NOT NULL,
    "file_name"                TEXT    NOT NULL,
    "file_format"              TEXT    NOT NULL,
    "track_number"             INTEGER NOT NULL DEFAULT 0,
    "duration_seconds"         INTEGER NOT NULL DEFAULT 0,
    "current_position_seconds" TEXT    NOT NULL DEFAULT '0',
    "is_listened"              BOOLEAN NOT NULL DEFAULT FALSE,
    "series_id"                INTEGER NOT NULL,
    "metadata"                 JSONB   NOT NULL DEFAULT '{}',
    CONSTRAINT "AudiobookFile_series_id_fkey"
        FOREIGN KEY ("series_id") REFERENCES "AudiobookSeries" ("id")
        ON DELETE RESTRICT ON UPDATE CASCADE
);
CREATE UNIQUE INDEX "AudiobookFile_path_key" ON "AudiobookFile"("path");
