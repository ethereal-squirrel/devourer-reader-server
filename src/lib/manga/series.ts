import path from "path";
import fs from "fs/promises";

import { extractChapterAndVolume, processFileInline } from "./mangaScanner";
import { downloadAndConvertToWebP } from "../library";
import { searchMetadata as search } from "../metadata";
import { prisma } from "../../prisma";
import { ApiError } from "../../types/api";

export const getSeries = async (
  libraryId: number,
  seriesId: number,
  userId: number
) => {
  let series: any = await prisma.mangaSeries.findFirst({
    where: { library_id: libraryId, id: seriesId },
  });

  if (!series) {
    throw new ApiError(404, "Series not found");
  }

  const userRating = await prisma.userRating.findFirst({
    where: { user_id: userId, file_type: "manga", file_id: seriesId },
  });

  if (userRating) {
    series.rating = userRating.rating;
  } else {
    series.rating = null;
  }

  const userTags = await prisma.userTag.findMany({
    where: { user_id: userId, file_type: "manga", file_id: seriesId },
  });

  if (userTags.length > 0) {
    series.tags = userTags.map((tag) => tag.tag);
  } else {
    series.tags = [];
  }

  return series;
};

export const createMangaSeries = async (series: any) => {
  if (!series.title || series.title.length === 0) {
    throw new ApiError(400, "Title is required");
  }

  if (!series.path || series.path.length === 0) {
    throw new ApiError(400, "Path is required");
  }

  const library = (await prisma.library.findFirst({
    where: { id: series.library_id },
  })) as any;

  if (!library) {
    throw new ApiError(404, "Library not found");
  }

  const metadata = await search(
    library.metadata?.provider,
    "title",
    series.title,
    library.metadata.apiKey
  );

  const seriesPath = path.join(library.path, series.path);

  const existingSeries = await prisma.mangaSeries.findFirst({
    where: { library_id: library.id, path: seriesPath },
  });

  if (existingSeries) {
    throw new ApiError(400, "Series already exists at this path");
  }

  const fileDir = path.join(library.path, series.path);

  try {
    await fs.mkdir(fileDir, { recursive: true });
  } catch (error) {
    console.error(`[Manga] Error creating series directory: ${error}`);
    throw new ApiError(500, "Failed to create series directory");
  }

  const newSeries = await prisma.mangaSeries.create({
    data: {
      title: series.title,
      path: seriesPath,
      cover: "",
      manga_data: metadata,
      library_id: library.id,
    },
  });

  const previewDir = path.join(
    library.path,
    ".devourer",
    "series",
    newSeries.id.toString(),
    "previews"
  );

  try {
    await fs.mkdir(previewDir, { recursive: true });
  } catch (error) {
    console.error(`[Manga] Error creating series preview directory: ${error}`);

    await prisma.mangaSeries.delete({
      where: { id: newSeries.id },
    });

    throw new ApiError(500, "Failed to create series preview directory");
  }

  if (metadata.coverImage) {
    try {
      await downloadAndConvertToWebP(
        metadata.coverImage,
        path.join(
          library.path,
          ".devourer",
          "series",
          newSeries.id.toString(),
          "cover.webp"
        )
      );
    } catch (error) {
      console.error(`[WebP] Error downloading and converting image: ${error}`);
    }
  }

  return newSeries;
};

export const uploadMangaFile = async (
  fileData: any,
  libraryId: number,
  seriesId: number
) => {
  const validExtensions = ["zip", "cbz", "rar", "cbr"];

  if (!validExtensions.includes(fileData.originalname.split(".").pop()!)) {
    throw new ApiError(400, "Invalid file extension");
  }

  const library = await prisma.library.findFirst({
    where: { id: libraryId },
  });

  if (!library) {
    throw new ApiError(404, "Library not found");
  }

  const series = await prisma.mangaSeries.findFirst({
    where: { library_id: libraryId, id: seriesId },
  });

  if (!series) {
    throw new ApiError(404, "Series not found");
  }

  const filePath = path.join(series.path, fileData.originalname);

  await fs.copyFile(fileData.path, filePath);

  try {
    await fs.unlink(fileData.path);
  } catch (error) {
    console.warn(`Failed to clean up temp file ${fileData.path}:`, error);
  }

  const previewDir = path.join(
    library.path,
    ".devourer",
    "series",
    series.id.toString(),
    "previews"
  );

  try {
    await fs.mkdir(previewDir, { recursive: true });
  } catch (error) {
    console.error(error);
  }

  const response = await processFileInline(
    filePath,
    path.join(previewDir, `${fileData.originalname}.jpg`)
  );

  const { volume, chapter } = extractChapterAndVolume(fileData.originalname);

  const newFile = await prisma.mangaFile.create({
    data: {
      path: filePath,
      file_name: fileData.originalname,
      file_format: fileData.originalname.split(".").pop(),
      volume: volume || 0,
      chapter: chapter || 0,
      total_pages: response?.pageCount ?? 0,
      current_page: 0,
      is_read: false,
      series_id: seriesId,
      metadata: {},
    },
  });

  return newFile;
};
