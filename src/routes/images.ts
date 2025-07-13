import { Router, Request, Response } from "express";
import fs from "fs";
import path from "path";

import { prisma } from "../prisma";
import { getLibrary } from "../lib/library";
import { asyncHandler } from "../middleware/asyncHandler";
import { ApiError } from "../types/api";

export const imagesRouter = Router();

imagesRouter.get(
  "/cover-image/:libraryId/:entityId.webp",
  asyncHandler(async (req: Request, res: Response) => {
    const { libraryId, entityId } = req.params;

    if (isNaN(Number(libraryId)) || isNaN(Number(entityId))) {
      throw new ApiError(400, "Invalid library ID or entity ID");
    }

    const library = await getLibrary(libraryId);

    if (!library) {
      throw new ApiError(404, "Library not found");
    }

    if (library.type === "book") {
      const file = await prisma.bookFile.findUnique({
        where: {
          id: Number(entityId),
        },
      });

      if (!file) {
        throw new ApiError(404, "File not found");
      }

      const coverPath = path.join(
        library.path,
        ".devourer",
        "files",
        entityId,
        "cover.webp"
      );

      if (fs.existsSync(coverPath)) {
        const fileBuffer = fs.readFileSync(coverPath);
        res.setHeader("Content-Type", "image/webp");
        res.send(fileBuffer);
      } else {
        throw new ApiError(404, "File not found");
      }
    } else {
      const series = await prisma.mangaSeries.findUnique({
        where: {
          id: Number(entityId),
        },
      });

      if (!series) {
        throw new ApiError(404, "File not found");
      }

      const coverPath = path.join(
        library.path,
        ".devourer",
        "series",
        entityId,
        "cover.webp"
      );

      if (fs.existsSync(coverPath)) {
        const fileBuffer = fs.readFileSync(coverPath);
        res.setHeader("Content-Type", "image/webp");
        res.send(fileBuffer);
      } else {
        throw new ApiError(404, "File not found");
      }
    }
  })
);

imagesRouter.get(
  "/preview-image/:libraryId/:seriesId/:entityId.jpg",
  asyncHandler(async (req: Request, res: Response) => {
    const { libraryId, seriesId, entityId } = req.params;

    if (
      isNaN(Number(libraryId)) ||
      isNaN(Number(seriesId)) ||
      isNaN(Number(entityId))
    ) {
      throw new ApiError(400, "Invalid library ID, series ID or entity ID");
    }

    const library = await getLibrary(libraryId);

    if (!library) {
      throw new ApiError(404, "Library not found");
    }

    const series = await prisma.mangaSeries.findUnique({
      where: {
        id: Number(seriesId),
      },
    });

    if (!series) {
      throw new ApiError(404, "Series not found");
    }

    const file = await prisma.mangaFile.findUnique({
      where: {
        id: Number(entityId),
        series_id: Number(seriesId),
      },
    });

    if (!file) {
      throw new ApiError(404, "File not found");
    }

    const previewPath = path.join(
      library.path,
      ".devourer",
      "series",
      seriesId,
      "previews",
      `${file.file_name}.jpg`
    );

    if (fs.existsSync(previewPath)) {
      const fileBuffer = fs.readFileSync(previewPath);
      res.setHeader("Content-Type", "image/jpeg");
      res.send(fileBuffer);
    } else {
      throw new ApiError(404, "File not found");
    }
  })
);

export default imagesRouter;
