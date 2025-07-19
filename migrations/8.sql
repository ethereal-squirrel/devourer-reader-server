CREATE TABLE "UserRating" (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "user_id" INTEGER NOT NULL,
    "file_type" TEXT NOT NULL,
    "file_id" INTEGER NOT NULL,
    "rating" INTEGER NOT NULL
);

CREATE TABLE "UserTag" (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "user_id" INTEGER NOT NULL,
    "file_type" TEXT NOT NULL,
    "file_id" INTEGER NOT NULL,
    "tag" TEXT NOT NULL
);