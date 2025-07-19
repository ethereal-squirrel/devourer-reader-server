import { prisma } from "../prisma";

export const getTags = async (
  libraryType: string,
  entityId: number,
  userId: number
) => {
  const tags = await prisma.userTag.findMany({
    where: {
      user_id: userId,
      file_type: libraryType,
      file_id: entityId,
    },
  });

  return tags;
};

export const createTag = async (
  libraryType: string,
  entityId: number,
  userId: number,
  tag: string
) => {
  const existingTag = await prisma.userTag.findFirst({
    where: {
      user_id: userId,
      file_type: libraryType,
      file_id: entityId,
      tag: tag,
    },
  });

  if (existingTag) {
    return false;
  } else {
    await prisma.userTag.create({
      data: {
        user_id: userId,
        file_type: libraryType,
        file_id: entityId,
        tag: tag,
      },
    });

    return true;
  }
};

export const deleteTag = async (
  libraryType: string,
  entityId: number,
  userId: number,
  tag: string
) => {
  await prisma.userTag.deleteMany({
    where: {
      user_id: userId,
      file_type: libraryType,
      file_id: entityId,
      tag: tag,
    },
  });
};
