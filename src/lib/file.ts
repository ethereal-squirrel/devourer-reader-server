import fs from "fs";
import os from "os";
import path from "path";

export const MEMORY_THRESHOLDS = [
  { threshold: 16 * 1024 * 1024 * 1024, buffer: 32 * 1024 * 1024 },
  { threshold: 8 * 1024 * 1024 * 1024, buffer: 16 * 1024 * 1024 },
  { threshold: 4 * 1024 * 1024 * 1024, buffer: 8 * 1024 * 1024 },
  { threshold: 2 * 1024 * 1024 * 1024, buffer: 4 * 1024 * 1024 },
];

const validImageExtensions = [
  ".jpg",
  ".jpeg",
  ".png",
  ".gif",
  ".webp",
  ".avif",
  ".tiff",
];

export function isImage(filename: string): boolean {
  const ext = filename.toLowerCase().slice(filename.lastIndexOf("."));
  return validImageExtensions.includes(ext);
}

export const getOptimalBufferSize = () => {
  const freeMemory = os.freemem();
  const setting = MEMORY_THRESHOLDS.find((t) => freeMemory >= t.threshold);
  return setting?.buffer ?? 1 * 1024 * 1024;
};

export const downloadImage = async (
  url: string,
  targetPath: string,
  jpegHeader: boolean = true
): Promise<void> => {
  const response = await fetch(url, {
    headers: {
      Accept: jpegHeader ? "image/jpeg" : "image/*",
    },
  });
  const blob = await response.blob();
  const buffer = await blob.arrayBuffer();
  fs.writeFileSync(targetPath, Buffer.from(buffer));
};

export const getAllFiles = (dirPath: string): string[] => {
  if (!fs.existsSync(dirPath)) {
    return [];
  }

  const arrayOfFiles: string[] = [];
  const files = fs.readdirSync(dirPath);

  files.forEach((file) => {
    const fullPath = path.join(dirPath, file);
    if (fs.statSync(fullPath).isDirectory()) {
      arrayOfFiles.push(...getAllFiles(fullPath));
    } else {
      arrayOfFiles.push(fullPath);
    }
  });

  return arrayOfFiles;
};

export const getTopLevelFolders = (dirPath: string) => {
  return fs
    .readdirSync(dirPath, { withFileTypes: true })
    .filter((dirent) => dirent.isDirectory())
    .map((dirent) => dirent.name);
};
