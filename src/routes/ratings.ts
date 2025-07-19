import { Router, Request, Response } from "express";
import fs from "fs";
import path from "path";

import { prisma } from "../prisma";
import { getLibrary } from "../lib/library";
import { asyncHandler } from "../middleware/asyncHandler";
import { ApiError } from "../types/api";
import { checkAuth } from "../lib/auth";

export const ratingsRouter = Router();

ratingsRouter.post(
  "/rate/:libraryId/:entityId",
  checkAuth,
  asyncHandler(async (req: Request, res: Response) => {
    const { libraryId, entityId } = req.params;
    const { rating } = req.body;

    if (isNaN(Number(rating)) || Number(rating) < 0 || Number(rating) > 5) {
      throw new ApiError(400, "Invalid rating");
    }

    if (isNaN(Number(libraryId)) || isNaN(Number(entityId))) {
      throw new ApiError(400, "Invalid library ID or entity ID");
    }

    const library = await getLibrary(libraryId);

    if (!library) {
      throw new ApiError(404, "Library not found");
    }

    const existingRating = await prisma.userRating.findFirst({
      where: {
        user_id: req.headers.user_id ? Number(req.headers.user_id) : 0,
        file_type: library.type,
        file_id: Number(entityId),
      },
    });

    if (existingRating) {
      await prisma.userRating.update({
        where: { id: existingRating.id },
        data: { rating: req.body.rating },
      });
    } else {
      await prisma.userRating.create({
        data: {
          user_id: req.headers.user_id ? Number(req.headers.user_id) : 0,
          file_type: library.type,
          file_id: Number(entityId),
          rating: req.body.rating,
        },
      });
    }

    res.json({
      status: true,
    });
  })
);

export default ratingsRouter;
