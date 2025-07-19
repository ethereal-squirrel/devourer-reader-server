import { Router, Request, Response } from "express";

import { checkLibrary } from "../lib/library";
import { asyncHandler } from "../middleware/asyncHandler";
import { ApiError } from "../types/api";
import { checkAuth } from "../lib/auth";
import { rateEntity } from "../lib/ratings";

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

    const library = await checkLibrary(req.params.libraryId);
    await rateEntity(
      library.type,
      Number(entityId),
      req.headers.user_id ? Number(req.headers.user_id) : 0,
      rating
    );

    res.json({
      status: true,
    });
  })
);

export default ratingsRouter;
