# Apps — Internal Tools Platform

**Date:** 2026-04-07
**Status:** In Progress

## Overview

Add an "Apps" section to ForgeBox where users can create internal tools powered by AI.
Each app runs inside an isolated Firecracker VM with access to granted tools (Database,
API, AI). The app's UI is accessible from the dashboard via an embedded iframe.

## Domain Model

### Entity: App (Aggregate Root)

```go
type AppRecord struct {
    ID          string    // UUID
    Name        string    // required, display name
    Description string    // optional, what the app does
    CreatedBy   string    // user ID of creator
    Sharing     string    // "personal" | "team" | "org"
    TeamID      string    // optional, for team-scoped apps
    Status      string    // "draft" | "deploying" | "running" | "stopped" | "error"
    Tools       string    // JSON array of granted tools: ["database", "api", "ai"]
    Config      string    // JSON blob for VM config, model settings, etc.
    URL         string    // iframe URL when running (set by scheduler)
    Enabled     bool      // soft-enable/disable
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

### Value Objects

- **AppStatus:** draft → deploying → running, running → stopped, any → error
- **AppTool:** database, api, ai
- **AppSharing:** personal, team, org (reuses existing pattern)

### Domain Events

- AppCreated, AppDeployed, AppStopped, AppDeleted

## API Design

### Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET    | /api/v1/apps | List apps (filtered by user/sharing) |
| POST   | /api/v1/apps | Create a new app |
| GET    | /api/v1/apps/{id} | Get app details |
| PUT    | /api/v1/apps/{id} | Update app |
| DELETE | /api/v1/apps/{id} | Delete app |

### Request/Response Types

**Create Request:**
```json
{
  "name": "Invoice Generator",
  "description": "Generates invoices from order data",
  "sharing": "team",
  "tools": ["database", "api", "ai"]
}
```

**App Response:**
```json
{
  "id": "uuid",
  "name": "Invoice Generator",
  "description": "Generates invoices from order data",
  "created_by": "user-id",
  "sharing": "team",
  "status": "draft",
  "tools": ["database", "api", "ai"],
  "config": "{}",
  "url": "",
  "enabled": true,
  "created_at": "2026-04-07T...",
  "updated_at": "2026-04-07T..."
}
```

## Frontend

### Routes

| Route | Page |
|-------|------|
| /apps | List all apps (card grid) |
| /apps/new | Create new app form |
| /apps/{id} | App detail — embedded iframe + controls |

### Components

- **Apps list page:** Card grid with status badges, tool icons, sharing badges
- **Create app page:** Form with name, description, tool selection, sharing
- **App detail page:** Embedded iframe (when running), status controls, config

### Navigation

Add "Apps" under the "Build" group in the sidebar, between Agents and Workflows.

## Architecture Decisions

1. Apps metadata stored in the same SQLite DB alongside automations/agents
2. App UI served from the VM — embedded via iframe with sandbox attributes
3. Status managed by the scheduler (future); for now, apps start as "draft"
4. Tools field stored as JSON array string in DB (same pattern as automation nodes)

## Test Strategy

- **Backend unit tests:** Domain validation, handler request/response
- **Backend integration tests:** Full CRUD lifecycle through HTTP
- **Frontend:** Manual testing of navigation, CRUD flows, empty states