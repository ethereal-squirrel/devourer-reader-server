import { Router, Request, Response } from "express";

import { asyncHandler } from "../middleware/asyncHandler";
import { migrateCalibre } from "../lib/migrate/calibre";
import { ApiError } from "../types/api";
import { checkAuth, checkRoles } from "../lib/auth";

export const migrateRouter = Router();

migrateRouter.post(
  "/migrate/calibre",
  checkAuth,
  asyncHandler(async (req: Request, res: Response) => {
    await checkRoles(req.headers.user_roles as string, "manage_library");

    const { calibrePath, libraryName, libraryMetadataProvider } = req.body;

    if (!calibrePath) {
      throw new ApiError(400, "Calibre path is required");
    }

    if (!libraryName) {
      throw new ApiError(400, "Library name is required");
    }

    if (!libraryMetadataProvider) {
      throw new ApiError(400, "Library metadata provider is required");
    }

    migrateCalibre(
      calibrePath as string,
      libraryName as string,
      libraryMetadataProvider as string,
      req.headers.user_id ? Number(req.headers.user_id) : 0
    );
    res.json({ status: true });
  })
);

export default migrateRouter;
