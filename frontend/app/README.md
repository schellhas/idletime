# idletime React app

This is the frontend for `idletime`.

## Run locally

```bash
cd /home/a/repos/idletime/frontend/app
cp .env.example .env.local  # optional
npm install
npm run dev
```

The app expects the Go backend to be available at `http://localhost:8080` by default.

If needed, override the API base URL with:

```bash
VITE_API_BASE_URL=http://localhost:8080
```

## What it currently supports

- register
- email verification via `/verify-email?token=...`
- login/logout
- viewing the current authenticated user
- CRUD for categories
- CRUD for activities
- CRUD for time entries
