Backend
=======

This is the current Go backend foundation for ``idletime``.

For the most up-to-date handoff checklist and progress tracker, also see ``progress_todo.rst``.
A basic React frontend now also exists in ``../frontend/app`` and talks to these endpoints.

Current status
--------------

As of 2026-04-06, the backend is now the first usable product slice: auth is in place, user-owned domain CRUD is live, and a first recommendation flow works end-to-end.

What already exists:

* a Go HTTP server in ``cmd/server``
* PostgreSQL configuration via environment variables
* a Docker Compose setup for local Postgres
* an initial SQL schema in ``migrations/001_init.sql``
* multi-user auth tables for ``users``, ``sessions``, and ``email_verification_tokens``
* session-cookie authentication endpoints
* authenticated CRUD for categories, activities, and time entries
* a v1 ``GET /recommendations`` endpoint that returns the single most-behind activity
* automatic creation/protection of the default ``root`` category for each user
* a development log mailer for verification links
* an auth design and continuation doc in ``authentication_plan.rst``

Code map
--------

Important backend locations:

* ``cmd/server/main.go``: application entry point
* ``internal/config``: environment/config loading
* ``internal/database``: PostgreSQL connection setup
* ``internal/httpapi/router.go``: base router and CORS setup
* ``internal/httpapi/auth.go``: auth HTTP handlers
* ``internal/httpapi/auth_middleware.go``: shared auth middleware and request-context user helper
* ``internal/httpapi/categories.go``: authenticated category CRUD handlers
* ``internal/httpapi/activities.go``: authenticated activity CRUD handlers
* ``internal/httpapi/time_entries.go``: authenticated time-entry CRUD handlers
* ``internal/auth/service.go``: registration, login, verification, sessions
* ``internal/mail/log_mailer.go``: development email logger
* ``migrations/001_init.sql``: current database schema

API endpoints currently wired
-----------------------------

* ``GET /``
* ``GET /health``
* ``GET /ready``
* ``POST /auth/register``
* ``POST /auth/verify-email``
* ``POST /auth/login``
* ``POST /auth/logout``
* ``GET /auth/me``
* ``POST /auth/resend-verification``
* ``GET /categories``
* ``POST /categories``
* ``GET /categories/{id}``
* ``PATCH /categories/{id}``
* ``DELETE /categories/{id}``
* ``GET /activities``
* ``POST /activities``
* ``GET /activities/{id}``
* ``PATCH /activities/{id}``
* ``DELETE /activities/{id}``
* ``GET /time-entries``
* ``POST /time-entries``
* ``GET /time-entries/{id}``
* ``PATCH /time-entries/{id}``
* ``DELETE /time-entries/{id}``
* ``GET /recommendations``

Development behavior
--------------------

In development, verification emails are **not** sent through a real provider yet.
Instead, the backend logs the verification URL through the log mailer, and non-production responses may also include the verification URL to simplify testing.

Runtime note
------------

The core auth and domain flow has now been runtime-verified locally:

* ``GET /health`` and ``GET /ready`` returned ``200 OK``
* register -> verify-email -> login -> ``/auth/me`` succeeded end-to-end
* authenticated category, activity, and time-entry routes were verified for user isolation behavior
* ``GET /recommendations`` returned a best-activity result from live tracked data
* the default ``root`` category is auto-created and protected from deletion

This verification was done in a user-space local environment. On a normal development machine, the standard ``go`` + ``docker compose`` workflow below is still the expected way to run the backend.

Quick start (when Go and Docker are available)
----------------------------------------------

1. Copy ``.env.example`` to ``.env`` if needed.
2. Start PostgreSQL:

   .. code-block:: bash

      docker compose down -v
      docker compose up -d postgres

3. Install dependencies and run the server:

   .. code-block:: bash

      go mod tidy
      go run ./cmd/server

Next recommended work
---------------------

The next AI or developer should pick up from here by focusing on the following:

* expand automated tests for register/login/verify/logout, session expiry, user isolation, and recommendation edge cases
* refine recommendation logic to consider available time windows and ``minimum_minutes`` more directly
* polish the React frontend flows and timer-oriented UX in ``../frontend/app``
* integrate a real email provider later if needed
* keep refining API structure as the protected route surface grows
