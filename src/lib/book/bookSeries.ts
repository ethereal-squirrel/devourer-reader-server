import path from "path";
import fs from "fs/promises";

import { searchMetadata as search } from "../metadata";
import { googleBooksLimiter, openLibraryLimiter } from "../rateLimit";
import { prisma } from "../../prisma";
import { ApiError } from "../../types/api";
import { Library } from "../../types/types";
import { scanEpub } from "./bookScanner";
import { convertImageDataToWebP, downloadAndConvertToWebP } from "../library";

const retrieveBookMetadata = async (
  by: string,
  query: string,
  provider: string
) => {
  let metadata = null as any;

  if (provider === "googlebooks") {
    metadata = await googleBooksLimiter.schedule(() =>
      search("googlebooks", by, query)
    );
  } else {
    if (query.includes("(")) {
      query = query.split("(")[0].trim();
    }

    metadata = await openLibraryLimiter.schedule(() =>
      search("openlibrary", by, query)
    );
  }

  if (!metadata) {
    return {
      original_title: query,
      title: null,
      isbn_10: null,
      isbn_13: null,
      publish_date: null,
      oclc_numbers: [],
      work_key: null,
      key: null,
      dewey_decimal_class: null,
      description: null,
      authors: [],
      genres: [],
      publishers: [],
      identifiers: [],
      subtitle: null,
      number_of_pages: null,
      cover: null,
      subjects: [],
    };
  }

  if (!metadata.subtitle || metadata.subtitle.length === 0) {
    if (metadata.title.includes(":")) {
      const arr = metadata.title.split(":");

      metadata.subtitle = arr[1].trim();
      metadata.title = arr[0].trim();
    }
  }

  return metadata;
};

export const createBookSeriesPayload = async (
  libraryId: number,
  series: string,
  path: string,
  isbn: any = null,
  retrieveMetadata: boolean = false
) => {
  try {
    const library = (await prisma.library.findUnique({
      where: {
        id: libraryId,
      },
    })) as Library | null;

    if (!library) {
      throw new ApiError(404, "Library not found");
    }

    let metadata = null as any;

    if (retrieveMetadata) {
      metadata = await retrieveBookMetadata(
        isbn && isbn.length === 10
          ? "isbn_10"
          : isbn && isbn.length === 13
          ? "isbn_13"
          : "title",
        isbn ? isbn : series,
        library.metadata?.provider || "googlebooks"
      );
    }

    if (!metadata) {
      return {
        title: series,
        path,
        cover: "",
        library_id: libraryId,
        metadata: {
          original_title: series,
          title: null,
          isbn_10: null,
          isbn_13: null,
          publish_date: null,
          oclc_numbers: [],
          work_key: null,
          key: null,
          dewey_decimal_class: null,
          description: null,
          authors: [],
          genres: [],
          publishers: [],
          identifiers: [],
          subtitle: null,
          number_of_pages: null,
          cover: null,
          subjects: [],
        },
      };
    }

    return {
      title: series,
      path,
      cover: "",
      library_id: libraryId,
      metadata,
    };
  } catch (error) {
    console.error(error);

    return {
      title: series,
      path,
      cover: "",
      library_id: libraryId,
      metadata: {
        original_title: series,
        title: null,
        isbn_10: null,
        isbn_13: null,
        publish_date: null,
        oclc_numbers: [],
        work_key: null,
        key: null,
        dewey_decimal_class: null,
        description: null,
        authors: [],
        genres: [],
        publishers: [],
        identifiers: [],
        subtitle: null,
        number_of_pages: null,
        cover: null,
        subjects: [],
      },
    };
  }
};

export const uploadBookFile = async (fileData: any, libraryId: number) => {
  const validExtensions = ["epub", "pdf"];

  if (!validExtensions.includes(fileData.originalname.split(".").pop()!)) {
    throw new ApiError(400, "Invalid file extension");
  }

  const library = (await prisma.library.findFirst({
    where: { id: libraryId },
  })) as Library | null;

  if (!library) {
    throw new ApiError(404, "Library not found");
  }

  if (library.type !== "book") {
    throw new ApiError(400, "Library is not a book library");
  }

  const metadata = await retrieveBookMetadata(
    "title",
    fileData.originalname,
    library.metadata?.provider || "googlebooks"
  );

  let filePath = null;

  if (metadata.title) {
    try {
      const folderName = metadata.title.replace(/[^a-zA-Z0-9\s-]/g, "");

      await fs.mkdir(path.join(library.path, folderName), {
        recursive: true,
      });

      filePath = path.join(library.path, folderName, fileData.originalname);
    } catch (error) {
      console.warn("Folder already exists, skipping...");
      console.error(error);
    }
  } else {
    try {
      await fs.mkdir(path.join(library.path, "uploads"), {
        recursive: true,
      });
    } catch (error) {
      console.error(error);
    }

    filePath = path.join(library.path, "uploads", fileData.originalname);
  }

  if (!filePath) {
    throw new ApiError(500, "Failed to create file path");
  }

  await fs.copyFile(fileData.path, filePath);

  try {
    await fs.unlink(fileData.path);
  } catch (error) {
    console.warn(`Failed to clean up temp file ${fileData.path}:`, error);
  }

  let epubMetadata = null as any;

  if (fileData.originalname.split(".").pop() === "epub") {
    epubMetadata = (await scanEpub(filePath)) as any;
    metadata.epub = {
      ...epubMetadata,
      cover: null,
      coverMimeType: null,
    };
  }

  const newBook = await prisma.bookFile.create({
    data: {
      title: metadata.title
        ? metadata.title
        : fileData.originalname.split(".")[0].trim(),
      path: filePath,
      file_name: fileData.originalname,
      file_format: fileData.originalname.split(".").pop(),
      total_pages: 0,
      current_page: "0",
      is_read: false,
      library_id: libraryId,
      metadata,
      formats: [
        {
          format: fileData.originalname.split(".").pop()?.toLowerCase(),
          name: fileData.originalname,
          path: filePath,
        },
      ],
      tags: [],
    },
  });

  const previewDir = path.join(
    library.path,
    ".devourer",
    "files",
    newBook.id.toString()
  );

  try {
    await fs.mkdir(previewDir, { recursive: true });
  } catch (error) {
    await prisma.bookFile.delete({
      where: { id: newBook.id },
    });

    console.error(error);
  }

  let hasCover = false;

  if (
    fileData.originalname.split(".").pop() === "epub" &&
    epubMetadata &&
    epubMetadata.cover &&
    epubMetadata.coverMimeType === "image/jpeg"
  ) {
    await convertImageDataToWebP(
      metadata.cover,
      path.join(previewDir, "cover.webp")
    );

    hasCover = true;
  }

  if (!hasCover) {
    if (
      metadata.epub &&
      metadata.epub.cover &&
      metadata.epub.cover.length > 10
    ) {
      await downloadAndConvertToWebP(
        metadata.epub.cover,
        path.join(
          library.path,
          ".devourer",
          "files",
          newBook.id.toString(),
          "cover.webp"
        )
      );

      hasCover = true;
    } else if (metadata.isbn_13 && metadata.isbn_13.length > 0) {
      const coverUrl = `https://covers.openlibrary.org/b/isbn/${metadata.isbn_13}-L.jpg`;
      await downloadAndConvertToWebP(
        coverUrl,
        path.join(
          library.path,
          ".devourer",
          "files",
          newBook.id.toString(),
          "cover.webp"
        )
      );

      hasCover = true;
    }
  }

  return newBook;
};
