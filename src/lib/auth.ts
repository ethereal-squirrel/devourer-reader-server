import { Request, Response } from "express";
import { v4 as uuidv4 } from "uuid";
import bcrypt from "bcryptjs";
import jwt from "jsonwebtoken";

import { prisma } from "../prisma";

const JWT_EXPIRY = "90d";

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

export const checkAuth = async (req: Request, res: Response, next: any) => {
  try {
    const authHeader = req.headers.authorization;
    const configVar = await prisma.config.findFirst({
      where: {
        key: "allow_public",
      },
    });

    if (!configVar) {
      next();
      return;
    }

    if (configVar.value === "1") {
      next();
      return;
    }

    let user = null;

    if (authHeader && authHeader.startsWith("Bearer ")) {
      const token = authHeader.substring(7);

      try {
        const jwtSecret = await getJwtSecret();
        const decoded = jwt.verify(token, jwtSecret) as any;

        user = await prisma.user.findUnique({
          where: { id: decoded.userId },
        });

        if (user) {
          (req as any).user = user;
          req.headers.user_id = user.id.toString();
        }
      } catch (jwtError) {
        console.error("JWT verification failed:", jwtError);
      }
    } else {
      req.headers.user_id = "0";
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

export const handleLogin = async (email: string, password: string) => {
  const user = await prisma.user.findFirst({
    where: { email },
  });

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
  passwordConfirm: string,
  roles?: string[]
) => {
  if (email === "" || password === "" || passwordConfirm === "") {
    return { status: false, message: "All fields are required." };
  }

  if (password !== passwordConfirm) {
    return { status: false, message: "Passwords do not match." };
  }

  if (password.length < 8) {
    return {
      status: false,
      message: "Password must be at least 8 characters long.",
    };
  }

  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

  if (!emailRegex.test(email)) {
    return {
      status: false,
      message: "Please enter a valid email address.",
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
      roles: roles || ["user"],
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
