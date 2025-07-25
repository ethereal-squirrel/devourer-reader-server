import { Router, Request, Response } from "express";

import authRouter from "./auth";
import librariesRouter from "./libraries";
import seriesRouter from "./series";
import filesRouter from "./files";
import imagesRouter from "./images";
import clientRouter from "./client";
import opdsRouter from "./opds";
import ratingsRouter from "./ratings";
import tagsRouter from "./tags";
import metadataRouter from "./metadata";
import migrateRouter from "./migrate";

const router = Router();

router.get("/", (req: Request, res: Response) => {
  res.send("Devourer Server");
});

router.get("/version", (req: Request, res: Response) => {
  res.json({
    version: require("../../package.json").version,
  });
});

router.use(authRouter);
router.use(librariesRouter);
router.use(seriesRouter);
router.use(filesRouter);
router.use(imagesRouter);
router.use(clientRouter);
router.use(ratingsRouter);
router.use(tagsRouter);
router.use(metadataRouter);
router.use(migrateRouter);
router.use("/opds", opdsRouter);

export default router;
