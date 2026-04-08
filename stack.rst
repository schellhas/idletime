Technical Stack
===============

This file captures the current technical direction for ``idletime``.

Stack
-----

* **Frontend:** React (Vite) in ``frontend/app/``
* **Backend:** Go
* **Database:** PostgreSQL
* **API style:** REST to start

Current handoff status (2026-04-08)
-----------------------------------

* **Frontend runtime:** verified locally on ``http://localhost:3000``
* **Backend runtime:** verified locally on ``http://localhost:8080``
* **Authentication:** server-side sessions with an ``HttpOnly`` cookie and email verification
* **Current product slice:** authenticated CRUD for categories, activities, and time entries, plus a working recommendation flow, timer logging, and progress view
* **Library model:** every user gets a default internal ``root`` category that is shown in the frontend as ``Library``
* **Hierarchy support:** categories can be nested and are displayed as an expandable folder/file tree
* **Local dev resilience:** backend startup now auto-applies ``migrations/001_init.sql`` so schema changes are picked up automatically in development
