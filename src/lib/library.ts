import fs from "fs";
import * as webp from "webp-wasm";
import { Jimp } from "jimp";
import path from "path";

import { getAllFiles } from "./file";
import { isValidBook, scanEpub } from "./book/bookScanner";
import { createBookSeriesPayload } from "./book/bookSeries";
import {
  extractChapterAndVolume,
  processFileInline,
} from "./manga/mangaScanner";
import { createSeriesPayload } from "./manga/mangaSeries";
import { prisma } from "../prisma";
import { ApiError } from "../types/api";
import { Library } from "../types/types";

export async function convertImageDataToWebP(
  imageBuffer: Buffer,
  outputPath: string,
  maxWidth: number = 600,
  quality: number = 85
): Promise<void> {
  try {
    if (
      imageBuffer.slice(0, 4).toString() === "RIFF" &&
      imageBuffer.slice(8, 12).toString() === "WEBP"
    ) {
      console.log("[WebP] Image is already in WebP format");
      fs.writeFileSync(outputPath, imageBuffer);
      return;
    }

    const image = await Jimp.read(imageBuffer);

    if (image.bitmap.width > maxWidth) {
      image.resize({ w: maxWidth });
    }

    const imageData = {
      data: new Uint8ClampedArray(image.bitmap.data),
      width: image.bitmap.width,
      height: image.bitmap.height,
    };

    const webpBuffer = await webp.encode(imageData, { quality });

    if (!webpBuffer) {
      throw new Error("Failed to encode WebP");
    }

    fs.writeFileSync(outputPath, Buffer.from(webpBuffer));

    console.log(`[WebP] Converted and saved: ${outputPath}`);
  } catch (error) {
    console.error(`[WebP] Error converting image: ${error}`);
    throw error;
  }
}

async function downloadAndConvertToWebP(
  imageUrl: string,
  outputPath: string,
  maxWidth: number = 600,
  quality: number = 85
): Promise<void> {
  try {
    const response = await fetch(imageUrl);
    if (!response.ok) {
      throw new Error(`Failed to download image: ${response.statusText}`);
    }

    const imageBuffer = await response.arrayBuffer();
    await convertImageDataToWebP(
      Buffer.from(imageBuffer),
      outputPath,
      maxWidth,
      quality
    );
  } catch (error) {
    console.error(`[WebP] Error downloading and converting image: ${error}`);
    throw error;
  }
}

export interface ScanProgress {
  series: string;
  libraryType: string;
  status: "scanning" | "complete" | "error";
  progress?: {
    current: number;
    total: number;
  };
  error?: string;
}

export interface ScanStatus {
  inProgress: boolean;
  libraryType: string;
  series: ScanProgress[];
  startTime: Date;
  completedSeries: number;
  totalSeries: number;
}

export interface ScanLibraryResponse {
  status: boolean;
  message?: string;
  inProgress?: boolean;
  remaining?: string[];
}

export interface GetScanStatusResponse {
  status: boolean;
  message?: string;
  inProgress?: boolean;
  libraryType: string;
  progress?: {
    completed: number;
    total: number;
    series: ScanProgress[];
  };
  startTime?: Date;
  remaining?: string[];
}

const scanStatusMap: Record<number, ScanStatus> = {};

export const getScanStatusMap = () => scanStatusMap;
export const setScanStatus = (id: number, status: ScanStatus) => {
  scanStatusMap[id] = status;
};
export const clearScanStatus = (id: number) => {
  delete scanStatusMap[id];
};

export const createLibrary = async (payload: Library) => {
  if (!payload.path || !payload.name || !payload.type || !payload.metadata) {
    throw new ApiError(400, "All fields are required");
  }

  if (payload.type !== "book" && payload.type !== "manga") {
    throw new ApiError(400, "Invalid library type");
  }

  const existingLibrary = await prisma.library.findFirst({
    where: {
      path: payload.path,
    },
  });

  if (existingLibrary) {
    throw new ApiError(400, "Library at this path already exists");
  }

  const library = await prisma.library.create({
    data: payload,
  });

  scanLibrary(library.id);

  return library;
};

export const getLibraries = async () => {
  const librariesData: any[] = [];
  const libraries = await prisma.library.findMany();

  for (const library of libraries) {
    if (library.type === "book") {
      const series = await prisma.bookFile.findMany({
        select: {
          id: true,
        },
        take: 3,
        where: { library_id: library.id },
      });

      librariesData.push({
        ...library,
        series,
        seriesCount: await prisma.bookFile.count({
          where: { library_id: library.id },
        }),
      });
    } else {
      const series = await prisma.mangaSeries.findMany({
        select: {
          id: true,
          title: true,
        },
        take: 3,
        where: { library_id: library.id },
      });

      librariesData.push({
        ...library,
        series,
        seriesCount: await prisma.mangaSeries.count({
          where: { library_id: library.id },
        }),
      });
    }
  }

  return librariesData;
};

export const getLibrary = async (id: string, userId: number) => {
  const libraryData = await prisma.library.findFirst({
    where: {
      id: parseInt(id),
    },
  });

  if (!libraryData) {
    throw new ApiError(404, "Library not found");
  }

  const library = {
    id: libraryData.id,
    name: libraryData.name,
    path: libraryData.path,
    type: libraryData.type,
    series: [] as any,
    collections: [] as any,
  };

  if (libraryData.type === "book") {
    library.series = await prisma.bookFile.findMany({
      where: {
        library_id: libraryData.id,
      },
    });
  } else {
    library.series = await prisma.mangaSeries.findMany({
      where: {
        library_id: libraryData.id,
      },
    });
  }

  const collections = await prisma.collection.findMany({
    where: {
      library_id: libraryData.id,
      OR: [
        {
          user_id: userId,
        },
        {
          user_id: 0,
        },
      ],
    },
  });

  const userRatings = await prisma.userRating.findMany({
    where: {
      user_id: userId,
      file_type: libraryData.type,
      file_id: { in: library.series.map((s: any) => s.id) },
    },
  });

  for (const series of library.series) {
    const userRating = userRatings.find((r: any) => r.file_id === series.id);

    if (userRating) {
      series.rating = userRating.rating;
    } else {
      series.rating = null;
    }
  }

  library.collections = collections;

  return library;
};

export const updateLibrary = async (id: number, data: any) => {
  const library = await prisma.library.findFirst({
    where: {
      id,
    },
  });

  if (!library) {
    throw new ApiError(404, "Library not found");
  }

  await prisma.library.update({
    where: { id },
    data,
  });

  return {
    status: true,
    message: "Library updated",
  };
};

export const deleteLibrary = async (id: number) => {
  const library = await prisma.library.findFirst({
    where: {
      id,
    },
  });

  if (!library) {
    throw new ApiError(404, "Library not found");
  }

  let folderIds = [] as number[];

  if (library.type === "book") {
    const books = await prisma.bookFile.findMany({
      select: {
        id: true,
      },
      where: { library_id: id },
    });

    folderIds = books.map((b) => b.id);

    await prisma.bookFile.deleteMany({
      where: { library_id: id },
    });
  } else {
    const series = await prisma.mangaSeries.findMany({
      select: {
        id: true,
      },
      where: { library_id: id },
    });

    const seriesIds = series.map((s) => s.id);
    folderIds = seriesIds;

    await prisma.mangaFile.deleteMany({
      where: { series_id: { in: seriesIds } },
    });

    await prisma.mangaSeries.deleteMany({
      where: { library_id: id },
    });
  }

  await prisma.collection.deleteMany({
    where: { library_id: id },
  });

  await prisma.library.delete({
    where: { id },
  });

  try {
    for (const folderId of folderIds) {
      fs.rmSync(
        path.join(
          library.path,
          ".devourer",
          library.type === "book" ? "files" : "series",
          folderId.toString()
        ),
        {
          recursive: true,
        }
      );
    }
  } catch (error) {
    console.error(`[Library] Error deleting library:`, error);
  }

  return {
    status: true,
    message: "Library deleted",
  };
};

export const scanLibrary = async (id: number) => {
  const library = await prisma.library.findFirst({
    where: {
      id,
    },
  });

  if (!library) {
    throw new ApiError(404, "Library not found");
  }

  if (getScanStatusMap()[library.id]?.inProgress) {
    return {
      status: false,
      message: "Scan already in progress",
    };
  }

  const topLevelFolders = fs
    .readdirSync(library.path)
    .filter((folder: string) => folder !== ".devourer");

  setScanStatus(library.id, {
    inProgress: true,
    series: topLevelFolders.map((folder) => ({
      series: folder,
      status: "scanning",
      libraryType: library.type,
    })),
    libraryType: library.type,
    startTime: new Date(),
    completedSeries: 0,
    totalSeries: topLevelFolders.length,
  });

  if (library.type === "book") {
    scanBookLibrary(library as Library, topLevelFolders);
  } else {
    scanMangaLibrary(library as Library, topLevelFolders);
  }

  return {
    status: true,
    message: "Library scan started",
    remaining: topLevelFolders,
  };
};

export const getScanStatus = async (
  id: number
): Promise<GetScanStatusResponse> => {
  const status = getScanStatusMap()[id];

  if (!status) {
    return {
      status: false,
      message: "No scan in progress",
      libraryType: "",
      remaining: [],
    };
  }

  const remainingSeries = status.series
    .filter((s) => s.status === "scanning")
    .map((s) => s.series);

  return {
    status: true,
    inProgress: status.inProgress,
    libraryType: status.libraryType,
    progress: {
      completed: status.completedSeries,
      total: status.totalSeries,
      series: status.series,
    },
    startTime: status.startTime,
    remaining: remainingSeries,
  };
};

const scanBookLibrary = async (library: Library, folders: string[]) => {
  let collections: any = {};
  let collectionId = 1;
  let folderIndex = 0;

  console.log(
    `[Library] Starting scan of ${folders.length} book entities (processing one at a time due to API limits)`
  );

  const processSeries = async (
    folder: string,
    isFolder: boolean,
    collectionId?: number
  ) => {
    const startTime = Date.now();

    try {
      console.log(
        `[Library] Processing entity ${folderIndex} of ${folders.length}: ${folder}`
      );

      const seriesIndex = getScanStatusMap()[library.id!].series.findIndex(
        (s) => s.series === folder
      );
      if (seriesIndex !== -1) {
        getScanStatusMap()[library.id!].series[seriesIndex].status = "scanning";
      }

      let files = [];

      if (isFolder) {
        files = getAllFiles(path.join(library.path, folder));
      } else {
        files = [path.join(library.path, folder)];
      }

      for (const file of files) {
        let existingFile = await prisma.bookFile.findFirst({
          where: {
            path: file,
          },
        });

        if (!existingFile) {
          const bookName = path.basename(file);
          const cleanBookName = bookName
            .replace(/[\[\(\<].*?[\]\)\>]/g, "")
            .trim();
          const cleanBookNameWithoutExt = cleanBookName.replace(
            /\.[^/.]+$/,
            ""
          );

          let metadata = null;
          let series = null;

          if (file.endsWith(".epub")) {
            metadata = (await scanEpub(file)) as any;
          }

          if (metadata) {
            series = await createBookSeriesPayload(
              library.id!,
              metadata.title,
              path.join(library.path, folder),
              metadata.isbn ?? null,
              true
            );
          } else {
            series = await createBookSeriesPayload(
              library.id!,
              cleanBookNameWithoutExt,
              path.join(library.path, folder),
              null,
              true
            );
          }

          if (!series.metadata) {
            series.metadata = {};
          }

          if (!series.metadata.title) {
            series.metadata.title = cleanBookNameWithoutExt;
          }

          if (metadata) {
            series.metadata.epub = metadata;
          }

          const createdFile = await prisma.bookFile.create({
            data: {
              title:
                series.metadata.epub && series.metadata.epub.title
                  ? series.metadata.epub.title
                  : series.metadata.title,
              path: file,
              file_name: path.basename(file),
              file_format: path.extname(file).slice(1),
              total_pages: series.metadata.pageCount || 0,
              current_page: "0",
              is_read: false,
              metadata: {
                ...series.metadata,
                epub: series.metadata.epub
                  ? { ...series.metadata.epub, cover: null }
                  : null,
              },
              library_id: library.id,
              tags: [],
              formats: [
                {
                  format: path.extname(file).slice(1),
                  name: path.basename(file),
                  path: file,
                },
              ],
            },
          });

          if (isFolder && collectionId) {
            if (!collections[collectionId]) {
              collections[collectionId] = {
                folder: folder,
                contents: [createdFile.id],
              };
            } else {
              collections[collectionId].contents.push(createdFile.id);
            }
          }

          const previewDir = path.join(
            library.path,
            ".devourer",
            "files",
            createdFile.id.toString()
          );

          try {
            fs.mkdirSync(previewDir, { recursive: true });
          } catch (error) {
            console.error(`[Library] Error creating preview directory:`, error);
          }

          if (file.endsWith(".epub")) {
            metadata = (await scanEpub(file)) as any;
          }

          let hasCover = false;

          if (
            file.endsWith(".epub") &&
            metadata &&
            metadata.cover &&
            metadata.coverMimeType === "image/jpeg"
          ) {
            await convertImageDataToWebP(
              metadata.cover,
              path.join(previewDir, "cover.webp")
            );

            hasCover = true;
          }

          if (file.endsWith(".pdf")) {
            try {
              // @TODO: Implement.
              hasCover = false;
            } catch (error) {
              console.error(`[Library] Error processing PDF:`, error);
            }
          }

          if (!hasCover) {
            if (
              series.metadata &&
              series.metadata.cover &&
              series.metadata.cover.length > 10
            ) {
              await downloadAndConvertToWebP(
                series.metadata.cover,
                path.join(
                  library.path,
                  ".devourer",
                  "files",
                  createdFile.id.toString(),
                  "cover.webp"
                )
              );

              hasCover = true;
            } else if (
              series.metadata &&
              series.metadata.isbn_13 &&
              series.metadata.isbn_13.length > 0
            ) {
              const coverUrl = `https://covers.openlibrary.org/b/isbn/${series.metadata.isbn_13}-L.jpg`;
              await downloadAndConvertToWebP(
                coverUrl,
                path.join(
                  library.path,
                  ".devourer",
                  "files",
                  createdFile.id.toString(),
                  "cover.webp"
                )
              );

              hasCover = true;
            }
          }

          console.log(
            `[Library] Created file: ${createdFile.title} | ${createdFile.path}`
          );
        }
      }

      updateSeriesComplete(library.id!, folder);
    } catch (error) {
      console.error(`[Library] Error scanning series ${folder}:`, error);
      updateError(
        library.id!,
        folder,
        error instanceof Error ? error.message : "Unknown error"
      );
    }

    const endTime = Date.now();
    const duration = (endTime - startTime) / 1000;
    console.log(`[Library] Series ${folder} completed in ${duration} seconds`);
  };

  for (const folder of folders) {
    folderIndex++;

    const stats = await fs.promises.stat(path.join(library.path, folder));

    if (!stats.isDirectory()) {
      const isValid = isValidBook(path.join(library.path, folder));

      if (!isValid) {
        console.log(`[Library] Skipping ${folder} as it is not a valid book`);
        continue;
      }
      await processSeries(folder, false);

      continue;
    } else {
      const contents = getAllFiles(path.join(library.path, folder));

      if (contents.length > 1) {
        collections[collectionId] = {
          folder: folder,
          contents: [],
        };

        await processSeries(folder, true, collectionId);

        collectionId++;
      } else {
        await processSeries(folder, true);
      }
    }
  }

  for (const collectionId of Object.keys(collections)) {
    const c = collections[collectionId];

    const existingCollection = await prisma.collection.findFirst({
      where: {
        library_id: library.id,
        name: c.folder,
      },
    });

    if (!existingCollection) {
      await prisma.collection.create({
        data: {
          library_id: library.id!,
          name: c.folder,
          series: c.contents,
          user_id: 0,
        },
      });
    } else {
      const existingSeries = existingCollection.series as string[];
      const uniqueSeries = [...new Set([...existingSeries, ...c.contents])];
      c.contents = uniqueSeries;

      await prisma.collection.update({
        where: {
          id: existingCollection.id,
        },
        data: {
          series: c.contents,
        },
      });
    }
  }

  if (library.type === "book") {
    const series = await prisma.bookFile.findMany();

    for (const s of series) {
      if (!fs.existsSync(s.path)) {
        console.log(
          `[Library] Book ${s.title} folder has been removed, deleting...`
        );

        await prisma.bookFile.deleteMany({
          where: { id: s.id },
        });

        try {
          fs.rmSync(
            path.join(library.path, ".devourer", "files", s.id.toString()),
            {
              recursive: true,
            }
          );
        } catch (error) {
          console.error(
            `[Library] Error deleting .devourer folder for book ${s.title}:`,
            error
          );
        }
      }
    }
  } else {
    const series = await prisma.mangaSeries.findMany();

    for (const s of series) {
      if (!fs.existsSync(s.path)) {
        console.log(
          `[Library] Manga series ${s.title} folder has been removed, deleting...`
        );

        await prisma.mangaFile.deleteMany({
          where: { series_id: s.id },
        });

        await prisma.mangaSeries.delete({
          where: { id: s.id },
        });

        try {
          fs.rmSync(
            path.join(library.path, ".devourer", "series", s.id.toString()),
            {
              recursive: true,
            }
          );
        } catch (error) {
          console.error(
            `[Library] Error deleting .devourer folder for manga series: ${s.title}:`,
            error
          );
        }
      } else {
        const existingFiles = await prisma.mangaFile.findMany({
          where: { series_id: s.id },
        });

        for (const file of existingFiles) {
          if (!fs.existsSync(file.path)) {
            await prisma.mangaFile.delete({
              where: { id: file.id },
            });

            try {
              const previewPath = path.join(
                library.path,
                ".devourer",
                "series",
                s.id.toString(),
                "previews",
                `${file.path}.jpg`
              );

              fs.rmSync(previewPath);
            } catch (error) {
              console.error(
                `[Library] Error deleting preview image for: ${file.file_name}:`,
                error
              );
            }
          }
        }
      }
    }
  }

  console.log("[Library] Scan completed");
  getScanStatusMap()[library.id!].inProgress = false;
};

const scanMangaLibrary = async (library: Library, folders: string[]) => {
  let folderIndex = 0;

  console.log(
    `[Library] Starting scan of ${folders.length} manga series (processing one at a time due to API limits)`
  );

  const processSeries = async (folder: string) => {
    const startTime = Date.now();

    try {
      console.log(
        `[Library] Processing folder ${folderIndex} of ${folders.length}: ${folder}`
      );

      const seriesIndex = getScanStatusMap()[library.id!].series.findIndex(
        (s) => s.series === folder
      );
      if (seriesIndex !== -1) {
        getScanStatusMap()[library.id!].series[seriesIndex].status = "scanning";
      }

      let existingSeries = await prisma.mangaSeries.findFirst({
        where: {
          library_id: library.id,
          title: folder,
        },
      });

      if (!existingSeries) {
        let series = await createSeriesPayload(
          library.metadata?.provider ?? "myanimelist",
          library.id!,
          folder,
          path.join(library.path, folder),
          null,
          true,
          library.metadata?.apiKey
        );

        console.log(`[Library] Creating new series: ${series.title}`);
        updateProgress(library.id!, folder, "creating_series");

        existingSeries = await prisma.mangaSeries.create({
          data: { ...series, manga_data: series.manga_data },
        });

        const seriesDir = path.join(
          library.path,
          ".devourer",
          "series",
          existingSeries.id.toString(),
          "previews"
        );

        fs.mkdirSync(seriesDir, { recursive: true });

        if (series.manga_data) {
          if (series.manga_data.coverImage) {
            await downloadAndConvertToWebP(
              series.manga_data.coverImage,
              path.join(
                library.path,
                ".devourer",
                "series",
                existingSeries.id.toString(),
                "cover.webp"
              )
            );
          }

          await new Promise((resolve) => setTimeout(resolve, 1000));
        }
      }

      if (existingSeries) {
        updateProgress(library.id!, folder, "scanning_files");

        const seriesPath = path.join(library.path, folder);
        let allFiles = getAllFiles(seriesPath);

        if (!allFiles) {
          throw new Error("Failed to read directory");
        }

        const filteredFiles = allFiles.filter((file: string) =>
          /\.(zip|cbz|rar|cbr)$/i.test(file)
        );

        if (filteredFiles.length === 0) {
          console.log(
            `[Library] Skipping folder-based series: ${existingSeries.title}`
          );
          updateSeriesComplete(library.id!, folder);
          return;
        }

        const existingFiles = await prisma.mangaFile.findMany({
          where: { series_id: existingSeries.id },
          select: { id: true, path: true },
        });

        const filesToDelete = existingFiles.filter(
          (file: any) => !fs.existsSync(file.path)
        );
        const existingFilePaths = new Set(
          existingFiles
            .filter((file: any) => fs.existsSync(file.path))
            .map((f: any) => f.path)
        );

        const previewDir = path.join(
          library.path,
          ".devourer",
          "series",
          existingSeries.id.toString(),
          "previews"
        );
        fs.mkdirSync(previewDir, { recursive: true });

        const filesToCreate = [];
        console.log(
          `[Library] Processing ${filteredFiles.length} files for series: ${existingSeries.title}`
        );
        updateProgress(
          library.id!,
          folder,
          "processing_files",
          undefined,
          filteredFiles.length
        );

        for (const [index, file] of filteredFiles.entries()) {
          if (existingFilePaths.has(file)) continue;

          const startFile = Date.now();

          console.log(`[Library] Processing new file: ${path.basename(file)}`);
          const { volume, chapter } = extractChapterAndVolume(file);

          try {
            const response = await processFileInline(
              file,
              path.join(previewDir, `${path.basename(file)}.jpg`)
            );

            filesToCreate.push({
              path: file,
              file_name: path.basename(file),
              file_format: path.extname(file).slice(1),
              volume: volume ?? 0,
              chapter: chapter ?? 0,
              total_pages: response?.pageCount ?? 0,
              current_page: 0,
              is_read: false,
              series_id: existingSeries.id,
              metadata: {},
            });
          } catch (error) {
            console.error(
              `[Library] Error processing file ${path.basename(file)}:`,
              error
            );
          }

          updateProgress(
            library.id!,
            folder,
            "file_processed",
            index + 1,
            filteredFiles.length
          );

          const endFile = Date.now();
          const fileDuration = (endFile - startFile) / 1000;
          console.log(
            `[Library] File ${path.basename(
              file
            )} processed in ${fileDuration} seconds`
          );
        }

        if (filesToDelete.length > 0 || filesToCreate.length > 0) {
          await prisma.$transaction([
            ...(filesToDelete.length > 0
              ? [
                  prisma.mangaFile.deleteMany({
                    where: {
                      id: {
                        in: filesToDelete.map((f: any) => f.id),
                      },
                    },
                  }),
                ]
              : []),
            ...(filesToCreate.length > 0
              ? filesToCreate.map((data) => prisma.mangaFile.create({ data }))
              : []),
          ]);

          if (filesToDelete.length > 0) {
            console.log(
              `[Library] Removed ${filesToDelete.length} deleted files`
            );
            updateProgress(
              library.id!,
              folder,
              "files_removed",
              filesToDelete.length
            );
          }
        }
      }

      updateSeriesComplete(library.id!, folder);
    } catch (error) {
      console.error(`[Library] Error scanning series ${folder}:`, error);
      updateError(
        library.id!,
        folder,
        error instanceof Error ? error.message : "Unknown error"
      );
    }

    const endTime = Date.now();
    const duration = (endTime - startTime) / 1000;
    console.log(`[Library] Series ${folder} completed in ${duration} seconds`);
  };

  for (const folder of folders) {
    folderIndex++;
    await processSeries(folder);
  }

  const series = await prisma.mangaSeries.findMany();

  for (const s of series) {
    if (!fs.existsSync(s.path)) {
      console.log(`[Library] Series ${s.title} has no files, deleting`);

      await prisma.mangaFile.deleteMany({
        where: { series_id: s.id },
      });

      await prisma.mangaSeries.delete({
        where: { id: s.id },
      });
    }
  }

  console.log("[Library] Scan completed");
  getScanStatusMap()[library.id!].inProgress = false;
};

const updateProgress = (
  libraryId: number,
  series: string,
  status: string,
  current?: number,
  total?: number,
  count?: number
) => {
  const idx = getScanStatusMap()[libraryId].series.findIndex(
    (s) => s.series === series
  );
  if (idx !== -1) {
    if (current !== undefined && total !== undefined) {
      getScanStatusMap()[libraryId].series[idx].progress = { current, total };
    }
  }
};

const updateError = (libraryId: number, series: string, error: string) => {
  const idx = getScanStatusMap()[libraryId].series.findIndex(
    (s) => s.series === series
  );
  if (idx !== -1) {
    getScanStatusMap()[libraryId].series[idx].status = "error";
    getScanStatusMap()[libraryId].series[idx].error = error;
  }
};

const updateSeriesComplete = (libraryId: number, series: string) => {
  const idx = getScanStatusMap()[libraryId].series.findIndex(
    (s) => s.series === series
  );
  if (idx !== -1) {
    getScanStatusMap()[libraryId].series[idx].status = "complete";
  }
  getScanStatusMap()[libraryId].completedSeries++;
};

export const getRecentlyRead = async (userId: number) => {
  const recentlyRead = await prisma.recentlyRead.findMany({
    where: {
      user_id: userId,
    },
    orderBy: {
      id: "desc",
    },
    take: 10,
  });

  return recentlyRead;
};

export const updateRecentlyRead = async (
  libraryId: number,
  fileId: number,
  page: number,
  userId: number
) => {
  const library = await prisma.library.findUnique({
    where: { id: libraryId },
  });

  if (!library) {
    return false;
  }

  let file: any = null;

  if (library.type === "book") {
    file = await prisma.bookFile.findUnique({
      where: { id: fileId },
    });
  } else {
    file = await prisma.mangaFile.findUnique({
      where: { id: fileId },
    });
  }

  if (!file) {
    return false;
  }

  await prisma.recentlyRead.deleteMany({
    where: { library_id: libraryId, file_id: fileId, user_id: userId },
  });

  await prisma.recentlyRead.create({
    data: {
      is_local: false,
      library_id: libraryId,
      series_id: file.series_id || 0,
      file_id: fileId,
      current_page: page.toString(),
      total_pages: file.total_pages || 0,
      volume: file.volume || 0,
      chapter: file.chapter || 0,
      user_id: userId,
    },
  });

  const existingRead = await prisma.recentlyRead.findMany({
    where: { user_id: userId },
    select: {
      id: true,
    },
    orderBy: {
      id: "desc",
    },
    take: 10,
  });

  await prisma.recentlyRead.deleteMany({
    where: {
      id: { notIn: existingRead.map((read) => read.id) },
      user_id: userId,
    },
  });
};
