# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Development Commands

This is a Maven multi-module project (Java 25, Spring Boot 3.5.10). Use the Maven wrapper (`./mvnw`).

### Full Build
```bash
./mvnw clean package        # Builds all modules (api → frontend → server)
```

### Backend Only
```bash
./mvnw -pl ragbag-server spring-boot:run -Dspring-boot.run.profiles=h2
```

### Frontend Only
```bash
cd ragbag-frontend && npm run dev       # Vite dev server
cd ragbag-frontend && npm run build     # Production build
cd ragbag-frontend && npm run check     # Type checking (svelte-check + tsc)
```

### Code Generation (from OpenAPI spec)
```bash
./mvnw -pl ragbag-api generate-sources  # Generates Spring interfaces + TypeScript fetch client
```
The OpenAPI spec lives at `ragbag-api/openapi.yaml`. Generation outputs:
- Spring controller interfaces → `ragbag-server/target/generated-sources` (package `de.serbroda.ragbag.generated.api` / `.model`)
- TypeScript fetch client → `ragbag-frontend/src/lib/api/gen/`

### Tests
```bash
./mvnw -pl ragbag-server test                                    # All server tests
./mvnw -pl ragbag-server test -Dtest=RagbagServerApplicationTests  # Single test class
```

### Formatting
```bash
./mvnw -pl ragbag-server spotless:apply   # Auto-format Java code
./mvnw -pl ragbag-server spotless:check   # Check formatting (runs at package phase)
```
Spotless enforces Palantir Java Format with tab indentation (4 spaces per tab).

## Architecture

### Module Structure
- **ragbag-api** — OpenAPI spec + code generation (both Spring server interfaces and TypeScript fetch client)
- **ragbag-frontend** — Svelte 5 + TypeScript SPA (Vite). Built by Maven via frontend-maven-plugin and copied into server's static resources.
- **ragbag-server** — Spring Boot backend. Depends on ragbag-api for generated interfaces.

### API-First Design
Controllers implement generated interfaces from the OpenAPI spec. When adding/changing endpoints:
1. Edit `ragbag-api/openapi.yaml`
2. Run code generation (`./mvnw -pl ragbag-api generate-sources`)
3. Implement the generated interface in the server controller

### Feature-Based Package Organization (`de.serbroda.ragbag`)
Code is organized by domain feature, not by technical layer:
- `auth/` — Authentication (login, register, JWT token generation)
- `user/` — User entity, repository, service, controller
- `space/` — Spaces (top-level containers) with member roles (OWNER, ADMIN, VIEWER)
- `collection/` — Hierarchical collections within spaces
- `bookmark/` — Bookmarks within collections
- `invite/` — Invitation system (token-based invites to spaces/collections)
- `security/` — Permission evaluation, SecurityUtils, UserDetailsService
- `shared/` — Base entity, exception handling, API constants

Each feature package contains its own Entity, Repository, Service, and Controller.

### Key Patterns
- **Entity base class**: All entities extend `AbstractBaseEntity` (UUID primary key, `@Version` optimistic locking, auto-managed `createdAt`/`updatedAt`)
- **Constructor injection**: Via Lombok `@RequiredArgsConstructor`
- **Security**: JWT (HS256) via Spring Security OAuth2 Resource Server. Token versioning for invalidation on logout. Method-level `@PreAuthorize` with custom `DomainPermissionEvaluator`.
- **Database migrations**: Liquibase changelogs in `ragbag-server/src/main/resources/db/changelog/`
- **Profiles**: default (H2 in-memory), `h2` (H2 TCP server with file-based DB at `data/`), `prod` (env-var-driven PostgreSQL/MariaDB)

### API Prefix
All REST endpoints are under `/api`. Constant defined in `shared/ApiConstants.java`.

### Swagger UI
Available at `/api/docs/swagger-ui` when the server is running.
