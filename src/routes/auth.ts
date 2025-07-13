import { Router, Request, Response } from "express";

import { handleLogin, handleRegister } from "../lib/auth";
import { asyncHandler } from "../middleware/asyncHandler";
import { ApiError, AuthLoginRequest, AuthRegisterRequest } from "../types/api";

const router = Router();

router.post(
  "/login",
  asyncHandler(
    async (req: Request<any, any, AuthLoginRequest>, res: Response) => {
      const { username, password } = req.body;

      if (!username || !password) {
        throw new ApiError(400, "Username and password are required");
      }

      const response = await handleLogin(username, password);

      if (!response.status) {
        throw new ApiError(400, response.message || "Failed to set auth key");
      }

      res.json(response);
    }
  )
);

router.post(
  "/register",
  asyncHandler(
    async (req: Request<any, any, AuthRegisterRequest>, res: Response) => {
      const { username, password, passwordConfirm } = req.body;

      if (!username || !password || !passwordConfirm) {
        throw new ApiError(400, "All fields are required");
      }

      const response = await handleRegister(
        username,
        password,
        passwordConfirm
      );

      if (!response.status) {
        throw new ApiError(400, response.message || "Failed to set auth key");
      }

      res.json(response);
    }
  )
);

export default router;
