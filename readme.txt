idletime
========

Problem
-------

I have too many interests and too little time.

When a time slot finally opens up for me, there are 2 issues:

* I feel overwhelmed by all the things I could spend this time frame on.
* I don't feel progress in my interests because I only do one of them for a short time.

idletime addresses both.

Solution
--------

It lets you input all your interests and activities. It lets you assign how much you want to do each activity in relation to the others.
The app will tell you what activity to do right now. When you are done, it will add the time spent to that activity.
Then you can look at a graph and see your progress.

Example
-------

Setup
~~~~~

I have many interests I want to keep track of.

* For sports, I like to do: Swimming, Jogging, Gym.
* For self-care, I like to do: Reading, Bathing, Journaling.
* I want to learn: Spanish, Italian.

I want to do 2 times as much swimming as I do jogging. I also want to go to the gym 1.5 times as much as I go jogging.

So I input the following activities into the app:

* swimming = 2.0
* jogging = 1.0
* gym = 1.5

I can also assign "minimal_time" values to these activities. Which means: when I go swimming, I need at least 30 minutes to do so. Drive there, change, shower... So I could not go swimming in a 20-minute time window. The app needs to know that. So I assign swimming a "minimal_time" of 20 minutes.

Usage
~~~~~

When I now start the app and tell it the size of my time frame, it will recommend whatever activity I currently should do in order for my activities to stay in the relation that I assigned to them.

Now I can start a timer, and after I stop it, the app will automatically add this time to the activity. I can also choose to input the time manually, or, when I do not want to do this activity right now, to "skip" and be recommended a different activity.

Now, I do not know whether I want to do Sports, Self-Care, or Languages. Which is why earlier, I assigned each of these categories a multiplier as well. I assigned a 2.0 multiplier to Sports, a 1.0 multiplier to Self-Care, and a 0.3 multiplier to Languages. This way I do three times as much self-care as I learn languages, and twice as much sports as I do self-care.

Now I can tell the app to give me an activity out of all of these categories. It will then give me an activity.

Development
-----------

Tech stack: PostgreSQL, Go (backend), React + TypeScript + Vite (frontend).
idletime is a web app with a single responsive web interface for desktop and mobile.

Layout:

* backend/   Go REST API (net/http, pgx, goose migrations)
* frontend/  React single-page app (Vite)

Run everything with Docker (app on http://localhost:3000):

    cp .env.example .env
    make up

Run the parts locally for development:

    make db            # PostgreSQL via Docker
    make backend-run   # API on :8080 (needs DATABASE_URL from .env.example)
    make frontend-dev  # Vite dev server on :3000, proxies /api to the backend

The app
~~~~~~~

* Now: tell it how much time you have and it suggests what to do. "Start timer" times the
  activity on the server (a running timer survives reloads and works across devices),
  "Skip" suggests the next one, "Log time" adds time after the fact.
* Library: manage categories (nestable) and activities with their multipliers and minimum time.
* Progress: time per day, and target vs. actual share for every category and activity.

How the suggestion works: every category and activity has a multiplier, its weight among
its siblings (the subcategories and activities directly below the same category). The
time share it should get is its weight divided by the sum of its siblings' weights,
multiplied down the tree. So a category with x2 gets twice the time of a sibling with x1,
no matter how many activities it contains. The app descends from the top and always picks
the sibling with the least tracked time relative to its weight; ties go to the heavier
weight. Activities whose minimum time does not fit the slot, and skipped ones, are ignored.

User accounts
~~~~~~~~~~~~~

* Sign up with username, email and password; the account is activated through a link sent by email.
* Log in with username or email. Sessions use an HttpOnly cookie and last 14 days.
* Forgot your password? Request a reset link by email.
* The account page lets you change your username and password, download all your data as JSON
  and delete your account (all data is removed).
* Every user only sees their own data; ownership is enforced in the database.

In development every email lands in a local mailbox (Mailpit): http://localhost:8025
Without SMTP_HOST the backend only logs emails. For production set SMTP_*, APP_BASE_URL
and COOKIE_SECURE=true (see .env.example).

Run the tests:

    make test   # unit tests (backend runs in Go, frontend in Vitest)
    make e2e    # end-to-end checks against the running stack (after make up)

Run "make help" to list all targets.
