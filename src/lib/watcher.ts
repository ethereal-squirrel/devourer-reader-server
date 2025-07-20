import chokidar from "chokidar";
import { prisma } from "../prisma";
import { processBook, processManga } from "./library";
import { Library } from "../types/types";

const processQueue: string[] = [];
const targetExtensions = [".cbz", ".zip", ".cbr", ".rar", ".pdf", ".epub"];
let libraryPaths: string[] = [];
let processing = false;

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
        log(`File ${path} has been added.`);
        processQueue.push(path);

        if (!processing) {
          processing = true;
          processWatching();
        }
      })
      .on("unlink", (path: string) => log(`File ${path} has been removed`));
  }, 5000);
};

export const stopWatching = async () => {
  await watcher.close().then(() => console.log("closed"));
};

export const processWatching = async () => {
  if (!processing) {
    return;
  }

  if (processQueue.length > 0) {
    processing = true;

    const path = processQueue.shift();

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
              await processBook(path, library);
            } else if (library.type === "manga") {
              await processManga(path, library);
            }
          }
        }
      }
    }

    processing = false;
    processWatching();
  }
};
