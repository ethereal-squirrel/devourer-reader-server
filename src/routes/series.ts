import { Router, Request, Response } from "express";
import path from "path";
import fs from "fs";
import multer from "multer";

import { prisma } from "../prisma";
import { checkAuth, checkRoles } from "../lib/auth";
import { downloadImage } from "../lib/file";
import { checkLibrary, convertImageDataToWebP } from "../lib/library";
import { getBook } from "../lib/book/book";
import { getSeries } from "../lib/manga/series";
import { asyncHandler } from "../middleware/asyncHandler";
import { ApiError } from "../types/api";

export const seriesRouter = Router();

const upload = multer({
  storage: multer.memoryStorage(),
  limits: {
    fileSize: 10 * 1024 * 1024,
  },
  fileFilter: (req, file, cb) => {
    if (file.mimetype.startsWith("image/")) {
      cb(null, true);
    } else {
      cb(new Error("Only image files are allowed"));
    }
  },
});

seriesRouter.get(
  "/series/:libraryId/:seriesId",
  checkAuth,
  asyncHandler(async (req: Request, res: Response) => {
    let series: any = null;

    const library = await checkLibrary(req.params.libraryId);

    if (library.type === "book") {
      series = await getBook(
        library.id,
        Number(req.params.seriesId),
        req.headers.user_id ? Number(req.headers.user_id) : 0
      );
    } else {
      series = await getSeries(
        library.id,
        Number(req.params.seriesId),
        req.headers.user_id ? Number(req.headers.user_id) : 0
      );
    }

    if (!series) {
      throw new ApiError(404, "Series not found");
    }

    res.json({
      status: true,
      series,
    });
  })
);

seriesRouter.get(
  "/series/:libraryId/:seriesId/files",
  checkAuth,
  asyncHandler(async (req: Request, res: Response) => {
    const library = await checkLibrary(req.params.libraryId);

    const series = await prisma.mangaSeries.findFirst({
      where: { library_id: library.id, id: Number(req.params.seriesId) },
    });

    if (!series) {
      throw new ApiError(404, "Series not found");
    }

    const files = await prisma.mangaFile.findMany({
      where: { series_id: series.id },
      orderBy: [
        {
          volume: "asc",
        },
        {
          chapter: "asc",
        },
      ],
    });

    let data = [] as any;

    for (const file of files) {
      const readingStatus = await prisma.readingStatus.findFirst({
        where: { file_id: file.id, user_id: Number(req.headers.user_id) },
      });

      data.push({
        ...file,
        current_page: readingStatus?.current_page || 0,
      });
    }

    res.json({
      status: true,
      files: data,
    });
  })
);

seriesRouter.patch(
  "/series/:libraryId/:seriesId/metadata",
  checkAuth,
  asyncHandler(async (req: Request, res: Response) => {
    await checkRoles(req.headers.user_roles as string, "edit_metadata");

    const { metadata } = req.body;

    if (!metadata) {
      throw new ApiError(400, "Metadata is required");
    }

    const library = await checkLibrary(req.params.libraryId);

    if (library.type === "book") {
      const book = (await prisma.bookFile.findFirst({
        where: { library_id: library.id, id: Number(req.params.seriesId) },
      })) as any;

      if (!book) {
        throw new ApiError(404, "Book not found");
      }

      await prisma.bookFile.update({
        where: { id: book.id },
        data: { metadata },
      });
    } else {
      const series = await prisma.mangaSeries.findFirst({
        where: { library_id: library.id, id: Number(req.params.seriesId) },
      });

      if (!series) {
        throw new ApiError(404, "Series not found");
      }

      await prisma.mangaSeries.update({
        where: { id: series.id },
        data: { manga_data: metadata },
      });
    }

    res.json({
      status: true,
    });
  })
);

seriesRouter.patch(
  "/series/:libraryId/:seriesId/cover",
  checkAuth,
  upload.single("cover"),
  asyncHandler(async (req: Request, res: Response) => {
    await checkRoles(req.headers.user_roles as string, "edit_metadata");

    const uploadedFile = req.file;
    const { cover: coverUrl } = req.body;

    if (!uploadedFile && !coverUrl) {
      throw new ApiError(400, "Cover image file or URL is required");
    }

    const library = await checkLibrary(req.params.libraryId);

    if (library.type === "book") {
      const book = (await prisma.bookFile.findFirst({
        where: { library_id: library.id, id: Number(req.params.seriesId) },
      })) as any;

      if (!book) {
        throw new ApiError(404, "Book not found");
      }

      const coverDir = path.join(
        library.path,
        ".devourer",
        "files",
        book.id.toString()
      );
      const coverPath = path.join(coverDir, "cover.webp");

      if (!fs.existsSync(coverDir)) {
        fs.mkdirSync(coverDir, { recursive: true });
      }

      try {
        if (uploadedFile) {
          await convertImageDataToWebP(uploadedFile.buffer, coverPath);
        } else if (coverUrl) {
          await downloadImage(coverUrl, path.join(coverDir, "cover.jpg"), true);

          const downloadedBuffer = fs.readFileSync(
            path.join(coverDir, "cover.jpg")
          );
          await convertImageDataToWebP(downloadedBuffer, coverPath);

          fs.unlinkSync(path.join(coverDir, "cover.jpg"));
        }
      } catch (error) {
        console.error(`[Series] Error saving cover:`, error);
        throw new ApiError(500, "Failed to save cover image");
      }
    } else {
      const series = await prisma.mangaSeries.findFirst({
        where: { library_id: library.id, id: Number(req.params.seriesId) },
      });

      if (!series) {
        throw new ApiError(404, "Series not found");
      }

      const coverDir = path.join(
        library.path,
        ".devourer",
        "series",
        series.id.toString()
      );
      const coverPath = path.join(coverDir, "cover.webp");

      if (!fs.existsSync(coverDir)) {
        fs.mkdirSync(coverDir, { recursive: true });
      }

      try {
        if (uploadedFile) {
          await convertImageDataToWebP(uploadedFile.buffer, coverPath);
        } else if (coverUrl) {
          await downloadImage(coverUrl, path.join(coverDir, "cover.jpg"), true);

          const downloadedBuffer = fs.readFileSync(
            path.join(coverDir, "cover.jpg")
          );
          await convertImageDataToWebP(downloadedBuffer, coverPath);

          fs.unlinkSync(path.join(coverDir, "cover.jpg"));
        }
      } catch (error) {
        console.error(`[Series] Error saving cover:`, error);
        throw new ApiError(500, "Failed to save cover image");
      }
    }

    res.json({
      status: true,
      message: uploadedFile
        ? "Cover uploaded successfully"
        : "Cover downloaded successfully",
    });
  })
);

export default seriesRouter;
