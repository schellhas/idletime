Authentication and Multi-User Plan
==================================

Status
------

This document describes the authentication and multi-user design for ``idletime``.

As of 2026-04-06, the backend foundation for this flow has been added:

* auth routes exist
* the PostgreSQL schema includes users, sessions, and verification tokens
* session-cookie auth is wired in
* verification emails currently use a development log mailer
* the core auth flow has been runtime-verified locally for register, verify-email, login, and ``/auth/me``
* each user now gets a default ``root`` category, and protected domain CRUD is live on top of the auth layer

The system still needs broader automated tests, frontend polish, and additional production-readiness work.

Purpose
-------

The backend must support:

* multiple users with isolated data
* account registration with ``username``, ``email``, and ``password``
* email confirmation before the account is considered active
* login/logout and authenticated API access

Recommended decisions
---------------------

For the first version, use the following defaults:

* **database:** PostgreSQL
* **backend:** Go
* **password hashing:** ``argon2id`` preferred, ``bcrypt`` acceptable if simplicity is needed
* **auth strategy:** server-side sessions with an ``HttpOnly`` cookie
* **email verification:** one-time verification token with expiration
* **v1 scope:** local username/password accounts only; no OAuth/social login

Why sessions instead of JWT?
----------------------------

Use secure cookie-based sessions first.

Reasons:

* simpler revocation and logout
* easier server-side control
* better default fit for a React web app talking to one backend
* fewer pitfalls than rolling custom JWT behavior too early

Only switch to JWT later if there is a clear product need.

Ownership model
---------------

Every user-owned record must belong to a specific ``users.id``.

That means:

* categories must have ``user_id``
* activities must be owned by a user, either directly or through their category
* time entries must resolve back to the owning user
* every query in handlers, repositories, and services must scope by the authenticated ``user_id``

Never trust a frontend-provided user identifier. Always derive the active user from the authenticated session.

Database plan
-------------

Create at least these tables.

``users``
~~~~~~~~~

Suggested fields:

* ``id BIGSERIAL PRIMARY KEY``
* ``username TEXT NOT NULL UNIQUE``
* ``email TEXT NOT NULL UNIQUE``
* ``password_hash TEXT NOT NULL``
* ``email_verified_at TIMESTAMPTZ NULL``
* ``created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()``
* ``updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()``

``email_verification_tokens``
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Suggested fields:

* ``id BIGSERIAL PRIMARY KEY``
* ``user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE``
* ``token_hash TEXT NOT NULL UNIQUE``
* ``expires_at TIMESTAMPTZ NOT NULL``
* ``used_at TIMESTAMPTZ NULL``
* ``created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()``

``sessions``
~~~~~~~~~~~~

Suggested fields:

* ``id BIGSERIAL PRIMARY KEY``
* ``user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE``
* ``token_hash TEXT NOT NULL UNIQUE``
* ``expires_at TIMESTAMPTZ NOT NULL``
* ``created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()``
* ``revoked_at TIMESTAMPTZ NULL``
* optional metadata such as ``user_agent`` and ``ip_address``

Existing domain tables
~~~~~~~~~~~~~~~~~~~~~~

The existing application tables should be migrated to support ownership:

* add ``user_id`` to ``categories``
* ensure activity and time-entry access is constrained to the authenticated user
* add indexes on foreign keys used in per-user queries

Authentication flows
--------------------

Register
~~~~~~~~

``POST /auth/register``

Expected request:

.. code-block:: json

   {
     "username": "alice",
     "email": "alice@example.com",
     "password": "correct horse battery staple"
   }

Backend steps:

1. validate username, email, and password length/format
2. normalize email to lowercase
3. hash the password
4. insert the user as unverified
5. generate a random email verification token
6. store only the **hash** of that token in the database
7. send the verification link by email
8. return a success response without exposing sensitive details

Email verification
~~~~~~~~~~~~~~~~~~

Recommended endpoint:

``POST /auth/verify-email``

Flow:

1. user opens a link from the email
2. frontend extracts the token and sends it to the backend
3. backend hashes the provided token and looks it up
4. reject if expired, already used, or unknown
5. mark the user as verified by setting ``email_verified_at``
6. mark the token as used

Login
~~~~~

``POST /auth/login``

Backend steps:

1. find the user by email or username
2. compare the provided password against the stored password hash
3. reject invalid credentials with a generic error message
4. reject login if email is not verified
5. create a session token and store only its hash in ``sessions``
6. return the session via a secure cookie

Logout
~~~~~~

``POST /auth/logout``

Backend steps:

1. read the current session cookie
2. revoke or delete the corresponding session row
3. clear the cookie

Current user
~~~~~~~~~~~~

``GET /auth/me``

This endpoint should return the authenticated user profile needed by the frontend, for example:

* ``id``
* ``username``
* ``email``
* whether the email is verified

Resend verification
~~~~~~~~~~~~~~~~~~~

``POST /auth/resend-verification``

Use this when a registered but unverified user needs a new confirmation email.

Security requirements
---------------------

Passwords
~~~~~~~~~

* never store plain passwords
* use ``argon2id`` if feasible; otherwise ``bcrypt``
* enforce a reasonable minimum length
* do not log passwords or password hashes

Tokens
~~~~~~

* generate tokens from a cryptographically secure random source
* store **hashed** tokens in the database, not raw tokens
* make verification and session tokens expire
* treat tokens as one-time use where appropriate

Cookies
~~~~~~~

For session cookies:

* ``HttpOnly`` = true
* ``Secure`` = true in production
* ``SameSite=Lax`` as the default starting point
* choose a clear expiry and renewal policy

API hardening
~~~~~~~~~~~~~

Add these as the auth work progresses:

* rate limiting on register/login endpoints
* consistent generic error messages for login failures
* validation on all auth payloads
* audit-friendly logs without sensitive data

Email delivery plan
-------------------

Abstract email sending behind an interface so the rest of the auth flow does not depend on a specific provider.

Suggested behavior:

* in development, a logger-based mailer can print the verification link
* in production, use an email service such as Resend, Postmark, Mailgun, or SendGrid

Suggested backend structure
---------------------------

When this is implemented, keep auth code separated by responsibility:

* ``internal/auth`` for registration/login/session logic
* ``internal/middleware`` for authentication middleware
* ``internal/mail`` for email sending abstractions
* ``internal/store`` or repository packages for database access
* ``internal/httpapi`` for the HTTP handlers and route wiring

Implementation order and handoff status
--------------------------------------

The following items are already in place:

1. SQL support for ``users``, ``email_verification_tokens``, and ``sessions`` has been added
2. ``user_id`` ownership has been added to ``categories``
3. password hashing and token generation helpers have been added
4. register, verify-email, login, logout, resend-verification, and ``/auth/me`` routes have been wired
5. a development mailer logs verification links for local testing
6. authenticated category CRUD has been added and locally verified for user isolation behavior
7. authenticated activity and time-entry CRUD have been added and locally verified for user isolation behavior
8. shared auth middleware and request-context user lookup have been added for protected domain routes
9. a first recommendation endpoint is now wired on top of the user-owned domain model, and users get a default ``root`` category automatically

The next AI or developer should continue with these remaining steps:

1. expand automated tests for registration, login, verification, session expiry, broader user isolation, and recommendation edge cases
2. build and polish the React auth UI against the existing endpoints
3. integrate a real email provider when needed for non-development environments
4. refine the new recommendation logic with available-time awareness and better alternative suggestions
5. keep ``progress_todo.rst`` updated as milestones are completed
6. keep refining protected-route structure as the API surface grows

Guardrails for future AI work
-----------------------------

* do **not** store raw passwords
* do **not** trust ``user_id`` from the client
* do **not** allow unverified users to authenticate as fully active users unless the product explicitly changes that rule
* do **not** mix authentication concerns directly into unrelated handlers
* do update this document if the design changes materially

Out of scope for the first auth milestone
-----------------------------------------

These can come later:

* password reset flow
* change-email flow
* social login / OAuth
* multi-factor authentication
* admin roles and permissions

Definition of done for the first auth milestone
-----------------------------------------------

The first milestone should be considered fully complete only when all of the following are true:

* users can register with username, email, and password
* passwords are safely hashed
* email verification works end-to-end in a real runtime environment
* users can log in and log out
* authenticated requests resolve the current user from a session cookie
* user data is isolated by ``user_id`` in all relevant queries
* tests cover the main success and failure paths

Current note: the auth foundation plus user-scoped category, activity, and time-entry CRUD are now in place and have been runtime-verified locally. Shared auth middleware, a default ``root`` category, and a first recommendation flow are also in place, while broader automated test coverage and frontend polish are still outstanding.
