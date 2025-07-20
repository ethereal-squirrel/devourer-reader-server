import { searchMetadata as search } from "../metadata";

import { googleBooksLimiter, openLibraryLimiter } from "../rateLimit";
import { prisma } from "../../prisma";
import { ApiError } from "../../types/api";
import { Library } from "../../types/types";

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
      let by = "title";

      if (isbn && isbn.length === 10) {
        by = "isbn_10";
      } else if (isbn && isbn.length === 13) {
        by = "isbn_13";
      }

      if (library.metadata?.provider === "googlebooks") {
        metadata = await googleBooksLimiter.schedule(() =>
          search(
            "googlebooks",
            by as "id" | "title" | "isbn_13" | "isbn_10",
            isbn || series
          )
        );
      } else {
        if (series.includes("(")) {
          series = series.split("(")[0].trim();
        }

        metadata = await openLibraryLimiter.schedule(() =>
          search(
            "openlibrary",
            by as "id" | "title" | "isbn_13" | "isbn_10",
            isbn || series
          )
        );
      }
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

    if (!metadata.subtitle || metadata.subtitle.length === 0) {
      if (metadata.title.includes(":")) {
        const arr = metadata.title.split(":");

        metadata.subtitle = arr[1].trim();
        metadata.title = arr[0].trim();
      }
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
