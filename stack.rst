Technical Stack
===============

This file captures the current technical direction for ``idletime``.

Stack
-----

* **Frontend:** React (Vite) in ``frontend/app/``
* **Backend:** Go
* **Database:** PostgreSQL
* **API style:** REST to start

Current handoff status (2026-04-06)
-----------------------------------

* **Frontend runtime:** verified locally on ``http://localhost:3000``
* **Backend runtime:** verified locally on ``http://localhost:8080``
* **Authentication:** server-side sessions with an ``HttpOnly`` cookie
* **Current product slice:** authenticated CRUD for categories, activities, and time entries, plus a first recommendation endpoint
* **Convenience rule:** every user gets a default ``root`` category so categories remain optional in practice
