import { Router } from "express";
import express from "express";
import path from "path";

export const clientRouter = Router();

clientRouter.use(
  "/client",
  express.static(path.join(__dirname, "../../client"))
);

export default clientRouter;
