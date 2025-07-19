import { prisma } from "../../prisma";
import { ApiError } from "../../types/api";

export const getSeries = async (
  libraryId: number,
  seriesId: number,
  userId: number
) => {
  let series: any = await prisma.mangaSeries.findFirst({
    where: { library_id: libraryId, id: seriesId },
  });

  if (!series) {
    throw new ApiError(404, "Series not found");
  }

  const userRating = await prisma.userRating.findFirst({
    where: { user_id: userId, file_type: "manga", file_id: seriesId },
  });

  if (userRating) {
    series.rating = userRating.rating;
  } else {
    series.rating = null;
  }

  return series;
};
