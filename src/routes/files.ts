import { Router, Request, Response } from "express";
import fs from "fs";

import { prisma } from "../prisma";
import { scanEpub } from "../lib/book/bookScanner";
import { updateRecentlyRead } from "../lib/library";
import { asyncHandler } from "../middleware/asyncHandler";
import { ApiError } from "../types/api";

const router = Router();

router.get(
  "/file/:libraryId/:id",
  asyncHandler(async (req: Request<any, any, any>, res: Response) => {
    const { libraryId, id } = req.params;

    if (!id) {
      throw new ApiError(400, "File ID is required");
    }

    if (!libraryId) {
      throw new ApiError(400, "Library ID is required");
    }

    let file: any = null;

    const library = await prisma.library.findUnique({
      where: { id: parseInt(libraryId) },
    });

    if (!library) {
      throw new ApiError(400, "Library not found");
    }

    if (library.type === "book") {
      file = await prisma.bookFile.findUnique({
        where: { id: parseInt(id) },
      });
    } else {
      file = await prisma.mangaFile.findUnique({
        where: { id: parseInt(id) },
      });

      file.nextFile = null;

      if (file.volume && file.volume > 0) {
        const nextFile = await prisma.mangaFile.findFirst({
          select: {
            id: true,
            series_id: true,
          },
          where: {
            series_id: file.series_id,
            volume: file.volume + 1,
          },
        });

        if (nextFile) {
          file.nextFile = nextFile;
        }
      } else if (file.chapter && file.chapter > 0) {
        const nextFile = await prisma.mangaFile.findFirst({
          select: {
            id: true,
            series_id: true,
          },
          where: {
            series_id: file.series_id,
            chapter: file.chapter + 1,
          },
        });

        if (nextFile) {
          file.nextFile = nextFile;
        }
      }
    }

    if (!file) {
      throw new ApiError(400, "File not found");
    }

    res.json({ status: true, file });
  })
);

router.get(
  "/stream/:libraryId/:id",
  asyncHandler(async (req: Request<any, any, any>, res: Response) => {
    const { libraryId, id } = req.params;

    if (!id) {
      throw new ApiError(400, "File ID is required");
    }

    if (!libraryId) {
      throw new ApiError(400, "Library ID is required");
    }

    let file: any = null;

    const library = await prisma.library.findUnique({
      where: { id: parseInt(libraryId) },
    });

    if (!library) {
      throw new ApiError(400, "Library not found");
    }

    if (library.type === "book") {
      file = await prisma.bookFile.findUnique({
        where: { id: parseInt(id) },
      });
    } else {
      file = await prisma.mangaFile.findUnique({
        where: { id: parseInt(id) },
      });
    }

    if (!file) {
      throw new ApiError(400, "File not found");
    }

    if (library.type === "book") {
      if (fs.existsSync(file.path)) {
        const fileBuffer = fs.readFileSync(file.path);
        res.setHeader("Content-Type", "application/octet-stream");
        res.setHeader(
          "Content-Disposition",
          `attachment; filename="${file.file_name}"`
        );
        res.send(fileBuffer);
        return;
      } else {
        throw new ApiError(404, "File not found");
      }
    } else {
      if (file.file_format === "cbz" || file.file_format === "zip") {
        if (fs.existsSync(file.path)) {
          const fileBuffer = fs.readFileSync(file.path);
          res.setHeader("Content-Type", "application/zip");
          res.send(fileBuffer);
          return;
        } else {
          throw new ApiError(404, "File not found");
        }
      } else if (file.file_format === "cbr" || file.file_format === "rar") {
        if (fs.existsSync(file.path)) {
          const fileBuffer = fs.readFileSync(file.path);
          res.setHeader("Content-Type", "application/rar");
          res.send(fileBuffer);
          return;
        } else {
          throw new ApiError(404, "File not found");
        }
      } else {
        // @TODO: Convert to zip.
        throw new ApiError(400, "File is not supported");
      }
    }
  })
);

router.post(
  "/file/:id/scan",
  asyncHandler(async (req: Request<any, any, any>, res: Response) => {
    const { id } = req.params;

    if (!id) {
      throw new ApiError(400, "File ID is required");
    }

    const file = await prisma.bookFile.findUnique({
      where: { id: parseInt(id) },
    });

    if (!file) {
      throw new ApiError(400, "File not found");
    }

    const response = await scanEpub(file.path);

    if (!response) {
      throw new ApiError(400, "Failed to scan file");
    }

    res.json(response);
  })
);

router.post(
  "/file/:libraryId/:id/mark-as-read",
  asyncHandler(async (req: Request<any, any, any>, res: Response) => {
    const { libraryId, id } = req.params;

    if (!id) {
      throw new ApiError(400, "File ID is required");
    }

    if (!libraryId) {
      throw new ApiError(400, "Library ID is required");
    }

    let file: any = null;

    const library = await prisma.library.findUnique({
      where: { id: parseInt(libraryId) },
    });

    if (!library) {
      throw new ApiError(400, "Library not found");
    }

    if (library.type === "book") {
      file = await prisma.bookFile.findUnique({
        where: { id: parseInt(id) },
      });
    } else {
      file = await prisma.mangaFile.findUnique({
        where: { id: parseInt(id) },
      });
    }

    if (!file) {
      throw new ApiError(400, "File not found");
    }

    if (library.type === "book") {
      await prisma.bookFile.update({
        where: { id: parseInt(id) },
        data: { is_read: true },
      });
    } else {
      await prisma.mangaFile.update({
        where: { id: parseInt(id) },
        data: { is_read: true },
      });
    }

    res.json({ status: true });
  })
);

router.delete(
  "/file/:libraryId/:id/mark-as-read",
  asyncHandler(async (req: Request<any, any, any>, res: Response) => {
    const { libraryId, id } = req.params;

    if (!id) {
      throw new ApiError(400, "File ID is required");
    }

    if (!libraryId) {
      throw new ApiError(400, "Library ID is required");
    }

    let file: any = null;

    const library = await prisma.library.findUnique({
      where: { id: parseInt(libraryId) },
    });

    if (!library) {
      throw new ApiError(400, "Library not found");
    }

    if (library.type === "book") {
      file = await prisma.bookFile.findUnique({
        where: { id: parseInt(id) },
      });
    } else {
      file = await prisma.mangaFile.findUnique({
        where: { id: parseInt(id) },
      });
    }

    if (!file) {
      throw new ApiError(400, "File not found");
    }

    if (library.type === "book") {
      await prisma.bookFile.update({
        where: { id: parseInt(id) },
        data: { is_read: false },
      });
    } else {
      await prisma.mangaFile.update({
        where: { id: parseInt(id) },
        data: { is_read: false },
      });
    }

    res.json({ status: true });
  })
);

router.post(
  "/file/page-event",
  asyncHandler(async (req: Request<any, any, any>, res: Response) => {
    const { libraryId, fileId, page } = req.body;

    if (!fileId) {
      throw new ApiError(400, "File ID is required");
    }

    if (!libraryId) {
      throw new ApiError(400, "Library ID is required");
    }

    if (!page) {
      throw new ApiError(400, "Page is required");
    }

    let file: any = null;

    const library = await prisma.library.findUnique({
      where: { id: parseInt(libraryId) },
    });

    if (!library) {
      throw new ApiError(400, "Library not found");
    }

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
      throw new ApiError(400, "File not found");
    }

    if (library.type === "book") {
      await prisma.bookFile.update({
        where: { id: fileId },
        data: { current_page: page },
      });
    } else {
      await prisma.mangaFile.update({
        where: { id: fileId },
        data: { current_page: page },
      });
    }

    updateRecentlyRead(libraryId, fileId, page);
    res.json({ status: true });
  })
);

export default router;
