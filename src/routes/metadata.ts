import { Router, Request, Response } from "express";

import { checkAuth } from "../lib/auth";
import { loadMetadataProviders } from "../lib/metadata";
import { asyncHandler } from "../middleware/asyncHandler";

export const metadataRouter = Router();

metadataRouter.get(
  "/metadata/providers",
  checkAuth,
  asyncHandler(async (req: Request, res: Response) => {
    const providers = await loadMetadataProviders();

    res.status(201).json({
      status: true,
      providers,
    });
  })
);

export default metadataRouter;
