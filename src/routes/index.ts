import { Router, Request, Response } from "express";

import authRouter from "./auth";
import librariesRouter from "./libraries";
import seriesRouter from "./series";
import filesRouter from "./files";
import imagesRouter from "./images";
import clientRouter from "./client";
import opdsRouter from "./opds";

const router = Router();

router.get("/", (req: Request, res: Response) => {
  res.send("Devourer Server");
});

router.use(authRouter);
router.use(librariesRouter);
router.use(seriesRouter);
router.use(filesRouter);
router.use(imagesRouter);
router.use(clientRouter);
router.use("/opds", opdsRouter);

export default router;
