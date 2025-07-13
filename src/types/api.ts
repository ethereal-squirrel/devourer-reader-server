import { Collection } from "./types";

// Common response types
export interface ApiResponse {
  status: boolean;
  message?: string;
}

// File related types
export interface FileResponse extends ApiResponse {
  file?: {
    id: number;
    path: string;
    fileName: string;
    fileFormat: string;
    volume: number;
    chapter: number;
    totalPages: number;
    currentPage: number;
    isRead: boolean;
    seriesId: number;
  };
}

export interface StreamFileResponse {
  fileName: string;
  filePath: string;
  temporary: boolean;
  directSend?: boolean;
}

export interface PageEventRequest {
  fileId: number;
  page: number;
}

export interface PageEventResponse {
  status: boolean;
  message?: string;
}

export interface FileUpdateResponse {
  status: boolean;
  message?: string;
}

// Library related types
export interface LibraryCreateRequest {
  name: string;
  path: string;
}

export interface Library {
  id: number;
  name: string;
  path: string;
}

export interface LibraryResponse extends ApiResponse {
  library?: Library | null;
  series?: Array<{
    id: number;
    title: string;
    path: string;
    cover: string;
    libraryId: number;
    mangaData: any;
  }>;
}

export interface CollectionResponse extends ApiResponse {
  collection?: Collection | null;
}

export interface CollectionsResponse extends ApiResponse {
  collections: Collection[];
}

export interface LibrariesResponse extends ApiResponse {
  libraries: Library[];
}

export interface LibraryScanResponse {
  status: boolean;
  message?: string;
  inProgress?: boolean;
  remaining?: string[];
}

export interface LibraryScanStatusResponse {
  status: boolean;
  message?: string;
  inProgress?: boolean;
  progress?: {
    completed: number;
    total: number;
    series: any[];
  };
  startTime?: Date;
  remaining?: string[];
}

// Series related types
export interface SeriesResponse extends ApiResponse {
  id?: number;
  title?: string;
  path?: string;
  cover?: string;
  libraryId?: number;
  mangaData?: any;
  fileCount?: number;
}

export interface SeriesFilesResponse extends ApiResponse {
  files: Array<{
    id: number;
    path: string;
    fileName: string;
    fileFormat: string;
    volume: number;
    chapter: number;
    totalPages: number;
    currentPage: number;
    isRead: boolean;
    seriesId: number;
  }>;
}

// Auth related types
export interface AuthLoginRequest {
  username: string;
  password: string;
}

export interface AuthRegisterRequest {
  username: string;
  password: string;
  passwordConfirm: string;
}

// Error types
export class ApiError extends Error {
  constructor(public status: number, message: string, public code?: string) {
    super(message);
    this.name = "ApiError";
  }
}
