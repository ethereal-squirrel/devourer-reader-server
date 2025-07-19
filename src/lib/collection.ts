import { getLibrary } from "./library";
import { prisma } from "../prisma";
import { ApiError } from "../types/api";

export const getCollections = async (libraryId: number, userId?: number) => {
  const collections = await prisma.collection.findMany({
    where: {
      library_id: libraryId,
      user_id: userId ?? 0,
    },
  });

  return collections;
};

export const getCollection = async (
  libraryId: number,
  collectionId: number,
  userId?: number
) => {
  const library = await getLibrary(libraryId.toString(), userId ?? 0);

  if (!library) {
    throw new ApiError(404, "Library not found");
  }

  let collection = (await prisma.collection.findFirst({
    where: {
      library_id: libraryId,
      id: collectionId,
      OR: [
        {
          user_id: userId ?? 0,
        },
        {
          user_id: 0,
        },
      ],
    },
  })) as any;

  if (!collection) {
    throw new ApiError(404, "Collection not found");
  }

  if (library.type === "book") {
    collection.entities = await prisma.bookFile.findMany({
      where: { id: { in: collection.series as number[] } },
    });
  } else {
    collection.entities = await prisma.mangaSeries.findMany({
      where: { id: { in: collection.series as number[] } },
    });
  }

  return collection;
};

export const createCollection = async (
  title: string,
  libraryId: number,
  userId?: number
) => {
  await prisma.collection.create({
    data: {
      name: title,
      series: [],
      library_id: libraryId,
      user_id: userId ?? 0,
    },
  });

  return true;
};

export const deleteCollection = async (
  collectionId: number,
  userId?: number
) => {
  await prisma.collection.delete({
    where: { id: collectionId, user_id: userId ?? 0 },
  });

  return true;
};

export const addToCollection = async (
  collectionId: number,
  fileId: number,
  userId?: number
) => {
  const collection = await prisma.collection.findFirst({
    where: { id: collectionId, user_id: userId ?? 0 },
  });

  if (!collection) {
    throw new ApiError(404, "Collection not found");
  }

  const series = collection.series as number[];

  if (!series.includes(fileId)) {
    series.push(fileId);
  }

  await prisma.collection.update({
    where: { id: collectionId },
    data: { series },
  });

  return true;
};

export const deleteFromCollection = async (
  collectionId: number,
  fileId: number,
  userId?: number
) => {
  const collection = await prisma.collection.findFirst({
    where: { id: collectionId, user_id: userId ?? 0 },
  });

  if (!collection) {
    throw new ApiError(404, "Collection not found");
  }

  const series = collection.series as number[];
  series.splice(series.indexOf(fileId), 1);

  await prisma.collection.update({
    where: { id: collectionId },
    data: { series },
  });

  return true;
};
