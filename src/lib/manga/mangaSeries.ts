import { searchMetadata as search } from "../metadata";

import { jikanLimiter, comicVineLimiter } from "../rateLimit";

export const createSeriesPayload = async (
  provider: string,
  libraryId: number,
  series: string,
  path: string,
  mal_id: any = null,
  retrieveMetadata: boolean = false,
  apiKey?: string
) => {
  try {
    let metadata = null as any;

    if (retrieveMetadata) {
      if (provider === "comicvine") {
        metadata = await comicVineLimiter.schedule(() =>
          search(provider, mal_id ? "id" : "title", mal_id || series, apiKey)
        );
      } else {
        metadata = await jikanLimiter.schedule(() =>
          search(provider, mal_id ? "id" : "title", mal_id || series)
        );
      }
    }

    return {
      title: series,
      path,
      cover: "",
      library_id: libraryId,
      manga_data: metadata,
    };
  } catch (error) {
    console.error(error);
    return {
      title: series,
      path,
      cover: "",
      library_id: libraryId,
      manga_data: {},
    };
  }
};
