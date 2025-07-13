import fs from "fs";
import { createCanvas, loadImage } from "canvas";
import { Uint8ArrayWriter, ZipReader } from "@zip.js/zip.js";
import { Readable } from "stream";
import webp from "webp-wasm";
import { createExtractorFromData } from "node-unrar-js";

import { getOptimalBufferSize, isImage } from "../file";

export interface ProcessFileResponse {
  pageCount?: number;
  error?: string;
}

type FileType = "zip" | "rar";

const getFileType = (fileName: string): FileType => {
  const ext = fileName.toLowerCase().split('.').pop();
  return ext === 'rar' || ext === 'cbr' ? 'rar' : 'zip';
};

export const extractChapterAndVolume = (fileName: string) => {
  const result: { chapter?: number; volume?: number } = {};

  // Match volume patterns
  const volumePatterns = [/v(?:ol(?:ume)?)?\.?\s*(\d+)/i, /\(v(\d+)\)/i];

  // Match chapter patterns
  const chapterPatterns = [/ch(?:apter)?\.?\s*(\d+\.?\d*)/i, /c(\d+\.?\d*)/i];

  for (const pattern of volumePatterns) {
    const match = fileName.match(pattern);
    if (match) {
      result.volume = parseInt(match[1]);
      break;
    }
  }

  for (const pattern of chapterPatterns) {
    const match = fileName.match(pattern);
    if (match) {
      result.chapter = parseFloat(match[1]);
      break;
    }
  }

  if (!result.volume && !result.chapter) {
    const withoutBrackets = fileName.replace(/[\[\(].*?[\]\)]/g, "");
    const numberMatch = withoutBrackets.match(/\d+/);

    if (numberMatch) {
      result.chapter = parseInt(numberMatch[0]);
    }
  }

  return result;
};

const processZipFile = async (
  file: string,
  previewPath: string
): Promise<ProcessFileResponse> => {
  const fileStream = fs.createReadStream(file, {
    highWaterMark: getOptimalBufferSize(),
    autoClose: true,
  });
  const webStream = Readable.toWeb(fileStream) as ReadableStream<Uint8Array>;
  const zipReader = new ZipReader(webStream, {
    useWebWorkers: false,
    preventClose: false,
  });

  const entries = await zipReader.getEntries({
    filenameEncoding: "utf-8",
  });

  const imageEntries = entries
    .filter((entry) => !entry.directory && isImage(entry.filename))
    .sort((a, b) => a.filename.localeCompare(b.filename));

  if (imageEntries.length === 0) {
    await zipReader.close();
    return { pageCount: 0 };
  }

  const firstImage = imageEntries[0] as any;
  const writer = new Uint8ArrayWriter();
  const imageData = await firstImage.getData(writer);
  const imageDataBuffer = Buffer.from(imageData);

  await processImageAndSave(imageDataBuffer, previewPath);
  await zipReader.close();

  return { pageCount: imageEntries.length };
};

const processImageAndSave = async (
  imageDataBuffer: Buffer,
  previewPath: string
): Promise<void> => {
  const isWebP =
    imageDataBuffer.toString("ascii", 0, 4) === "RIFF" &&
    imageDataBuffer.toString("ascii", 8, 12) === "WEBP";

  let width: number;
  let height: number;
  let imageToRender: any;

  if (isWebP) {
    const decoded = await webp.decode(imageDataBuffer);
    const tempCanvas = createCanvas(decoded.width, decoded.height);
    const tempCtx = tempCanvas.getContext("2d");
    imageToRender = tempCtx.createImageData(decoded.width, decoded.height);
    imageToRender.data.set(decoded.data);

    width = decoded.width;
    height = decoded.height;
  } else {
    imageToRender = await loadImage(imageDataBuffer);
    width = imageToRender.width;
    height = imageToRender.height;
  }

  const maxWidth = 512;
  const scale = maxWidth / width;
  const targetWidth = Math.round(width * scale);
  const targetHeight = Math.round(height * scale);
  const canvas = createCanvas(targetWidth, targetHeight);
  const ctx = canvas.getContext("2d");

  if (isWebP) {
    const tempCanvas = createCanvas(width, height);
    const tempCtx = tempCanvas.getContext("2d");
    tempCtx.putImageData(imageToRender, 0, 0);
    ctx.drawImage(tempCanvas, 0, 0, targetWidth, targetHeight);
  } else {
    ctx.drawImage(imageToRender, 0, 0, targetWidth, targetHeight);
  }

  const buffer = canvas.toBuffer("image/jpeg", {
    quality: 0.7,
    progressive: true,
  });

  await fs.promises.writeFile(previewPath, buffer);
};

const processRarFile = async (
  file: string,
  previewPath: string
): Promise<ProcessFileResponse> => {
  const fileBuffer = await fs.promises.readFile(file);
  const extractor = await createExtractorFromData({ 
    data: fileBuffer.buffer.slice(fileBuffer.byteOffset, fileBuffer.byteOffset + fileBuffer.byteLength)
  });

  const fileList = extractor.getFileList();
  const imageFiles = Array.from(fileList.fileHeaders)
    .filter((file) => !file.flags.directory && isImage(file.name))
    .sort((a, b) => a.name.localeCompare(b.name));

  if (imageFiles.length === 0) {
    return { pageCount: 0 };
  }

  const firstImageFile = imageFiles[0];
  const extracted = extractor.extract({ files: [firstImageFile.name] });
  
  for (const file of extracted.files) {
    if (file.fileHeader?.name === firstImageFile.name && file.extraction) {
      const imageDataBuffer = Buffer.from(file.extraction);
      await processImageAndSave(imageDataBuffer, previewPath);
      break;
    }
  }

  return { pageCount: imageFiles.length };
};

export const processFileInline = async (
  file: string,
  previewPath: string
): Promise<ProcessFileResponse> => {
  try {
    const fileType = getFileType(file);
    
    if (fileType === 'rar') {
      return await processRarFile(file, previewPath);
    } else {
      return await processZipFile(file, previewPath);
    }
  } catch (error) {
    console.error("Error processing file", error);
    return { error: error instanceof Error ? error.message : String(error) };
  }
};
