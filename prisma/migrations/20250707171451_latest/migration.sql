/*
  Warnings:

  - Added the required column `metadata` to the `User` table without a default value. This is not possible if the table is not empty.

*/
-- RedefineTables
PRAGMA defer_foreign_keys=ON;
PRAGMA foreign_keys=OFF;
CREATE TABLE "new_User" (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "email" TEXT NOT NULL,
    "password" TEXT NOT NULL,
    "api_key" TEXT NOT NULL,
    "roles" JSONB NOT NULL,
    "metadata" JSONB NOT NULL,
    "created_at" DATETIME NOT NULL
);
INSERT INTO "new_User" ("api_key", "created_at", "email", "id", "password", "roles") SELECT "api_key", "created_at", "email", "id", "password", "roles" FROM "User";
DROP TABLE "User";
ALTER TABLE "new_User" RENAME TO "User";
CREATE UNIQUE INDEX "User_email_key" ON "User"("email");
PRAGMA foreign_keys=ON;
PRAGMA defer_foreign_keys=OFF;
