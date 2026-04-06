Backend
=======

This is the current Go backend foundation for ``idletime``.

Current status
--------------

As of 2026-04-06, the backend has been scaffolded and the first authentication foundation has been added.

What already exists:

* a Go HTTP server in ``cmd/server``
* PostgreSQL configuration via environment variables
* a Docker Compose setup for local Postgres
* an initial SQL schema in ``migrations/001_init.sql``
* multi-user auth tables for ``users``, ``sessions``, and ``email_verification_tokens``
* session-cookie authentication endpoints
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
* ``internal/auth/service.go``: registration, login, verification, sessions
* ``internal/mail/log_mailer.go``: development email logger
* ``migrations/001_init.sql``: current database schema

Auth endpoints currently wired
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

Development behavior
--------------------

In development, verification emails are **not** sent through a real provider yet.
Instead, the backend logs the verification URL through the log mailer, and non-production responses may also include the verification URL to simplify testing.

Runtime note
------------

The code has been added and the editor currently reports no backend errors.
However, runtime verification still needs to be done on a machine with ``go`` and ``docker`` installed.

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

* run and verify the backend end-to-end with a real local Go + Docker environment
* add tests for register/login/verify/logout flows
* add authenticated category and activity CRUD scoped by ``user_id``
* integrate a real email provider later if needed
* build the React frontend auth screens against these endpoints
