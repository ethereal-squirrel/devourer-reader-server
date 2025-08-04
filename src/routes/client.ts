import { Router } from "express";
import express from "express";
import path from "path";

export const clientRouter = Router();

clientRouter.use(
  "/client",
  express.static(path.join(__dirname, "../../client"))
);

clientRouter.get("/client/unrar.wasm", (req, res) => {
  res.set("Content-Type", "application/wasm");
  res.sendFile(path.join(__dirname, "../../client/unrar.wasm"));
});

export default clientRouter;
