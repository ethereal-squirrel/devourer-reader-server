import EPub from "epub";

import { googleBooksLimiter } from "../rateLimit";

export const validBookExtensions = [
  "epub",
  "mobi",
  "pdf",
  "txt",
  "docx",
  "doc",
  "rtf",
  "html",
];

export const isValidBook = (path: string) => {
  const extension = path.split(".").pop();
  return validBookExtensions.includes(extension ?? "");
};

export const getCombinedExtensions = () => {
  return validBookExtensions.join("|");
};

export interface ProcessFileResponse {
  pageCount?: number;
  error?: string;
}

export const getGoogleUrl = (by: string, query: string) => {
  if (!["id", "title"].includes(by)) {
    throw new Error("invalid selector");
  }

  return `https://www.googleapis.com/books/v1/volumes?q=${
    by === "id" ? `isbn:${query}` : query.replace(/\s+/g, "+").toLowerCase()
  }`;
};

export const getOpenLibraryUrl = (by: string, query: string) => {
  if (!["id", "title"].includes(by)) {
    throw new Error("invalid selector");
  }

  return `https://openlibrary.org/search.json?q=${query}`;
};

export const getDevourerMetadata = async (by: string, query: string) => {
  if (!["isbn_10", "isbn_13", "title"].includes(by)) {
    throw new Error("invalid selector");
  }

  const url = `http://metadata.devourer.app/openlibrary/_search`;

  try {
    let body = {} as any;

    if (query.includes("-")) {
      const arr = query.split("-");
      query = arr[0].trim();
    }

    switch (by) {
      case "isbn_10":
        body = {
          query: { term: { isbn_10: query } },
          size: 10,
        };
        break;
      case "isbn_13":
        body = {
          query: { term: { isbn_13: query } },
          size: 10,
        };
        break;
      case "title":
        body = {
          query: { match_phrase: { title: query } },
          size: 10,
        };
        break;
    }

    const response = await fetch(url, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(body),
    });

    const data = await response.json();

    if (data.hits && data.hits.hits && data.hits.hits.length > 0) {
      return data.hits.hits.map((hit: any) => hit._source);
    } else {
      return [];
    }
  } catch (error) {
    console.error(error);
    return [];
  }
};

export const getBookMetadata = async (query: string, isbn?: string) => {
  if (!query) {
    throw new Error("invalid selector");
  }

  if (query.length === 0) {
    throw new Error("query cannot be empty");
  }

  try {
    let metadata = null;
    let selectedBook = null;

    const dataGoogle = await getGoogleMetadata("title", query);

    try {
      if (dataGoogle) {
        metadata = {
          ...dataGoogle,
          original_title: query,
        };
        return metadata;
      } else {
        throw new Error("no results found");
      }
    } catch (error) {
      console.error(`failed to fetch google metadata: ${error}`);
    }

    if (!metadata) {
      if (isbn && isbn.length === 13) {
        const dataDevourer = await getDevourerMetadata("isbn_13", isbn);

        if (dataDevourer.length > 0) {
          selectedBook = dataDevourer[0];
        }
      }

      if (isbn && isbn.length === 10 && !selectedBook) {
        const dataDevourer = await getDevourerMetadata("isbn_10", isbn);

        if (dataDevourer.length > 0) {
          selectedBook = dataDevourer[0];
        }
      }

      if (!selectedBook) {
        const dataDevourer = await getDevourerMetadata("title", query);

        if (dataDevourer.length > 0) {
          selectedBook = dataDevourer[0];
        }
      }
    }

    if (selectedBook) {
      metadata = {
        original_title: query,
        title: selectedBook.title || null,
        isbn_10: selectedBook.isbn_10 || null,
        isbn_13: selectedBook.isbn_13 || null,
        publish_date: selectedBook.publish_date || null,
        oclc_numbers: selectedBook.oclc_numbers || [],
        work_key: selectedBook.work_key || null,
        key: selectedBook.key || null,
        dewey_decimal_class: selectedBook.dewey_decimal_class || null,
        description: selectedBook.description || null,
        authors: selectedBook.authors || [],
        genres: selectedBook.genres || [],
        publishers: selectedBook.publishers || [],
        identifiers: selectedBook.identifiers || [],
        subtitle: selectedBook.subtitle || null,
        number_of_pages: selectedBook.number_of_pages || null,
        cover: selectedBook.cover || null,
        subjects: selectedBook.subjects || [],
        provider: "devourer",
      };

      if (!selectedBook.subtitle || selectedBook.subtitle.length === 0) {
        if (selectedBook.title.includes(":")) {
          const arr = selectedBook.title.split(":");

          metadata.subtitle = arr[1].trim();
          metadata.title = arr[0].trim();
        }
      }

      if (metadata.identifiers.length === 0) {
        if (metadata.isbn_13) {
          metadata.identifiers.push({
            type: "ISBN_13",
            identifier: metadata.isbn_13,
          });
        }

        if (metadata.isbn_10) {
          metadata.identifiers.push({
            type: "ISBN_10",
            identifier: metadata.isbn_10,
          });
        }
      }

      return metadata;
    } else {
      throw new Error("no results found");
    }
  } catch (error) {
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
};

const getGoogleMetadata = async (by: string, query: string) => {
  if (!["id", "title"].includes(by)) {
    throw new Error("invalid selector");
  }

  if (query.length === 0) {
    throw new Error("query cannot be empty");
  }

  try {
    let url = getGoogleUrl(by, query);

    const apiKey = process.env.GOOGLE_BOOKS_API_KEY;

    if (apiKey && apiKey.length > 0) {
      url = `${url}&key=${apiKey}`;
    }

    const response = await googleBooksLimiter.schedule(() => fetch(url));

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }

    const metadataResp = await response.json();

    if (!metadataResp.items || metadataResp.items.length === 0) {
      throw new Error("no results found");
    }

    let selectedBook = null;

    metadataResp.items = metadataResp.items.filter((item: any) => {
      if (!item.volumeInfo.categories) return true;
      return !item.volumeInfo.categories.includes("Comics & Graphic Novels");
    });

    if (metadataResp.items.length === 0) {
      throw new Error("no results found after filtering comics");
    }

    for (const e of metadataResp.items) {
      if (!e.volumeInfo.description || !e.volumeInfo.title) {
        continue;
      }

      if (e.volumeInfo.title.toLowerCase() === query.toLowerCase()) {
        selectedBook = e;
        break;
      }
    }

    if (!selectedBook) {
      selectedBook = metadataResp.items[0];
    }

    return {
      title: selectedBook.volumeInfo.title || null,
      isbn_10: null,
      isbn_13: null,
      publish_date: selectedBook.volumeInfo.publishedDate || null,
      oclc_numbers: null,
      work_key: null,
      key: null,
      dewey_decimal_class: null,
      description: selectedBook.volumeInfo.description || null,
      authors: selectedBook.volumeInfo.authors || [],
      genres: selectedBook.volumeInfo.categories || [],
      publishers: selectedBook.volumeInfo.publisher
        ? [selectedBook.volumeInfo.publisher]
        : [],
      identifiers: selectedBook.volumeInfo.industryIdentifiers || [],
      subtitle: selectedBook.volumeInfo.subtitle || null,
      number_of_pages: selectedBook.volumeInfo.pageCount || null,
      cover: selectedBook.volumeInfo.imageLinks?.thumbnail || null,
      subjects: selectedBook.volumeInfo.categories || [],
      provider: "google",
    };
  } catch (error) {
    console.error(`failed to fetch data: ${error}`);
    return null;
  }
};

export const scanEpub = async (path: string) => {
  try {
    return new Promise((resolve, reject) => {
      const book = new EPub(path);

      book.on("end", function () {
        const metadata = { ...book.metadata } as any;

        const obj = {
          title: metadata.title || null,
          author: metadata.creator || null,
          publisher: metadata.publisher || null,
          date: metadata.date || null,
          description: metadata.description || null,
          cover: metadata.cover || null,
          coverMimeType: null as string | null,
          language: metadata.language || null,
          isbn: metadata.ISBN || metadata.isbn || metadata.identifier || null,
        };

        if (obj.cover) {
          try {
            book.getImage(obj.cover, (error, img, mimeType) => {
              if (error) {
                console.error("getImage", error);
              }

              if (img && mimeType) {
                obj.cover = img;
                obj.coverMimeType = mimeType;
              }
              resolve(obj);
            });
          } catch (e) {
            obj.cover = null;
            obj.coverMimeType = null;
            resolve(obj);
          }
        } else {
          resolve(obj);
        }
      });

      book.on("error", (err) => {
        console.error("full error", err);
        reject(err);
      });

      book.parse();
    });
  } catch (err) {
    console.error(err);
    return null;
  }
};
