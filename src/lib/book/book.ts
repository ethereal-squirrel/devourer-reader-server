import { prisma } from "../../prisma";
import { ApiError } from "../../types/api";

export const getBook = async (
  libraryId: number,
  bookId: number,
  userId: number
) => {
  let book: any = await prisma.bookFile.findFirst({
    where: { library_id: libraryId, id: bookId },
  });

  if (!book) {
    throw new ApiError(404, "Series not found");
  }

  const userRating = await prisma.userRating.findFirst({
    where: { user_id: userId, file_type: "book", file_id: bookId },
  });

  if (userRating) {
    book.rating = userRating.rating;
  } else {
    book.rating = null;
  }

  const userTags = await prisma.userTag.findMany({
    where: { user_id: userId, file_type: "book", file_id: bookId },
  });

  if (userTags.length > 0) {
    book.tags = userTags.map((tag) => tag.tag);
  } else {
    book.tags = [];
  }

  return book;
};
