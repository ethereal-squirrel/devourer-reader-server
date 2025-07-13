import { ErrorRequestHandler } from 'express';
import { ApiError } from '../types/api';
import { PrismaClientKnownRequestError } from '@prisma/client/runtime/library';

export const errorHandler: ErrorRequestHandler = (
  error: Error,
  req,
  res,
  next
) => {
  console.error(`Error processing ${req.method} ${req.path}:`, error);

  if (error instanceof ApiError) {
    res.status(error.status).json({
      status: false,
      message: error.message,
      code: error.code
    });
    return;
  }

  if (error instanceof PrismaClientKnownRequestError) {
    switch (error.code) {
      case 'P2002':
        res.status(409).json({
          status: false,
          message: 'A unique constraint violation occurred',
          code: 'UNIQUE_VIOLATION'
        });
        return;
      case 'P2025':
        res.status(404).json({
          status: false,
          message: 'Record not found',
          code: 'NOT_FOUND'
        });
        return;
      default:
        res.status(500).json({
          status: false,
          message: 'Database error occurred',
          code: 'DATABASE_ERROR'
        });
        return;
    }
  }

  res.status(500).json({
    status: false,
    message: 'An unexpected error occurred',
    code: 'INTERNAL_ERROR'
  });
};
