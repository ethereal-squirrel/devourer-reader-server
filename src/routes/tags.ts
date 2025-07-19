import { Router, Request, Response } from "express";

import { checkLibrary } from "../lib/library";
import { asyncHandler } from "../middleware/asyncHandler";
import { ApiError } from "../types/api";
import { checkAuth } from "../lib/auth";
import { createTag, deleteTag, getTags } from "../lib/tags";

export const tagsRouter = Router();

tagsRouter.get(
  "/tag/:libraryId/:entityId",
  checkAuth,
  asyncHandler(async (req: Request, res: Response) => {
    const { libraryId, entityId } = req.params;

    if (isNaN(Number(libraryId)) || isNaN(Number(entityId))) {
      throw new ApiError(400, "Invalid library ID or entity ID");
    }

    const library = await checkLibrary(libraryId);

    const tags = await getTags(
      library.type,
      Number(entityId),
      req.headers.user_id ? Number(req.headers.user_id) : 0
    );

    res.json({
      status: true,
      tags,
    });
  })
);

tagsRouter.post(
  "/tag/:libraryId/:entityId",
  checkAuth,
  asyncHandler(async (req: Request, res: Response) => {
    const { libraryId, entityId } = req.params;
    const { tag } = req.body;

    if (typeof tag !== "string" || tag.length > 32) {
      throw new ApiError(400, "Invalid tag");
    }

    if (isNaN(Number(libraryId)) || isNaN(Number(entityId))) {
      throw new ApiError(400, "Invalid library ID or entity ID");
    }

    const library = await checkLibrary(libraryId);

    await createTag(
      library.type,
      Number(entityId),
      req.headers.user_id ? Number(req.headers.user_id) : 0,
      tag
    );

    res.json({
      status: true,
    });
  })
);

tagsRouter.delete(
  "/tag/:libraryId/:entityId/:tag",
  checkAuth,
  asyncHandler(async (req: Request, res: Response) => {
    const { libraryId, entityId, tag } = req.params;

    if (typeof tag !== "string" || tag.length < 1 || tag.length > 32) {
      throw new ApiError(400, "Invalid tag");
    }

    if (isNaN(Number(libraryId)) || isNaN(Number(entityId))) {
      throw new ApiError(400, "Invalid library ID or entity ID");
    }

    const library = await checkLibrary(libraryId);

    await deleteTag(
      library.type,
      Number(entityId),
      req.headers.user_id ? Number(req.headers.user_id) : 0,
      tag
    );

    res.json({
      status: true,
    });
  })
);

export default tagsRouter;
