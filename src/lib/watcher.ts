import chokidar from "chokidar";

import { deleteBook, deleteManga, processBook, processManga } from "./library";
import { prisma } from "../prisma";
import { Library } from "../types/types";

const processQueue: string[] = [];
const processDeleteQueue: string[] = [];
const targetExtensions = [".cbz", ".zip", ".cbr", ".rar", ".pdf", ".epub"];
let libraryPaths: string[] = [];
let processing = false;
let processingDelete = false;

let watcher: any = null;

export const startWatcher = async () => {
  console.log("[Watcher] Starting watcher.");

  const libraries = await prisma.library.findMany();
  libraryPaths = libraries.map((library) => library.path);

  console.log(
    "[Watcher] Libraries:",
    libraries.map((library) => library.path)
  );

  watcher = chokidar.watch(
    libraries.map((library) => library.path),
    {
      ignored: (file, _stats) => {
        if (!_stats?.isFile()) return false;
        return !targetExtensions.some((ext) =>
          file.toLowerCase().endsWith(ext.toLowerCase())
        );
      },
      persistent: true,
      awaitWriteFinish: true,
    }
  );

  const log = console.log.bind(console);

  setTimeout(() => {
    watcher
      .on("add", (path: string) => {
        log(`[Watcher] File ${path} has been added.`);
        processQueue.push(path);

        if (!processing) {
          processing = true;
          processWatching();
        }
      })
      .on("unlink", (path: string) => {
        log(`[Watcher] File ${path} has been removed`);
        processDeleteQueue.push(path);

        if (!processingDelete) {
          processingDelete = true;
          processDeleteWatching();
        }
      });
  }, 5000);
};

export const stopWatching = async () => {
  await watcher.close().then(() => console.log("[Watcher] Stopped."));
};

export const processWatching = async () => {
  if (!processing) {
    return;
  }

  if (processQueue.length > 0) {
    processing = true;

    const filePath = processQueue.shift();

    if (filePath) {
      for (const libraryPath of libraryPaths) {
        if (filePath.startsWith(libraryPath)) {
          const library = (await prisma.library.findFirst({
            where: {
              path: libraryPath,
            },
          })) as Library;

          console.log(
            `[Watcher] Processing file ${filePath} in library ${libraryPath}`
          );

          if (library) {
            if (library.type === "book") {
              await processBook(filePath, library);
            } else if (library.type === "manga") {
              const relativePath = filePath.substring(libraryPath.length);
              const pathSeparator = relativePath.includes("\\") ? "\\" : "/";
              const cleanRelativePath = relativePath.startsWith(pathSeparator)
                ? relativePath.substring(1)
                : relativePath;
              const firstSubfolderEnd =
                cleanRelativePath.indexOf(pathSeparator);

              if (firstSubfolderEnd > 0) {
                const firstSubfolder = cleanRelativePath.substring(
                  0,
                  firstSubfolderEnd
                );

                await processManga(firstSubfolder, library);
              }
            }
          }
        }
      }
    }

    processing = false;
    processWatching();
  }
};

export const processDeleteWatching = async () => {
  if (!processingDelete) {
    return;
  }

  if (processDeleteQueue.length > 0) {
    processingDelete = true;

    const path = processDeleteQueue.shift();

    if (path) {
      for (const libraryPath of libraryPaths) {
        if (path.startsWith(libraryPath)) {
          const library = (await prisma.library.findFirst({
            where: {
              path: libraryPath,
            },
          })) as Library;

          console.log(
            `[Watcher] Processing file ${path} in library ${libraryPath}`
          );

          if (library) {
            if (library.type === "book") {
              await deleteBook(path);
            } else if (library.type === "manga") {
              await deleteManga(path);
            }
          }
        }
      }
    }

    processingDelete = false;
    processDeleteWatching();
  }
};
