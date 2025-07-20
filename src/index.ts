import dotenv from "dotenv";
import express, { Express } from "express";
import { PrismaBetterSQLite3 } from "@prisma/adapter-better-sqlite3";
import { v4 as uuidv4 } from "uuid";
import bcrypt from "bcryptjs";
import cors from "cors";
import path from "path";
import fs from "fs";

import router from "./routes";
import { PrismaClient } from "../generated/prisma/client";
import { errorHandler } from "./middleware/errorHandler";
import { resetPassword } from "./lib/auth";
import { startWatcher } from "./lib/watcher";
import { searchMetadata } from "./lib/metadata";

declare global {
  namespace NodeJS {
    interface Process {
      pkg?: boolean;
    }
  }
}

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

dotenv.config();

const DATABASE_VERSION = 8;

export const app: Express = express();
const port = process.env.PORT || 9024;

app.use(express.json({ limit: "50mb" }));
app.use(express.urlencoded({ limit: "50mb", extended: true }));

app.use(
  cors({
    exposedHeaders: ["X-File-Size"],
  })
);

app.use("/assets", express.static(process.env.ASSETS_PATH!));
app.use(router);
app.use(errorHandler);

async function initializeDatabase() {
  const prisma = new PrismaClient({
    adapter: new PrismaBetterSQLite3({
      url: process.env.DATABASE_URL!,
    }),
  });

  try {
    await prisma.$connect();

    const tableExists = await prisma.$queryRaw`
      SELECT name 
      FROM sqlite_master 
      WHERE type='table' AND name='Config'
    `;

    if (!tableExists || (tableExists as any[]).length === 0) {
      console.log("Tables do not exist, running initial migration...");

      const migrationSql = fs.readFileSync(
        path.join(__dirname, `../migrations/1.sql`),
        "utf8"
      );
      const statements = migrationSql
        .split(";")
        .map((stmt) => stmt.trim())
        .filter((stmt) => stmt.length > 0);

      for (const statement of statements) {
        await prisma.$executeRawUnsafe(statement);
      }

      await prisma.config.create({
        data: {
          key: "allow_public",
          value: "0",
        },
      });

      await prisma.config.create({
        data: {
          key: "allow_register",
          value: "0",
        },
      });

      await prisma.config.create({
        data: {
          key: "api_google_books",
          value: "",
        },
      });

      const jwtSecret = uuidv4() + uuidv4();
      await prisma.config.create({
        data: {
          key: "jwt_secret",
          value: jwtSecret,
        },
      });

      const randomPassword = Math.random().toString(36).substring(2, 14);
      const randomApiKey = uuidv4();

      const hashedPassword = await bcrypt.hash(randomPassword, 12);
      const hashedApiKey = await bcrypt.hash(randomApiKey, 12);

      await prisma.user.create({
        data: {
          email: "admin",
          password: hashedPassword,
          api_key: hashedApiKey,
          roles: ["admin"],
          metadata: {
            settings: {
              book_pagemode: "single",
              book_font: "default",
              book_background: "#000000",
              manga_direction: "ltr",
              manga_pagemode: "single",
              manga_resizemode: "fit",
              manga_background: "#000000",
            },
          },
          created_at: new Date(),
        },
      });

      console.log("Initial migration executed successfully");
      console.log("--------------------------------");
      console.log(
        "Your initial account has been created with the following credentials:"
      );
      console.log("Username: admin");
      console.log(`Password: ${randomPassword}`);
      console.log(`API key: ${randomApiKey}`);
      console.log("--------------------------------");

      await prisma.config.create({
        data: {
          key: "migration_version",
          value: "1",
        },
      });
    }

    const config = await prisma.config.findUnique({
      where: {
        key: "migration_version",
      },
    });

    if (config?.value !== DATABASE_VERSION.toString()) {
      console.log("Database is out of date, running migrations...");

      for (
        let i = parseInt(config?.value ?? "0") + 1;
        i <= DATABASE_VERSION;
        i++
      ) {
        if (i <= parseInt(config?.value ?? "0")) {
          continue;
        }

        console.log(
          `[Migration] Running migration ${i} (${DATABASE_VERSION}) (${parseInt(
            config?.value ?? "0"
          )})`
        );

        try {
          const migrationSql = fs.readFileSync(
            path.join(__dirname, `../migrations/${i}.sql`),
            "utf8"
          );
          const statements = migrationSql
            .split(";")
            .map((stmt) => stmt.trim())
            .filter((stmt) => stmt.length > 0);

          for (const statement of statements) {
            await prisma.$executeRawUnsafe(statement);
          }
        } catch (error) {
          console.error(`Failed to run migration ${i}:`, error);
          throw error;
        }
      }

      await prisma.config.update({
        where: {
          key: "migration_version",
        },
        data: {
          value: DATABASE_VERSION.toString(),
        },
      });

      console.log("Database migrations executed successfully");
    }

    console.log("Database initialized successfully");
  } catch (error) {
    console.error("Failed to initialize database:", error);
    throw error;
  } finally {
    await prisma.$disconnect();
  }
}

async function handleCommand(command: string, args: string[]) {
  switch (command) {
    case "create-library":
      {
        const libraryName = args[0];
        const libraryPath = args[1];
        const libraryType = args[2];
        const libraryProvider = args[3];
        const libraryApiKey = args[4];

        if (
          !libraryName ||
          !libraryPath ||
          libraryName.length === 0 ||
          libraryPath.length === 0 ||
          libraryType.length === 0 ||
          libraryProvider.length === 0
        ) {
          console.error("Invalid library name, path, type or provider");
          process.exit(1);
        }

        console.log(
          `[Command] Creating library ${libraryName} at ${libraryPath}`
        );

        await fetch(`http://localhost:${port}/libraries`, {
          method: "POST",
          body: JSON.stringify({
            name: libraryName,
            path: libraryPath,
            type: libraryType,
            metadata: { provider: libraryProvider, api_key: libraryApiKey },
          }),
        });
      }
      break;
    case "reset-password":
      {
        const email = args[0];
        const password = args[1];

        if (!email || !password) {
          console.error("Invalid email or password");
          process.exit(1);
        }

        const res = await resetPassword(email, password);
        console.log(res);
      }
      break;
    case "scan-library":
      {
        const libraryId = parseInt(args[0]);
        if (isNaN(libraryId)) {
          console.error("Invalid library ID");
          process.exit(1);
        }

        console.log(`[Command] Scanning library ${libraryId}`);

        await fetch(`http://localhost:${port}/library/${libraryId}/scan`, {
          method: "POST",
        });
      }
      break;
    case "scan-status":
      {
        const libraryId = parseInt(args[0]);

        if (isNaN(libraryId)) {
          console.error("Invalid library ID");
          process.exit(1);
        }

        console.log(
          `[Command] Retrieving scan status for library ${libraryId}`
        );

        const res = await fetch(
          `http://localhost:${port}/library/${libraryId}/scan`,
          {
            method: "GET",
          }
        );

        const data = await res.json();
        console.log(data);
      }
      break;
    default:
      console.error(`Unknown command: ${command}`);
      process.exit(1);
  }
}

async function startApp() {
  try {
    await initializeDatabase();
    await startWatcher();

    //await searchMetadata("jikan", "title", "Sword Art Online");
    //await searchMetadata("googlebooks", "title", "Sword Art Online");
    //await searchMetadata("openlibrary", "title", "return of the king");
    //await searchMetadata("comicvine", "title", "Spider-Man");

    app.listen(port, () => {
      console.log(`[Server] Devourer is running on port ${port}`);
      console.log(`[Server] Database: ${process.env.DATABASE_URL}`);
      console.log(`[Server] Assets: ${process.env.ASSETS_PATH}`);

      if (process.pkg) {
        console.log(`[Server] Running in packaged mode`);
      }
    });
  } catch (error) {
    console.error("Failed to start app:", error);
    process.exit(1);
  }
}

const args = process.argv.slice(2);

if (args.length === 0) {
  startApp();
} else {
  const [command, ...commandArgs] = args;
  handleCommand(command, commandArgs).catch((error) => {
    console.error("Command failed:", error);
    process.exit(1);
  });
}
