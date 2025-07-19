import { prisma } from "../prisma";

export const rateEntity = async (
  libraryType: string,
  entityId: number,
  userId: number,
  rating: number
) => {
  const existingRating = await prisma.userRating.findFirst({
    where: {
      user_id: userId,
      file_type: libraryType,
      file_id: entityId,
    },
  });

  if (existingRating) {
    await prisma.userRating.update({
      where: { id: existingRating.id },
      data: { rating: rating },
    });
  } else {
    await prisma.userRating.create({
      data: {
        user_id: userId,
        file_type: libraryType,
        file_id: entityId,
        rating: rating,
      },
    });
  }
};
