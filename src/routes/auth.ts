import { Router, Request, Response } from "express";

import {
  checkAuth,
  checkRoles,
  handleDeleteUser,
  handleEditUser,
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
  "/users",
  checkAuth,
  asyncHandler(
    async (req: Request<any, any, AuthRegisterRequest>, res: Response) => {
      await checkRoles(req.headers.user_roles as string, "create_user");

      const { username, password, role } = req.body;

      if (!username || !password || !role) {
        throw new ApiError(400, "All fields are required");
      }

      const response = await handleRegister(username, password, role);

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

router.get(
  "/users",
  checkAuth,
  asyncHandler(
    async (req: Request<any, any, AuthRegisterRequest>, res: Response) => {
      let users = (await prisma.user.findMany({
        select: {
          id: true,
          email: true,
          roles: true,
        },
      })) as any;

      for (let i = 0; i < users.length; i++) {
        const user = users[i];

        const userCollections = await prisma.collection.findMany({
          where: {
            user_id: user.id,
          },
        });

        users[i].collections = userCollections.length;
      }

      res.json({ status: true, users });
    }
  )
);

router.delete(
  "/user/:id",
  checkAuth,
  asyncHandler(
    async (req: Request<any, any, AuthRegisterRequest>, res: Response) => {
      await checkRoles(req.headers.user_roles as string, "create_user");

      const { id } = req.params;

      if (!id || isNaN(Number(id)) || Number(id) === 0) {
        throw new ApiError(400, "Invalid user id");
      }

      const response = await handleDeleteUser(id);

      if (!response.status) {
        throw new ApiError(400, response.message || "Failed to delete user");
      }

      res.json(response);
    }
  )
);

router.patch(
  "/user/:id",
  checkAuth,
  asyncHandler(
    async (req: Request<any, any, AuthRegisterRequest>, res: Response) => {
      await checkRoles(req.headers.user_roles as string, "create_user");

      const { id } = req.params;

      if (!id || isNaN(Number(id)) || Number(id) === 0) {
        throw new ApiError(400, "Invalid user id");
      }

      const { role, password } = req.body;

      if (!role) {
        throw new ApiError(400, "Role is required");
      }

      const response = await handleEditUser(id, role, password);

      if (!response.status) {
        throw new ApiError(400, response.message || "Failed to set auth key");
      }

      res.json(response);
    }
  )
);

export default router;
