Backend Progress and TODO
=========================

Purpose
-------

This document is the backend handoff and TODO tracker for ``idletime``.
It is meant to help both humans and future AIs quickly understand what is already working, what has been verified, and what should happen next.

Verified current state (2026-04-08)
-----------------------------------

The following has been verified in a real local runtime environment:

* ``GET /health`` returns ``200 OK``
* ``GET /ready`` returns ``200 OK`` with a working database connection
* ``POST /auth/register`` works
* ``POST /auth/verify-email`` works when given the token from the verification URL
* ``POST /auth/login`` works and creates a session cookie
* ``POST /auth/logout`` works
* ``GET /auth/me`` returns the authenticated user tied to that session
* ``GET /categories`` requires authentication
* authenticated users can create and list their own categories
* each user gets a default internal ``root`` category automatically, and that category cannot be deleted
* categories can be nested using ``parent_id`` and are returned correctly in the API
* new categories default under the user’s root/Library category when no explicit parent is chosen
* ``GET /activities`` and ``GET /time-entries`` require authentication
* authenticated users can create and manage their own activities and time entries
* one user cannot read or mutate another user's activities or time entries
* ``GET /recommendations`` returns the single best current activity for the authenticated user
* recommendation requests support category filtering and repeated skipping of already-rejected activities
* the React frontend in ``../frontend/app`` can display recommendations, run a timer, show progress, and browse the Library tree
* backend startup auto-applies the current SQL migration and successfully recovered a live schema mismatch during development

Notes:

* the verification flow is currently development-oriented and uses a log mailer
* in non-production responses, the backend may expose ``verification_url`` to simplify testing
* the current recommendation version intentionally ignores time-window filtering and only returns one best result
* runtime verification was performed locally using a user-space Go toolchain and PostgreSQL setup because Docker was not available in the execution environment

What is already implemented
---------------------------

Auth and infrastructure:

* Go HTTP server scaffold in ``cmd/server``
* environment-based configuration in ``internal/config``
* PostgreSQL connection setup in ``internal/database``
* automatic migration application on backend startup in ``internal/database`` + ``cmd/server``
* auth service in ``internal/auth``
* auth HTTP handlers in ``internal/httpapi/auth.go``
* shared auth middleware and request-context helper in ``internal/httpapi/auth_middleware.go``
* authenticated category CRUD handlers in ``internal/httpapi/categories.go``
* authenticated activity CRUD handlers in ``internal/httpapi/activities.go``
* authenticated time-entry CRUD handlers in ``internal/httpapi/time_entries.go``
* authenticated recommendation handler in ``internal/httpapi/recommendations.go``
* development log mailer in ``internal/mail``

Database schema currently includes:

* ``users``
* ``sessions``
* ``email_verification_tokens``
* ``categories``
* ``activities``
* ``time_entries``

Ownership direction already present:

* ``categories`` has a ``user_id`` foreign key and now also supports hierarchical ``parent_id`` links
* auth sessions resolve the current user from an ``HttpOnly`` cookie
* unverified users cannot log in
* each user gets a default ``root`` category as the top-level fallback bucket for uncategorized activities and top-level custom categories

Recommendation logic currently implemented:

* the API returns a single best recommendation, not a full ranked list
* the score is based on ``tracked_minutes / (category.multiplier * activity.multiplier)``
* ``exclude_activity_id`` / ``exclude_activity_ids`` enable repeated stateless skipping for v1
* category filters can target selected categories and their descendants
* available-time and ``minimum_minutes`` filtering are still future refinement work

What is not finished yet
------------------------

The backend now has the core pieces needed for the product goal of
"users authenticate and then can access only their own data".

Still missing or recommended next:

1. broader automated tests for auth success and failure cases, session expiry, deeper user isolation coverage, nested-category edge cases, and recommendation edge cases
2. recommendation v2 refinement: use available time windows and ``minimum_minutes`` more directly and possibly expose ranked alternatives
3. continued library-management UX polish on top of the current React scaffold in ``../frontend/app``
4. a real email provider for non-development environments
5. richer analytics, export/backup options, and continued API cleanup as the app surface expands

Recommended next implementation order
------------------------------------

1. expand tests for auth flows, session expiry, broader user isolation, nested-category behavior, and recommendation edge cases
2. refine the recommendation logic with available-time awareness and better alternative suggestions
3. polish the React frontend with better Library editing workflows and richer analytics on top of the current scaffold
4. integrate a real email provider if non-development email delivery is needed
5. keep this document updated as each milestone is completed
6. continue cleanup and reuse patterns as more protected endpoints are added

Guardrails for future work
--------------------------

* never trust a frontend-provided ``user_id``
* always derive the user from the authenticated session
* always scope data queries by ownership
* keep password and token handling server-side and hashed
* keep ``Secure`` cookies enabled in production

Helpful files
-------------

* ``readme.rst``: backend overview and quick start
* ``authentication_plan.rst``: auth design decisions and guardrails
* ``migrations/001_init.sql``: current schema
* ``internal/auth/service.go``: auth logic
* ``internal/httpapi/auth.go``: auth routes and cookie handling
* ``internal/httpapi/auth_middleware.go``: shared auth middleware and current-user context lookup
* ``internal/httpapi/categories.go``: authenticated category CRUD and ownership enforcement
* ``internal/httpapi/activities.go``: authenticated activity CRUD and ownership enforcement
* ``internal/httpapi/time_entries.go``: authenticated time-entry CRUD and tracked-minute updates
* ``internal/httpapi/recommendations.go``: current recommendation scoring and skip support
* ``internal/httpapi/router.go``: root router and health endpoints

Definition of the next backend milestone
----------------------------------------

The current backend ownership milestone should be considered complete when:

* a logged-in user can create, list, update, and delete their own categories
* activities and time entries are scoped to the authenticated user
* one user cannot read or mutate another user's data
* the default ``root`` category remains available as the fallback category
* the user can receive a v1 recommendation from their tracked data
* the main auth and user-isolation flows are covered by tests

Status: the current MVP milestone is functionally in place and locally verified; the next push should focus on recommendation refinement, more tests, and continued UX polish.
