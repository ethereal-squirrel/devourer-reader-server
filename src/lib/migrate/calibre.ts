import fs from "fs";
import path from "path";
import sqlite3 from "sqlite3";
import { parseString } from "xml2js";
import { prisma } from "../../prisma";
import { convertImageDataToWebP } from "../library";

export interface CalibreBook {
  id: number;
  title: string;
  sort: string | null;
  timestamp: string;
  pubdate: string;
  series_index: number;
  author_sort: string | null;
  isbn: string;
  path: string;
  uuid: string | null;
  has_cover: boolean;
  last_modified: string;
  authors: CalibreAuthor[];
  series: CalibreSeries | null;
  tags: CalibreTag[];
  files: CalibreBookFile[];
  metadata: CalibreMetadata | null;
}

export interface CalibreAuthor {
  id: number;
  name: string;
  sort: string | null;
  link: string;
}

export interface CalibreSeries {
  id: number;
  name: string;
  sort: string | null;
  link: string;
}

export interface CalibreSeriesWithBooks extends CalibreSeries {
  bookIds: number[];
}

export interface CalibreTag {
  id: number;
  name: string;
}

export interface CalibreBookFile {
  id: number;
  format: string;
  name: string;
  uncompressed_size: number;
  filePath: string;
  fileName: string;
}

export interface CalibreMetadata {
  calibreId?: string;
  uuid?: string;
  title?: string;
  description?: string;
  creator?: string;
  creatorFileAs?: string;
  contributor?: string;
  date?: string;
  publisher?: string;
  language?: string;
  series?: string;
  seriesIndex?: string;
  timestamp?: string;
  titleSort?: string;
  coverPath?: string;
}

export function scanLibrary(dbPath: string): Promise<sqlite3.Database> {
  if (!fs.existsSync(dbPath)) {
    throw new Error(`Calibre database not found at: ${dbPath}`);
  }

  return new Promise((resolve, reject) => {
    const db = new sqlite3.Database(dbPath, sqlite3.OPEN_READONLY, (err) => {
      if (err) {
        reject(err);
      } else {
        resolve(db);
      }
    });
  });
}

export async function getBooks(
  db: sqlite3.Database,
  calibreLibraryPath: string
): Promise<CalibreBook[]> {
  const dbAll = (query: string, params: any[] = []): Promise<any[]> => {
    return new Promise((resolve, reject) => {
      db.all(query, params, (err, rows) => {
        if (err) reject(err);
        else resolve(rows);
      });
    });
  };

  const dbGet = (query: string, params: any[] = []): Promise<any> => {
    return new Promise((resolve, reject) => {
      db.get(query, params, (err, row) => {
        if (err) reject(err);
        else resolve(row);
      });
    });
  };

  const books = await dbAll(`
    SELECT 
      id, title, sort, timestamp, pubdate, series_index, 
      author_sort, isbn, path, uuid, has_cover, last_modified
    FROM books
    ORDER BY id
  `);

  const batchSize = 10;
  const result: CalibreBook[] = [];

  for (let i = 0; i < books.length; i += batchSize) {
    const batch = books.slice(i, i + batchSize);

    const batchPromises = batch.map(async (book) => {
      let metadata: CalibreMetadata | null = null;

      try {
        metadata = await scanMetadata(calibreLibraryPath, book.path);
      } catch (error) {
        console.warn(
          `Failed to load metadata for book ${book.id} at path ${book.path}:`,
          error
        );
      }

      const fileData = await dbAll(
        `
        SELECT id, format, name, uncompressed_size
        FROM data
        WHERE book = ?
      `,
        [book.id]
      );

      const files: CalibreBookFile[] = fileData.map((file) => ({
        ...file,
        fileName: `${file.name}.${file.format.toLowerCase()}`,
        filePath: path.join(
          calibreLibraryPath,
          book.path,
          `${file.name}.${file.format.toLowerCase()}`
        ),
      }));

      const authors = await dbAll(
        `
        SELECT a.id, a.name, a.sort, a.link
        FROM authors a
        JOIN books_authors_link bal ON a.id = bal.author
        WHERE bal.book = ?
      `,
        [book.id]
      );

      const series = await dbGet(
        `
        SELECT s.id, s.name, s.sort, s.link
        FROM series s
        JOIN books_series_link bsl ON s.id = bsl.series
        WHERE bsl.book = ?
      `,
        [book.id]
      );

      const tags = await dbAll(
        `
        SELECT t.id, t.name
        FROM tags t
        JOIN books_tags_link btl ON t.id = btl.tag
        WHERE btl.book = ?
      `,
        [book.id]
      );

      return {
        ...book,
        authors: authors as CalibreAuthor[],
        series: series as CalibreSeries | null,
        tags: tags as CalibreTag[],
        files,
        metadata,
      };
    });

    const batchResults = await Promise.all(batchPromises);
    result.push(...batchResults);
  }

  return result;
}

export async function getSeries(
  db: sqlite3.Database
): Promise<CalibreSeries[]> {
  return new Promise((resolve, reject) => {
    db.all(
      `
      SELECT id, name, sort, link
      FROM series
      ORDER BY name
    `,
      (err, rows) => {
        if (err) reject(err);
        else resolve(rows as CalibreSeries[]);
      }
    );
  });
}

export async function getSeriesWithBooks(
  db: sqlite3.Database
): Promise<CalibreSeriesWithBooks[]> {
  const dbAll = (query: string, params: any[] = []): Promise<any[]> => {
    return new Promise((resolve, reject) => {
      db.all(query, params, (err, rows) => {
        if (err) reject(err);
        else resolve(rows);
      });
    });
  };

  const series = await dbAll(`
    SELECT id, name, sort, link
    FROM series
    ORDER BY name
  `);

  const batchSize = 10;
  const result: CalibreSeriesWithBooks[] = [];

  for (let i = 0; i < series.length; i += batchSize) {
    const batch = series.slice(i, i + batchSize);

    const batchPromises = batch.map(async (seriesItem) => {
      const bookLinks = await dbAll(
        `
        SELECT book
        FROM books_series_link
        WHERE series = ?
        ORDER BY book
      `,
        [seriesItem.id]
      );

      const bookIds = bookLinks.map((link) => link.book);

      return {
        ...seriesItem,
        bookIds,
      } as CalibreSeriesWithBooks;
    });

    const batchResults = await Promise.all(batchPromises);
    result.push(...batchResults);
  }

  return result;
}

export async function scanCalibreLibrary(
  calibreLibraryPath: string
): Promise<CalibreBook[]> {
  const dbPath = path.join(calibreLibraryPath, "metadata.db");

  const db = await scanLibrary(dbPath);

  try {
    const books = await getBooks(db, calibreLibraryPath);
    return books;
  } finally {
    db.close();
  }
}

export async function migrateCalibre(
  calibreLibraryPath: string,
  libraryName: string,
  libraryMetadataProvider: string
): Promise<void> {
  const dbPath = path.join(calibreLibraryPath, "metadata.db");

  const db = await scanLibrary(dbPath);
  let totalCount: number = 0;

  try {
    const books = await getBooks(db, calibreLibraryPath);

    totalCount = books.length;

    if (totalCount > 0) {
      const library = await prisma.library.create({
        data: {
          name: libraryName,
          path: calibreLibraryPath,
          type: "book",
          metadata: {
            provider: libraryMetadataProvider,
          },
        },
      });

      if (!library) {
        throw new Error("Failed to create library");
      }

      try {
        const devourerPath = path.join(calibreLibraryPath, ".devourer");
        if (!fs.existsSync(devourerPath)) {
          fs.mkdirSync(devourerPath, { recursive: true });
        }
      } catch (err) {
        console.error("Error creating devourer directory:", err);
      }

      for (const book of books) {
        console.log(`Migrating book ${book.id} ${book.title}`);
      }

      console.log(`Found ${totalCount} books in calibre library`);

      for (const book of books) {
        await convertToDevourer(book, library.id, calibreLibraryPath);
      }
    }
  } catch (err) {
    console.error("Error migrating calibre library:", err);
    throw err;
  } finally {
    db.close();
  }
}

const convertToDevourer = async (
  calibreBook: CalibreBook,
  libraryId: number,
  calibreLibraryPath: string
) => {
  let book = {
    title: calibreBook.title,
    path: "",
    file_name: "",
    file_format: "",
    total_pages: 0,
    current_page: "",
    is_read: false,
    library_id: libraryId,
    metadata: {
      original_title: calibreBook.title,
      title: null,
      isbn_10: null,
      isbn_13: null,
      publish_date: null,
      oclc_numbers: [],
      work_key: null,
      key: null,
      dewey_decimal_class: null,
      description: null as string | null,
      authors: [] as string[],
      genres: [] as string[],
      publishers: [] as string[],
      identifiers: [],
      subtitle: null,
      number_of_pages: null,
      cover: null,
      subjects: [],
    },
    formats: [] as {
      format: string;
      name: string;
      path: string;
    }[],
    tags: [] as string[],
  };

  if (calibreBook.metadata) {
    if (calibreBook.metadata.description) {
      book.metadata.description = calibreBook.metadata.description;
    }

    if (calibreBook.authors) {
      for (const a of calibreBook.authors) {
        book.metadata.authors.push(a.name);
      }
    }

    if (calibreBook.metadata.publisher) {
      book.metadata.publishers.push(calibreBook.metadata.publisher);
    }
  }

  if (calibreBook.tags) {
    for (const t of calibreBook.tags) {
      book.tags.push(t.name);
    }
  }

  const newBook = await prisma.bookFile.create({
    data: book,
  });

  const folderPath = path.join(
    calibreLibraryPath,
    ".devourer",
    "files",
    newBook.id.toString()
  );

  try {
    fs.mkdirSync(folderPath, { recursive: true });
  } catch (err) {
    console.error("Error creating devourer folder:", err);
  }

  const validFiles: CalibreBookFile[] = [];

  if (calibreBook.files) {
    for (const f of calibreBook.files) {
      if (
        f.format.toLowerCase() !== "epub" &&
        f.format.toLowerCase() !== "pdf"
      ) {
        continue;
      }

      validFiles.push(f);

      book.formats.push({
        format: f.format.toLowerCase(),
        name: f.fileName,
        path: f.filePath,
      });
    }
  }

  if (validFiles.length === 0) {
    await prisma.bookFile.delete({
      where: {
        id: newBook.id,
      },
    });

    return false;
  }

  const firstEpub = validFiles.find((f) => f.format.toLowerCase() === "epub");

  if (firstEpub) {
    book.file_name = firstEpub.fileName;
    book.path = firstEpub.filePath;
  } else {
    const firstPdf = validFiles.find((f) => f.format.toLowerCase() === "pdf");

    if (firstPdf) {
      book.file_name = firstPdf.fileName;
      book.path = firstPdf.filePath;
    }
  }

  if (book.file_name === "" || book.path === "") {
    await prisma.bookFile.delete({
      where: {
        id: newBook.id,
      },
    });

    return false;
  }

  await prisma.bookFile.update({
    data: {
      file_name: book.file_name,
      path: book.path,
      formats: book.formats,
    },
    where: {
      id: newBook.id,
    },
  });

  if (calibreBook.metadata && calibreBook.metadata.coverPath) {
    try {
      const sourcePath = calibreBook.path.split(/[/\\]/);
      const originalCoverPath = path.join(
        calibreLibraryPath,
        ...sourcePath,
        calibreBook.metadata.coverPath
      );
      const coverPath = path.join(
        calibreLibraryPath,
        ".devourer",
        "files",
        newBook.id.toString(),
        "cover.webp"
      );

      console.log(
        `Converting cover to webp: ${originalCoverPath} -> ${coverPath}`
      );

      if (fs.existsSync(originalCoverPath)) {
        const cover = fs.readFileSync(originalCoverPath);
        await convertImageDataToWebP(cover, coverPath);
      }
    } catch (err) {
      console.error(`Error converting cover to webp:`, err);
    }
  }

  console.log(`Migrated book ${book.title} to Devourer.`);

  return true;
};

export async function scanMetadata(
  calibreLibraryPath: string,
  bookPath: string
): Promise<CalibreMetadata> {
  const metadataPath = path.join(calibreLibraryPath, bookPath, "metadata.opf");

  if (!fs.existsSync(metadataPath)) {
    throw new Error(`Metadata file not found at: ${metadataPath}`);
  }

  const xmlContent = fs.readFileSync(metadataPath, "utf-8");

  return new Promise((resolve, reject) => {
    parseString(xmlContent, (err, result) => {
      if (err) {
        reject(err);
        return;
      }

      try {
        const metadata = result.package.metadata[0];
        const guide = result.package.guide?.[0];

        const parsed: CalibreMetadata = {};

        if (metadata["dc:identifier"]) {
          metadata["dc:identifier"].forEach((id: any) => {
            if (id.$?.["opf:scheme"] === "calibre") {
              parsed.calibreId = id._;
            } else if (id.$?.["opf:scheme"] === "uuid") {
              parsed.uuid = id._;
            }
          });
        }

        if (metadata["dc:title"]?.[0]) {
          parsed.title = metadata["dc:title"][0];
        }

        if (metadata["dc:description"]?.[0]) {
          parsed.description = metadata["dc:description"][0];
        }

        if (metadata["dc:creator"]?.[0]) {
          parsed.creator = metadata["dc:creator"][0]._;
          parsed.creatorFileAs = metadata["dc:creator"][0].$?.["opf:file-as"];
        }

        if (metadata["dc:contributor"]?.[0]) {
          parsed.contributor = metadata["dc:contributor"][0]._;
        }

        if (metadata["dc:date"]?.[0]) {
          parsed.date = metadata["dc:date"][0];
        }

        if (metadata["dc:publisher"]?.[0]) {
          parsed.publisher = metadata["dc:publisher"][0];
        }

        if (metadata["dc:language"]?.[0]) {
          parsed.language = metadata["dc:language"][0];
        }

        if (metadata.meta) {
          metadata.meta.forEach((meta: any) => {
            switch (meta.$?.name) {
              case "calibre:series":
                parsed.series = meta.$?.content;
                break;
              case "calibre:series_index":
                parsed.seriesIndex = meta.$?.content;
                break;
              case "calibre:timestamp":
                parsed.timestamp = meta.$?.content;
                break;
              case "calibre:title_sort":
                parsed.titleSort = meta.$?.content;
                break;
            }
          });
        }

        if (guide?.reference) {
          const coverRef = guide.reference.find(
            (ref: any) => ref.$?.type === "cover"
          );
          if (coverRef) {
            parsed.coverPath = coverRef.$?.href;
          }
        }

        resolve(parsed);
      } catch (parseError) {
        reject(parseError);
      }
    });
  });
}
