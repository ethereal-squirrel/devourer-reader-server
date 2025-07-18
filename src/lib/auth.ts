import { Request, Response } from "express";
import { v4 as uuidv4 } from "uuid";
import bcrypt from "bcryptjs";
import jwt from "jsonwebtoken";

import { prisma } from "../prisma";
import { ApiError } from "../types/api";

const JWT_EXPIRY = "90d";

const rolesData = {
  admin: {
    is_admin: true,
    add_file: true,
    delete_file: true,
    edit_metadata: true,
    manage_collections: true,
    manage_library: true,
    create_user: true,
  },
  moderator: {
    is_admin: false,
    add_file: true,
    delete_file: true,
    edit_metadata: true,
    manage_collections: true,
    manage_library: false,
    create_user: false,
  },
  upload: {
    is_admin: false,
    add_file: true,
    delete_file: false,
    edit_metadata: false,
    manage_collections: false,
    manage_library: false,
    create_user: false,
  },
  user: {
    is_admin: false,
    add_file: false,
    delete_file: false,
    edit_metadata: false,
    manage_collections: false,
    manage_library: false,
    create_user: false,
  },
};

const getJwtSecret = async (): Promise<string> => {
  const config = await prisma.config.findUnique({
    where: { key: "jwt_secret" },
  });

  if (!config) {
    throw new Error(
      "JWT secret not found in database. Please run initial migration."
    );
  }

  return config.value;
};

export const getRequestUser = (req: Request): any => {
  return (req as any).user;
};

export const verifyJwtToken = async (token: string) => {
  try {
    const jwtSecret = await getJwtSecret();
    const decoded = jwt.verify(token, jwtSecret) as any;
    const user = await prisma.user.findUnique({
      where: { id: decoded.userId },
    });
    return { valid: true, user, decoded };
  } catch (error) {
    return { valid: false, error };
  }
};

export const checkRoles = async (userRoles: string, requiredRole: string) => {
  if (!userRoles) {
    throw new ApiError(400, "You do not have permission to do this.");
  }

  const userRolesData = JSON.parse(userRoles);

  if (userRolesData.is_admin) {
    return true;
  }

  if (userRolesData[requiredRole]) {
    return true;
  }

  throw new ApiError(400, "You do not have permission to do this.");
};

export const checkAuth = async (req: Request, res: Response, next: any) => {
  try {
    const authHeader = req.headers.authorization;
    const configVar = await prisma.config.findFirst({
      where: {
        key: "allow_public",
      },
    });

    if (!configVar) {
      req.headers.user_id = "0";
      req.headers.user_roles = JSON.stringify(rolesData.user);
      next();
      return;
    }

    if (configVar.value === "1") {
      req.headers.user_id = "0";
      req.headers.user_roles = JSON.stringify(rolesData.user);
      next();
      return;
    }

    let user = null;

    if (authHeader && authHeader.startsWith("Bearer ")) {
      const token = authHeader.substring(7);

      try {
        const jwtSecret = await getJwtSecret();
        const decoded = jwt.verify(token, jwtSecret) as any;

        user = (await prisma.user.findUnique({
          where: { id: decoded.userId },
        })) as any;

        let roles = {
          is_admin: false,
          add_file: false,
          delete_file: false,
          edit_metadata: false,
          manage_collections: false,
          manage_library: false,
          create_user: false,
        } as any;

        if (user.roles.length > 0) {
          const userRoles = await prisma.roles.findMany({
            where: {
              title: { in: user.roles },
            },
          });

          for (const role of userRoles) {
            roles = rolesData[role.title as keyof typeof rolesData];
          }
        }

        if (user) {
          (req as any).user = user;
          req.headers.user_id = user.id.toString();
          req.headers.user_roles = JSON.stringify(roles);
        }
      } catch (jwtError) {
        console.error("JWT verification failed:", jwtError);
      }
    } else {
      req.headers.user_id = "0";
      req.headers.user_roles = JSON.stringify(rolesData.user);
    }

    if (!user && configVar.value === "0") {
      res
        .status(401)
        .json({ status: false, message: "Authentication required" });
      return;
    }

    next();
  } catch (error) {
    console.error(error);
    res.status(401).json({ status: false, message: "Authentication failed" });
    return;
  }
};

export const resetPassword = async (email: string, password: string) => {
  const user = await prisma.user.findFirst({
    where: { email },
  });

  if (!user) {
    return { status: false, message: "User not found." };
  }

  const hashedPassword = await bcrypt.hash(password, 12);
  await prisma.user.update({
    where: { id: user.id },
    data: { password: hashedPassword },
  });

  return { status: true, message: "Password reset successful." };
};

export const handleLogin = async (email: string, password: string) => {
  const user = (await prisma.user.findFirst({
    where: { email },
  })) as any;

  if (!user) {
    return { status: false, message: "User not found." };
  }

  const outcome = await bcrypt.compare(password, user.password);

  if (!outcome) {
    return { status: false, message: "Invalid password." };
  }

  const jwtSecret = await getJwtSecret();
  const token = jwt.sign(
    {
      userId: user.id,
      email: user.email,
      roles: user.roles,
    },
    jwtSecret,
    { expiresIn: JWT_EXPIRY }
  );

  const roles = {
    is_admin: false,
    add_file: false,
    delete_file: false,
    edit_metadata: false,
    manage_collections: false,
    manage_library: false,
    create_user: false,
  } as any;

  if (user.roles.length > 0) {
    const userRoles = await prisma.roles.findMany({
      where: {
        title: { in: user.roles },
      },
    });

    for (const role of userRoles) {
      roles[role.title as keyof typeof roles] =
        rolesData[role.title as keyof typeof rolesData];
    }
  }

  return {
    status: true,
    token,
    user: {
      id: user.id,
      email: user.email,
      roles: user.roles,
    },
    message: "Login successful.",
  };
};

export const handleRegister = async (
  email: string,
  password: string,
  role: string
) => {
  if (email === "" || password === "") {
    return { status: false, message: "All fields are required." };
  }

  if (password.length < 8) {
    return {
      status: false,
      message: "Password must be at least 8 characters long.",
    };
  }

  const salt = await bcrypt.genSalt(10);
  const hash = await bcrypt.hash(password, salt);

  const apiKey = uuidv4();
  const hashedApiKey = await bcrypt.hash(apiKey, 12);

  const user = await prisma.user.create({
    data: {
      email,
      password: hash,
      roles: [role],
      api_key: hashedApiKey,
      metadata: {
        settings: {
          book_pagemode: "single",
          book_font: "default",
          book_background: "#000000",
          manga_direction: "ltr",
          manga_pagemode: "single",
          manga_resizemode: "fit",
          manga_background: "#000000",
        },
      },
      created_at: new Date(),
    },
  });

  const jwtSecret = await getJwtSecret();
  const token = jwt.sign(
    {
      userId: user.id,
      email: user.email,
      roles: user.roles,
    },
    jwtSecret,
    { expiresIn: JWT_EXPIRY }
  );

  return {
    status: true,
    token,
    user: {
      id: user.id,
      email,
      api_key: apiKey,
      roles: user.roles,
    },
    message: "User created successfully.",
  };
};

export const handleDeleteUser = async (id: string) => {
  const user = await prisma.user.findFirst({
    where: { id: Number(id) },
  });

  if (!user) {
    return { status: false, message: "User not found." };
  }

  if (user.roles && (user.roles as string[]).includes("admin")) {
    return { status: false, message: "Cannot delete admin user." };
  }

  await prisma.collection.deleteMany({
    where: {
      user_id: user.id,
    },
  });

  await prisma.recentlyRead.deleteMany({
    where: {
      user_id: user.id,
    },
  });

  await prisma.readingStatus.deleteMany({
    where: {
      user_id: user.id,
    },
  });

  await prisma.user.delete({
    where: { id: user.id },
  });

  return { status: true, message: "User deleted successfully." };
};
