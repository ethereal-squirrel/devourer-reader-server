import { Router, Request, Response } from "express";

import { PrismaClient } from "../../generated/prisma/client";

const router = Router();
const prisma = new PrismaClient();

const naturalSort = (a: string, b: string): number => {
  const collator = new Intl.Collator(undefined, {
    numeric: true,
    sensitivity: "base",
  });
  return collator.compare(a, b);
};

interface CacheEntry {
  data: any[];
  timestamp: number;
}

const bookCache = new Map<string, CacheEntry>();
const CACHE_TTL = 300 * 1000;

const getCachedBooks = async (libraryId: string) => {
  const cacheKey = `library_${libraryId}`;
  const now = Date.now();

  const cached = bookCache.get(cacheKey);
  if (cached && now - cached.timestamp < CACHE_TTL) {
    return cached.data;
  }

  const books = await prisma.bookFile.findMany({
    where: { library_id: parseInt(libraryId) },
    select: { id: true, title: true, file_format: true, metadata: true },
  });

  const sortedBooks = books.sort((a, b) => naturalSort(a.title, b.title));

  bookCache.set(cacheKey, {
    data: sortedBooks,
    timestamp: now,
  });

  return sortedBooks;
};

const invalidateLibraryCache = (libraryId: string) => {
  const cacheKey = `library_${libraryId}`;
  bookCache.delete(cacheKey);
};

const clearAllCache = () => {
  bookCache.clear();
};

const OPDS_NAVIGATION_TYPE =
  "application/atom+xml;profile=opds-catalog;kind=navigation";
const OPDS_ACQUISITION_TYPE =
  "application/atom+xml;profile=opds-catalog;kind=acquisition";

const generateOpdsXml = (feed: any): string => {
  const { id, title, updated, links = [], entries = [] } = feed;

  const linkElements = links
    .map(
      (link: any) =>
        `<link rel="${link.rel}" href="${link.href}" type="${link.type}"/>`
    )
    .join("\n    ");

  const entryElements = entries
    .map((entry: any) => {
      const entryLinks =
        entry.links
          ?.map(
            (link: any) =>
              `<link rel="${link.rel}" href="${link.href}" type="${link.type}"/>`
          )
          .join("\n        ") || "";

      return `
    <entry>
        <id>${entry.id}</id>
        <title>${entry.title}</title>
        <updated>${entry.updated}</updated>
        ${
          entry.content ? `<content type="text">${entry.content}</content>` : ""
        }
        ${entry.author ? `<author><name>${entry.author}</name></author>` : ""}
        ${entryLinks}
    </entry>`;
    })
    .join("\n");

  return `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom" 
      xmlns:dc="http://purl.org/dc/terms/" 
      xmlns:opds="http://opds-spec.org/2010/catalog">
    <id>${id}</id>
    <title>${title}</title>
    <updated>${updated}</updated>
    <author>
        <name>Devourer Server</name>
    </author>
    ${linkElements}
    ${entryElements}
</feed>`;
};

router.get("/v1.2/catalog", async (req: Request, res: Response) => {
  try {
    const libraries = await prisma.library.findMany({
      where: { type: "book" },
      include: {
        _count: {
          select: { book_files: true },
        },
      },
    });

    const feed = {
      id: "urn:uuid:devourer-opds-root",
      title: "Devourer Library",
      updated: new Date().toISOString(),
      links: [
        {
          rel: "self",
          href: "/opds/v1.2/catalog",
          type: OPDS_NAVIGATION_TYPE,
        },
        {
          rel: "start",
          href: "/opds/v1.2/catalog",
          type: OPDS_NAVIGATION_TYPE,
        },
      ],
      entries: libraries.map((library: any) => ({
        id: `urn:uuid:devourer-library-${library.id}`,
        title: library.name,
        updated: new Date().toISOString(),
        content: `${library._count.book_files} books`,
        links: [
          {
            rel: "subsection",
            href: `/opds/v1.2/libraries/${library.id}`,
            type: OPDS_ACQUISITION_TYPE,
          },
        ],
      })),
    };

    res.setHeader("Content-Type", OPDS_NAVIGATION_TYPE);
    res.send(generateOpdsXml(feed));
  } catch (error) {
    console.error("Error generating root OPDS feed:", error);
    res.status(500).send("Internal Server Error");
  }
});

router.get("/v1.2/libraries/:libraryId", async (req: any, res: any) => {
  try {
    const { libraryId } = req.params;
    const page = parseInt(req.query.page as string) || 1;
    const limit = parseInt(req.query.limit as string) || 50;
    const offset = (page - 1) * limit;

    const library = await prisma.library.findUnique({
      where: { id: parseInt(libraryId) },
    });

    if (!library) {
      return res.status(404).send("Library not found");
    }

    const allBooks = await getCachedBooks(libraryId);

    const bookFiles = allBooks.slice(offset, offset + limit);

    const totalCount = allBooks.length;

    const hasNext = offset + limit < totalCount;
    const hasPrev = page > 1;

    const feed = {
      id: `urn:uuid:devourer-library-${libraryId}`,
      title: `${library.name} - Books`,
      updated: new Date().toISOString(),
      links: [
        {
          rel: "self",
          href: `/opds/v1.2/libraries/${libraryId}?page=${page}`,
          type: OPDS_ACQUISITION_TYPE,
        },
        {
          rel: "start",
          href: "/opds/v1.2/catalog",
          type: OPDS_NAVIGATION_TYPE,
        },
        {
          rel: "up",
          href: "/opds/v1.2/catalog",
          type: OPDS_NAVIGATION_TYPE,
        },
        ...(hasNext
          ? [
              {
                rel: "next",
                href: `/opds/v1.2/libraries/${libraryId}?page=${page + 1}`,
                type: OPDS_ACQUISITION_TYPE,
              },
            ]
          : []),
        ...(hasPrev
          ? [
              {
                rel: "prev",
                href: `/opds/v1.2/libraries/${libraryId}?page=${page - 1}`,
                type: OPDS_ACQUISITION_TYPE,
              },
            ]
          : []),
      ],
      entries: bookFiles.map((book: any) => {
        const metadata = (book.metadata as any) || {};
        return {
          id: `urn:uuid:devourer-book-${book.id}`,
          title: book.title,
          updated: new Date().toISOString(),
          content: metadata.description || "",
          author: metadata.author || "",
          links: [
            {
              rel: "http://opds-spec.org/acquisition",
              href: `/stream/${libraryId}/${book.id}`,
              type:
                book.file_format === "epub"
                  ? "application/epub+zip"
                  : "application/pdf",
            },
            {
              rel: "http://opds-spec.org/image",
              href: `/opds/covers/${libraryId}/${book.id}`,
              type: "image/webp",
            },
          ],
        };
      }),
    };

    res.setHeader("Content-Type", OPDS_ACQUISITION_TYPE);
    res.send(generateOpdsXml(feed));
  } catch (error) {
    console.error("Error generating library OPDS feed:", error);
    res.status(500).send("Internal Server Error");
  }
});

router.get("/covers/:libraryId/:fileId", async (req: any, res: any) => {
  try {
    const { libraryId, fileId } = req.params;
    res.redirect(`/cover-image/${libraryId}/${fileId}.webp`);
  } catch (error) {
    console.error("Error serving cover:", error);
    res.status(500).send("Internal Server Error");
  }
});

router.get("/v1.2/search", async (req: any, res: any) => {
  try {
    const query = req.query.q as string;
    if (!query) {
      return res.status(400).send("Search query required");
    }

    const bookFiles = await prisma.bookFile.findMany({
      where: {
        OR: [
          { title: { contains: query } },
          { file_name: { contains: query } },
        ],
      },
      take: 50,
      include: {
        Library: true,
      },
    });

    bookFiles.sort((a, b) => naturalSort(a.title, b.title));

    const feed = {
      id: `urn:uuid:devourer-search-${encodeURIComponent(query)}`,
      title: `Search Results for "${query}"`,
      updated: new Date().toISOString(),
      links: [
        {
          rel: "self",
          href: `/opds/v1.2/search?q=${encodeURIComponent(query)}`,
          type: OPDS_ACQUISITION_TYPE,
        },
        {
          rel: "start",
          href: "/opds/v1.2/catalog",
          type: OPDS_NAVIGATION_TYPE,
        },
      ],
      entries: bookFiles.map((book: any) => {
        const metadata = (book.metadata as any) || {};
        return {
          id: `urn:uuid:devourer-book-${book.id}`,
          title: book.title,
          updated: new Date().toISOString(),
          content: metadata.description || "",
          author: metadata.author || "",
          links: [
            {
              rel: "http://opds-spec.org/acquisition",
              href: `/stream/${book.Library.id}/${book.id}`,
              type:
                book.file_format === "epub"
                  ? "application/epub+zip"
                  : "application/pdf",
            },
            {
              rel: "http://opds-spec.org/image",
              href: `/opds/covers/${book.Library.id}/${book.id}`,
              type: "image/webp",
            },
          ],
        };
      }),
    };

    res.setHeader("Content-Type", OPDS_ACQUISITION_TYPE);
    res.send(generateOpdsXml(feed));
  } catch (error) {
    console.error("Error generating search OPDS feed:", error);
    res.status(500).send("Internal Server Error");
  }
});

router.get("/", (req: any, res: any) => {
  res.redirect(301, "/opds/v1.2/catalog");
});

router.get("/libraries/:libraryId", (req: any, res: any) => {
  const { libraryId } = req.params;
  const queryString = req.url.split("?")[1];
  const redirectUrl = `/opds/v1.2/libraries/${libraryId}${
    queryString ? `?${queryString}` : ""
  }`;
  res.redirect(301, redirectUrl);
});

router.get("/search", (req: any, res: any) => {
  const queryString = req.url.split("?")[1];
  const redirectUrl = `/opds/v1.2/search${
    queryString ? `?${queryString}` : ""
  }`;
  res.redirect(301, redirectUrl);
});

router.post("/v1.2/cache/clear", (req: any, res: any) => {
  const { libraryId } = req.body;

  if (libraryId) {
    invalidateLibraryCache(libraryId);
    res.json({ message: `Cache cleared for library ${libraryId}` });
  } else {
    clearAllCache();
    res.json({ message: "All caches cleared" });
  }
});

export { invalidateLibraryCache, clearAllCache };

export default router;
