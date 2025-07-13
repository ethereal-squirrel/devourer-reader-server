import { PrismaBetterSQLite3 } from "@prisma/adapter-better-sqlite3";
import { PrismaClient } from "../generated/prisma";
import path from "path";

const globalForPrisma = globalThis as unknown as {
  prisma: PrismaClient | undefined;
};

if (process.pkg) {
  const basePath = process.cwd();
  process.env.DATABASE_URL = `file:${path.join(basePath, "devourer.db")}`;
  process.env.ASSETS_PATH = path.join(basePath, "assets");
} else {
  process.env.DATABASE_URL = `file:${path.join(
    __dirname,
    "../prisma/devourer.db"
  )}`;
  process.env.ASSETS_PATH = path.join(__dirname, "../assets");
}

export const prisma =
  globalForPrisma.prisma ??
  new PrismaClient({
    adapter: new PrismaBetterSQLite3({
      url: process.env.DATABASE_URL,
    }),
  });

if (process.env.NODE_ENV !== "production") globalForPrisma.prisma = prisma;
