import { Router, Request, Response } from "express";
import fs from "fs";

import { checkAuth, checkRoles } from "../lib/auth";
import {
  getCollections,
  getCollection,
  createCollection,
  addToCollection,
  deleteFromCollection,
  deleteCollection,
} from "../lib/collection";
import {
  createLibrary,
  deleteLibrary,
  getLibraries,
  getLibrary,
  getRecentlyRead,
  getScanStatus,
  scanLibrary,
  updateLibrary,
} from "../lib/library";
import { asyncHandler } from "../middleware/asyncHandler";
import {
  ApiError,
  CollectionResponse,
  CollectionsResponse,
  LibraryResponse,
} from "../types/api";

export const libraryRouter = Router();

libraryRouter.get(
  "/status",
  checkAuth,
  asyncHandler(async (req: Request, res: Response) => {
    res.json({
      status: true,
    });
  })
);

libraryRouter.get(
  "/health",
  asyncHandler(async (req: Request, res: Response) => {
    res.json({
      status: true,
    });
  })
);

libraryRouter.get(
  "/libraries",
  checkAuth,
  asyncHandler(async (req: Request, res: Response) => {
    const librariesData = await getLibraries();

    res.json({
      status: true,
      libraries: librariesData,
    });
  })
);

libraryRouter.post(
  "/libraries",
  checkAuth,
  asyncHandler(async (req: Request, res: Response) => {
    await checkRoles(req.headers.user_roles as string, "manage_library");

    const { name, path, type, metadata } = req.body;
    if (!name || !path || !type || !metadata) {
      throw new ApiError(400, "Name, path and type are required");
    }

    if (!fs.existsSync(path)) {
      throw new ApiError(400, "Library path does not exist");
    }

    const library = await createLibrary({ name, path, type, metadata });

    res.status(201).json({
      status: true,
      library,
    });
  })
);

libraryRouter.get(
  "/recently-read",
  checkAuth,
  asyncHandler(async (req: Request, res: Response) => {
    const recentlyRead = await getRecentlyRead(
      Number(req.headers.user_id) || 0
    );

    res.json({
      status: true,
      recentlyRead,
    });
  })
);

libraryRouter.get(
  "/library/:id",
  checkAuth,
  asyncHandler(async (req: Request, res: Response<LibraryResponse>) => {
    const libraryId = Number(req.params.id);

    if (isNaN(libraryId)) {
      throw new ApiError(400, "Invalid library ID");
    }

    const library = await getLibrary(
      req.params.id,
      req.headers.user_id ? Number(req.headers.user_id) : 0
    );

    if (!library) {
      throw new ApiError(404, "Library not found");
    }

    res.json({ status: true, library });
  })
);

libraryRouter.patch(
  "/library/:id",
  checkAuth,
  asyncHandler(async (req: Request, res: Response<LibraryResponse>) => {
    await checkRoles(req.headers.user_roles as string, "manage_library");

    const libraryId = Number(req.params.id);
    const { name, path, metadata } = req.body;

    const outcome = await updateLibrary(libraryId, {
      name,
      path,
      metadata,
    });

    res.json(outcome);
  })
);

libraryRouter.delete(
  "/library/:id",
  checkAuth,
  asyncHandler(async (req: Request, res: Response<LibraryResponse>) => {
    await checkRoles(req.headers.user_roles as string, "manage_library");

    const libraryId = Number(req.params.id);

    if (isNaN(libraryId)) {
      throw new ApiError(400, "Invalid library ID");
    }

    const outcome = await deleteLibrary(libraryId);

    res.json(outcome);
  })
);

libraryRouter.get(
  "/library/:id/collections",
  checkAuth,
  asyncHandler(async (req: Request, res: Response<CollectionsResponse>) => {
    const libraryId = Number(req.params.id);

    if (isNaN(libraryId)) {
      throw new ApiError(400, "Invalid library ID");
    }

    const collections = await getCollections(
      libraryId,
      req.headers.user_id ? Number(req.headers.user_id) : 0
    );

    if (!collections) {
      throw new ApiError(404, "No collections found");
    }

    res.json({ status: true, collections });
  })
);

libraryRouter.post(
  "/library/:id/collections",
  checkAuth,
  asyncHandler(async (req: Request, res: Response<CollectionResponse>) => {
    await checkRoles(req.headers.user_roles as string, "manage_collections");

    const libraryId = Number(req.params.id);
    const { title } = req.body;

    if (isNaN(libraryId)) {
      throw new ApiError(400, "Invalid library ID");
    }

    const collection = await createCollection(
      title,
      libraryId,
      req.headers.user_id ? Number(req.headers.user_id) : 0
    );

    if (!collection) {
      throw new ApiError(404, "Failed to create collection");
    }

    res.json({ status: true });
  })
);

libraryRouter.delete(
  "/collections/:collectionId",
  checkAuth,
  asyncHandler(async (req: Request, res: Response<CollectionResponse>) => {
    await checkRoles(req.headers.user_roles as string, "manage_collections");

    const collectionId = Number(req.params.collectionId);

    if (isNaN(collectionId)) {
      throw new ApiError(400, "Invalid collectionId ID");
    }

    const collection = await deleteCollection(
      collectionId,
      req.headers.user_id ? Number(req.headers.user_id) : 0
    );

    if (!collection) {
      throw new ApiError(404, "Failed to create collection");
    }

    res.json({ status: true });
  })
);

libraryRouter.patch(
  "/collections/:collectionId/:fileId",
  checkAuth,
  asyncHandler(async (req: Request, res: Response<CollectionResponse>) => {
    await checkRoles(req.headers.user_roles as string, "manage_collections");

    const collectionId = Number(req.params.collectionId);
    const fileId = Number(req.params.fileId);

    const collection = await addToCollection(
      collectionId,
      fileId,
      req.headers.user_id ? Number(req.headers.user_id) : 0
    );

    if (!collection) {
      throw new ApiError(404, "Failed to create collection");
    }

    res.json({ status: true });
  })
);

libraryRouter.delete(
  "/collections/:collectionId/:fileId",
  checkAuth,
  asyncHandler(async (req: Request, res: Response<CollectionResponse>) => {
    await checkRoles(req.headers.user_roles as string, "manage_collections");

    const collectionId = Number(req.params.collectionId);
    const fileId = Number(req.params.fileId);

    const collection = await deleteFromCollection(
      collectionId,
      fileId,
      req.headers.user_id ? Number(req.headers.user_id) : 0
    );

    if (!collection) {
      throw new ApiError(404, "Failed to create collection");
    }

    res.json({ status: true });
  })
);

libraryRouter.get(
  "/library/:id/collections/:collectionId",
  checkAuth,
  asyncHandler(async (req: Request, res: Response<CollectionResponse>) => {
    const libraryId = Number(req.params.id);
    if (isNaN(libraryId)) {
      throw new ApiError(400, "Invalid library ID");
    }

    const collection = await getCollection(
      libraryId,
      Number(req.params.collectionId),
      req.headers.user_id ? Number(req.headers.user_id) : 0
    );

    if (!collection) {
      throw new ApiError(404, "Collection not found");
    }

    res.json({ status: true, collection });
  })
);

libraryRouter.post(
  "/library/:id/scan",
  checkAuth,
  asyncHandler(async (req: Request, res: Response<LibraryResponse>) => {
    await checkRoles(req.headers.user_roles as string, "add_file");

    const libraryId = Number(req.params.id);
    if (isNaN(libraryId)) {
      throw new ApiError(400, "Invalid library ID");
    }

    const outcome = await scanLibrary(libraryId);
    res.json(outcome);
  })
);

libraryRouter.get(
  "/library/:id/scan",
  checkAuth,
  asyncHandler(async (req: Request, res: Response<LibraryResponse>) => {
    const libraryId = Number(req.params.id);
    if (isNaN(libraryId)) {
      throw new ApiError(400, "Invalid library ID");
    }

    const outcome = await getScanStatus(libraryId);
    res.json(outcome);
  })
);

export default libraryRouter;
