import { Router, Request, Response } from "express";

import {
  checkAuth,
  checkRoles,
  handleDeleteUser,
  handleLogin,
  handleRegister,
} from "../lib/auth";
import { asyncHandler } from "../middleware/asyncHandler";
import { ApiError, AuthLoginRequest, AuthRegisterRequest } from "../types/api";
import { prisma } from "../prisma";

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
      await checkRoles(req.headers.user_roles as string, "create_user");

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

router.get(
  "/roles",
  checkAuth,
  asyncHandler(
    async (req: Request<any, any, AuthRegisterRequest>, res: Response) => {
      const user = await prisma.user.findUnique({
        where: {
          id: Number(req.headers.user_id),
        },
      });

      if (!user) {
        throw new ApiError(400, "User not found");
      }

      const roles = req.headers.user_roles as string;

      if (!roles) {
        throw new ApiError(400, "No roles found");
      }

      try {
        const rolesData = JSON.parse(roles);
        res.json({ status: true, roles: rolesData, username: user.email });
      } catch (error) {
        res.json({ status: true, roles: {} });
      }
    }
  )
);

router.delete(
  "/user",
  checkAuth,
  asyncHandler(
    async (req: Request<any, any, AuthRegisterRequest>, res: Response) => {
      await checkRoles(req.headers.user_roles as string, "create_user");

      const { username } = req.body;

      if (!username) {
        throw new ApiError(400, "Username required");
      }

      const response = await handleDeleteUser(username);

      if (!response.status) {
        throw new ApiError(400, response.message || "Failed to delete user");
      }

      res.json(response);
    }
  )
);

router.patch(
  "/user",
  checkAuth,
  asyncHandler(
    async (req: Request<any, any, AuthRegisterRequest>, res: Response) => {
      await checkRoles(req.headers.user_roles as string, "create_user");

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
