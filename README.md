# Devourer (Server)

A library server created to provide a highly performant system to archive, scan and catalog your books, manga and comics.

### Features

- A fast library server written in Node.js.
- Provides a simple API server to perform actions.
- SQLite powered storage.
- Supports multiple users in both public and private mode.
- Retrieves relevant cover images and generates a preview image for each archive in addition.
- Automatically retrieve metadata from various providers.
- Supports libraries and collections. A library at the top level is either a collection of "books" or "comics"; a collection exists within this.
- Supports Windows, Linux and Mac.

### Manga / Comic Features

- Supports .zip, .cbz, .rar and .cbr archives. More formats to follow shortly (such as folders of images and 7zip).
- Streams selected archives back to the client.

### Book Features

- A fast library server written in Node.js.
- Supports EPUB and PDF currently. More formats to follow shortly.
- Support for OPDS 1.2.

### Binary Releases

[Releases page](https://github.com/ethereal-squirrel/devourer-reader-server/releases)

### Manual Install

- Ensure you have Node.js installed.
- Clone this repository and cd into the folder.
- Install Dependencies: npm i
- Generate Prisma Client: npx prisma generate
- To Build: npx tsc
- To Run: node dist/index.js

### How To Use

The server exposes the following API endpoints on port 9024. Or, you can just use [Devourer](https://devourer.app); an application designed to work with this server.

**Note**: Most endpoints require authentication (Bearer token) except where noted.

## Authentication

#### [POST] /login

Authenticate a user and receive a JWT token.

Expects: { "username": "user", "password": "password" }

#### [POST] /register

Register a new user account.

Expects: { "username": "user", "password": "password" }

## System Status

#### [GET] /health

Health check endpoint (no authentication required).

#### [GET] /status

Get server status (authentication required).

#### [GET] /recently-read

Get recently read items across all libraries.

## Library Management

#### [GET] /libraries

Get a list of all libraries.

#### [POST] /libraries

Create a new library. Accept a JSON payload of **name**, **path**, **type**, and **metadata**.

Expects: { "name": "Manga", "path": "D:\\Manga", "type": "manga", "metadata": { "provider": "myanimelist", "apiKey": "optional" } }

#### [GET] /library/:id

Retrieve details for the specified library.

Parameters: [id: libraryId]

#### [PATCH] /library/:id

Update library details.

Parameters: [id: libraryId]

#### [DELETE] /library/:id

Delete the specified library and all its content.

Parameters: [id: libraryId]

#### [POST] /library/:id/scan

Scan the specified library for new content.

Parameters: [id: libraryId]

#### [GET] /library/:id/scan

Retrieve scan status details for the specified library.

Parameters: [id: libraryId]

## Collections

#### [GET] /library/:id/collections

Retrieve collections for a given library.

Parameters: [id: libraryId]

#### [POST] /library/:id/collections

Create a new collection. Accepts a JSON payload of **name** (name of collection).

Expects: { "name": "Action Manga" }

Parameters: [id: libraryId]

#### [GET] /library/:id/collections/:collectionId

Retrieve a specific collection.

Parameters: [id: libraryId, collectionId: collectionId]

#### [DELETE] /collections/:collectionId

Delete a collection.

Parameters: [collectionId: collectionId]

#### [PATCH] /collections/:collectionId/:fileId

Add a file to a collection.

Parameters: [collectionId: collectionId, fileId: fileId]

#### [DELETE] /collections/:collectionId/:fileId

Remove a file from a collection.

Parameters: [collectionId: collectionId, fileId: fileId]

## Series Management

#### [GET] /series/:libraryId/:seriesId

Retrieve details for the specified series.

Parameters: [libraryId: libraryId, seriesId: seriesId]

#### [GET] /series/:libraryId/:seriesId/files

Retrieve files for the specified series.

Parameters: [libraryId: libraryId, seriesId: seriesId]

#### [PATCH] /series/:libraryId/:seriesId/metadata

Update series metadata.

Parameters: [libraryId: libraryId, seriesId: seriesId]

#### [PATCH] /series/:libraryId/:seriesId/cover

Update series cover image. Supports multipart file upload.

Parameters: [libraryId: libraryId, seriesId: seriesId]

## File Management

#### [GET] /file/:libraryId/:id

Retrieve details for the specified file.

Parameters: [libraryId: libraryId, id: fileId]

#### [GET] /stream/:libraryId/:id

Stream/download the file. Returns the file as a .zip archive. If the archive is not a zip, it will extract it and convert it to a zip for client compatibility.

Parameters: [libraryId: libraryId, id: fileId]

#### [POST] /file/:id/scan

Scan an EPUB file for metadata and content.

Parameters: [id: fileId]

#### [POST] /file/:libraryId/:id/mark-as-read

Mark the specified file as read.

Parameters: [libraryId: libraryId, id: fileId]

#### [DELETE] /file/:libraryId/:id/mark-as-read

Mark the specified file as unread.

Parameters: [libraryId: libraryId, id: fileId]

#### [POST] /file/page-event

Update reading progress for a file.

Expects: { "fileId": 1, "page": 1, "libraryId": 1 }

## Images

#### [GET] /cover-image/:libraryId/:entityId.webp

Retrieve the cover image for the specified series or file.

Parameters: [libraryId: libraryId, entityId: seriesId or fileId]

#### [GET] /preview-image/:libraryId/:seriesId/:entityId.jpg

Retrieve the preview image for the specified file.

Parameters: [libraryId: libraryId, seriesId: seriesId, entityId: fileId]

## OPDS Support

The server provides full OPDS 1.2 support for e-reader compatibility:

#### [GET] /opds/

OPDS root endpoint (redirects to catalog).

#### [GET] /opds/v1.2/catalog

OPDS catalog root showing available libraries.

#### [GET] /opds/v1.2/libraries/:libraryId

OPDS feed for a specific library.

Parameters: [libraryId: libraryId]

#### [GET] /opds/v1.2/search

OPDS search functionality.

Query parameters: [q: search term]

#### [GET] /opds/covers/:libraryId/:fileId

OPDS cover image redirect.

Parameters: [libraryId: libraryId, fileId: fileId]

#### [POST] /opds/v1.2/cache/clear

Clear OPDS cache.

## Command Line Interface

You can also run the following commands at the command line:

#### ./devourer-server create-library :name :path :type :provider [:apiKey]

Create a library with the specified name, path, type, and metadata provider.

Example: `./devourer-server create-library "My Manga" "D:/Manga" "manga" "myanimelist"`

Parameters:
- **name**: Display name for the library
- **path**: Absolute path to the library folder
- **type**: Library type (`book` or `manga`)
- **provider**: Metadata provider (`myanimelist`, `googlebooks`, `comicvine`)
- **apiKey**: Optional API key for the metadata provider (required for comicvine)

#### ./devourer-server scan-library :libraryId

Scan the specified library for new content.

Example: `./devourer-server scan-library 1`

#### ./devourer-server scan-status :libraryId

Retrieve the scan status of the specified library.

Example: `./devourer-server scan-status 1`
