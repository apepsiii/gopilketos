# SMK NIBA E-Voting

A monolithic OSIS e-voting application built with Golang, Echo v5, SQLite, and Go HTML Templates. The project supports an admin dashboard, voter landing page, QR-based login, multi-step voting flow, audit logging, and WhatsApp notification via OneSender.

## Features

- Server-side rendering with Go HTML Templates
- SQLite database with schema migration
- Admin module with Basic Auth
  - Dashboard metrics
  - Candidate management (CRUD)
  - Voter management (DPT, UUID generation)
  - Audit logs with masked UUIDs
  - Announcement settings
- Voter module
  - Landing page with candidate profiles
  - Passwordless QR login scanner
  - Multi-step voting flow
  - Vote confirmation and final submission
- Asynchronous WhatsApp notification via OneSender
- Static assets for CSS and JavaScript

## Folder Structure

- `cmd/server/` - application entry point
- `config/` - application configuration (future use)
- `database/` - SQLite connection and migration logic
- `handlers/` - HTTP request handlers for voter and admin flows
- `models/` - data model definitions
- `routes/` - route registration (if extended)
- `services/` - external services integration (OneSender)
- `views/` - HTML templates for voter and admin pages
- `public/` - static assets (CSS, JS, uploads)

## Prerequisites

- Go `1.25.6`
- Git (optional)
- GCC toolchain / C compiler (required for `github.com/mattn/go-sqlite3` when CGO is enabled)

## Environment Variables

The app uses the following optional environment variables:

- `ADMIN_USER` - admin username (default: `admin`)
- `ADMIN_PASS` - admin password (default: `admin123`)
- `ONESENDER_API_URL` - OneSender API endpoint
- `ONESENDER_API_KEY` - OneSender API key

## Install Dependencies

In the project root, run:

```bash
cd /e/smk/go_pilketos
go mod tidy
```

## Run the Application

From the project root:

```bash
cd /e/smk/go_pilketos
CGO_ENABLED=1 go run ./cmd/server
```

Then open:

- `http://localhost:8080/` for the voter landing page
- `http://localhost:8080/admin` for the admin dashboard (Basic Auth required)

## Database

The SQLite database file is created at `database/evoting.db`. Database tables are auto-migrated at startup.

## Notes

- The admin module uses Basic Auth and defaults to `admin:admin123` if no environment variables are provided.
- The voting flow expects a valid UUID from the scanner and prevents double voting.
- Audit logs store masked UUIDs to preserve voter anonymity.
- OneSender notifications are sent asynchronously and do not block vote submission.
- On Windows or environments where Go disables CGO by default, run the application with `CGO_ENABLED=1` and ensure a C compiler is installed.

## Testing

Run the Go test suite with:

```bash
go test ./...
```

## Future Improvements

- Add file-based admin authentication or session login.
- Add candidate photo upload validation and storage cleanup.
- Add printable voter card generation.
- Add CSV import/export for voters and candidates.
- Add real-time charts on the admin dashboard.
