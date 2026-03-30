# Devourer (Server)

A library server providing a highly performant system to archive, scan and catalog your books, manga and comics.

> **Important:** Upon running the server for the first time, make note of the automatically generated username, password, and API key printed to the console. You'll need these to connect with a client.

## Features

- A fast library server written in Go.
- Simple REST API for all client interactions.
- SQLite-powered storage with WAL mode for concurrency.
- Supports multiple users with role-based access control.
- Retrieves relevant cover images and generates preview images for each archive.
- Automatically retrieves metadata from various providers.
- Supports libraries and collections. A library is either a collection of books or comics/manga; collections exist within libraries.
- Built-in SPA client served at `/client`.
- File system watcher for automatic library updates.
- Supports Windows, Linux and Mac.

### Manga / Comic Features

- Supports `.zip`, `.cbz`, `.rar`, `.cbr`, `.7z`, and `.cb7` archives.
- Streams selected archives back to the client.

### Book Features

- Supports EPUB, PDF, MOBI, DOCX, and more.
- OPDS 1.2 support for e-reader compatibility.
- Import existing libraries from Calibre.

---

## Binary Releases

See the [Releases page](#).

---

## Manual Install

1. Ensure you have [Go 1.24+](https://go.dev/dl/) installed.
2. Clone this repository and `cd` into the `server` folder.
3. Build the binary:
   ```
   make build
   ```
4. Run the server:
   ```
   ./bin/devourer serve
   ```

### Build Targets

| Command | Description |
|---|---|
| `make build` | Build for the current platform |
| `make build-windows` | Build for Windows x64 |
| `make build-linux` | Build for Linux x64 |
| `make build-linux-arm64` | Build for Linux ARM64 |
| `make build-macos-arm` | Build for macOS ARM64 |
| `make docker` | Build Docker image |
| `make clean` | Remove build output |

---

## Docker

A `Dockerfile` and `docker-compose.yml` are provided. The container exposes port `9024`.

```yaml
volumes:
  - ./data:/app/data
  - ./assets:/app/assets
  - /path/to/media:/media
```

---

## Configuration

The server is configured via environment variables:

| Variable | Default | Description |
|---|---|---|
| `PORT` | `9024` | HTTP port |
| `DATABASE_PATH` | `./devourer.db` | Path to SQLite database file |
| `ASSETS_PATH` | `./assets` | Path to static assets directory |
| `CLIENT_PATH` | `./client` | Path to SPA client directory |
| `PLUGINS_PATH` | `./plugins` | Path to metadata provider plugins |
| `MIGRATIONS_DIR` | `./migrations` | Path to database migration files |
| `UPLOAD_MAX_SIZE_MB` | `1024` | Maximum upload file size in MB |
| `UPLOAD_ALLOWED_EXTS` | See below | Comma-separated list of allowed extensions |

**Default allowed extensions:** `epub`, `pdf`, `mobi`, `docx`, `doc`, `rtf`, `html`, `txt`, `cbz`, `cbr`, `zip`, `rar`, `7z`, `cb7`

---

## How To Use

The server exposes a REST API on port `9024`. Or, use the Devourer client application, which is designed to work with this server.

> **Note:** Most endpoints require authentication via a Bearer token in the `Authorization` header. Exceptions are noted below.

---

## API Endpoints

### System

| Method | Endpoint | Description | Auth |
|---|---|---|---|
| `GET` | `/` | Root endpoint | No |
| `GET` | `/version` | Returns API version | No |
| `GET` | `/health` | Health check | No |
| `GET` | `/status` | Server status | Yes |

---

### Authentication & Users

#### `POST /login`
Authenticate and receive a JWT token. Subject to rate limiting.

```json
{ "username": "user", "password": "password" }
```

#### `POST /users`
Create a new user account.

```json
{ "username": "user", "password": "password", "role": "user" }
```

Required role: `create_user`

#### `GET /users`
List all user accounts.

#### `GET /roles`
Get the current authenticated user's roles and permissions.

#### `PATCH /user/:id`
Edit a user's role or password.

Parameters: `id` — user ID

#### `DELETE /user/:id`
Delete a user account.

Parameters: `id` — user ID

---

### Library Management

#### `GET /libraries`
Get a list of all libraries.

#### `POST /libraries`
Create a new library.

```json
{
  "name": "Manga",
  "path": "D:/Manga",
  "type": "manga",
  "metadata": {
    "provider": "jikan",
    "apiKey": "optional"
  }
}
```

Required role: `manage_library`

#### `GET /library/:id`
Retrieve details for the specified library.

Parameters: `id` — library ID

#### `PATCH /library/:id`
Update library details.

Parameters: `id` — library ID
Required role: `manage_library`

#### `DELETE /library/:id`
Delete the specified library and all its content.

Parameters: `id` — library ID
Required role: `manage_library`

#### `POST /library/:id/scan`
Trigger a scan of the specified library for new content.

Parameters: `id` — library ID
Required role: `add_file`

#### `GET /library/:id/scan`
Retrieve the current scan status for the specified library.

Parameters: `id` — library ID

#### `GET /recently-read`
Get recently read items across all libraries.

---

### Collections

#### `GET /library/:id/collections`
List all collections in a library.

Parameters: `id` — library ID

#### `POST /library/:id/collections`
Create a new collection.

```json
{ "name": "Action Manga" }
```

Parameters: `id` — library ID
Required role: `manage_collections`

#### `GET /library/:id/collections/:collectionId`
Retrieve a specific collection.

Parameters: `id` — library ID, `collectionId` — collection ID

#### `DELETE /collections/:collectionId`
Delete a collection.

Parameters: `collectionId` — collection ID
Required role: `manage_collections`

#### `PATCH /collections/:collectionId/:fileId`
Add a file to a collection.

Parameters: `collectionId` — collection ID, `fileId` — file ID
Required role: `manage_collections`

#### `DELETE /collections/:collectionId/:fileId`
Remove a file from a collection.

Parameters: `collectionId` — collection ID, `fileId` — file ID
Required role: `manage_collections`

---

### Series Management

#### `GET /series/:libraryId/:seriesId`
Retrieve details for a series.

Parameters: `libraryId`, `seriesId`

#### `GET /series/:libraryId/:seriesId/files`
List all files within a series.

Parameters: `libraryId`, `seriesId`

#### `PATCH /series/:libraryId/:seriesId/metadata`
Update series metadata.

Parameters: `libraryId`, `seriesId`
Required role: `edit_metadata`

#### `PATCH /series/:libraryId/:seriesId/cover`
Update the series cover image. Accepts a multipart file upload.

Parameters: `libraryId`, `seriesId`
Required role: `edit_metadata`

#### `PUT /series`
Create a new manga/comic series.

```json
{ "title": "One Piece", "path": "One Piece" }
```

Required role: `add_file`

---

### File Management

#### `PUT /book/:libraryId`
Upload a book to a library. Accepts a standard multipart form file upload.

Parameters: `libraryId`
Required role: `add_file`

#### `PUT /series/:libraryId/:seriesId/file`
Upload a file to a manga/comic series. Accepts a standard multipart form file upload.

Parameters: `libraryId`, `seriesId`
Required role: `add_file`

#### `GET /file/:libraryId/:id`
Retrieve details for a file.

Parameters: `libraryId`, `id` — file ID

#### `GET /stream/:libraryId/:id`
Stream or download a file. This endpoint does not require authentication.

Parameters: `libraryId`, `id` — file ID

#### `POST /file/:libraryId/:id/scan`
Rescan a file to extract or refresh metadata.

Parameters: `libraryId`, `id` — file ID

#### `POST /file/:libraryId/:id/mark-as-read`
Mark the specified file as read.

Parameters: `libraryId`, `id` — file ID

#### `DELETE /file/:libraryId/:id/mark-as-read`
Mark the specified file as unread.

Parameters: `libraryId`, `id` — file ID

#### `POST /file/page-event`
Update reading progress for a file.

```json
{ "fileId": 1, "page": 5, "libraryId": 1 }
```

---

### Ratings

#### `POST /rate/:libraryId/:entityId`
Rate a series or file on a scale of 0–5.

Parameters: `libraryId`, `entityId` — series or file ID

```json
{ "rating": 4 }
```

---

### Tags

#### `GET /tag/:libraryId/:entityId`
List all user-defined tags for an entity.

Parameters: `libraryId`, `entityId`

#### `POST /tag/:libraryId/:entityId`
Create a tag on an entity.

Parameters: `libraryId`, `entityId`

```json
{ "tag": "favourite" }
```

#### `DELETE /tag/:libraryId/:entityId/:tag`
Remove a tag from an entity.

Parameters: `libraryId`, `entityId`, `tag` — tag name

---

### Metadata

#### `GET /metadata/providers`
List all available metadata providers.

#### `POST /metadata/search`
Search for metadata using a specific provider.

```json
{ "provider": "jikan", "by": "title", "value": "One Piece" }
```

---

### Calibre Migration

#### `POST /migrate/calibre`
Import an existing Calibre library into Devourer. Runs asynchronously.

```json
{ "path": "/path/to/calibre/library", "name": "My Books", "provider": "googlebooks" }
```

---

### Images

#### `GET /cover-image/:libraryId/:entityId`
Retrieve the cover image for a series or file.

Parameters: `libraryId`, `entityId` — series or file ID

#### `GET /preview-image/:libraryId/:seriesId/:entityId`
Retrieve the preview image for a specific file.

Parameters: `libraryId`, `seriesId`, `entityId` — file ID

---

### OPDS 1.2

The server provides full OPDS 1.2 support for e-reader compatibility (Kobo, Kindle, etc.).

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/opds/v1.2/catalog` | OPDS catalog root |
| `GET` | `/opds/v1.2/libraries` | List all libraries as OPDS feed |
| `GET` | `/opds/v1.2/library/:libraryId` | OPDS feed for a specific library |
| `GET` | `/opds/v1.2/library/:libraryId/search` | Search within a library (`?q=term`) |
| `GET` | `/opds/v1.2/library/:libraryId/book/:bookId` | Single book OPDS entry |

---

## Command Line Interface

The server binary supports the following subcommands:

### `serve`
Start the HTTP server and file watcher.

```
./devourer serve
```

### `create-library`
Create a new library and trigger an initial scan.

```
./devourer create-library --name "My Manga" --path "D:/Manga" --type manga --provider jikan
./devourer create-library --name "My Books" --path "D:/Books" --type book --provider googlebooks --api-key YOUR_KEY
```

| Flag | Required | Description |
|---|---|---|
| `--name` | Yes | Display name for the library |
| `--path` | Yes | Absolute path to the library folder |
| `--type` | Yes | Library type: `book` or `manga` |
| `--provider` | Yes | Metadata provider: `jikan`, `googlebooks`, `comicvine`, `openlibrary` |
| `--api-key` | No | API key for the provider (required for `comicvine`) |

### `scan-library`
Trigger a manual scan of an existing library.

```
./devourer scan-library --id 1
```

### `scan-status`
Check the scan status of a library.

```
./devourer scan-status --id 1
```

### `reset-password`
Reset the password for a user account.

```
./devourer reset-password --username admin --password newpassword
```

### `migrate-calibre`
Import an existing Calibre library.

```
./devourer migrate-calibre --path "D:/CalibreLibrary" --name "My Books" --provider googlebooks
```

---

## Metadata Providers

Providers are loaded from JSON plugin configs in `./plugins/providers/`. The following providers are included:

| Provider | ID | Use For |
|---|---|---|
| Jikan (MyAnimeList) | `jikan` | Manga |
| Google Books | `googlebooks` | Books |
| Open Library | `openlibrary` | Books |
| ComicVine | `comicvine` | Comics (API key required) |

---

## User Roles

| Role | Permissions |
|---|---|
| `admin` | All permissions |
| `moderator` | `add_file`, `delete_file`, `edit_metadata`, `manage_collections` |
| `upload` | `add_file` |
| `user` | Read-only |

**Available permissions:** `is_admin`, `add_file`, `delete_file`, `edit_metadata`, `manage_library`, `manage_collections`, `create_user`
