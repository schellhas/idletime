#!/usr/bin/env bash
# DEVELOPMENT ONLY. Creates (or resets) a test user with example data so the app can be
# tried out quickly. Run it with the stack up (docker compose up -d), as often as you like:
# every run deletes the test user and everything it owns, then creates it again.
#
#   login:     a
#   password:  123
#
# The app itself would reject these values (usernames need 3+ characters, passwords 10+).
# So the user row is written directly into the database. "a" is stored as the account's
# email address; the login accepts username or email, so typing "a" works.
# The categories, activities and time entries are created through the real API.
set -euo pipefail
cd "$(dirname "$0")"

TEST_LOGIN="a"
TEST_USERNAME="test-user"
TEST_PASSWORD="123"
# bcrypt (cost 12) of "123", so no hashing tool is needed here.
TEST_PASSWORD_HASH='$2a$12$Ekm.xHlL3evCbyJwQhoT7O4sGSvvpTDjG3.Sz5VSeGD8Y0fD1r1sm'
HISTORY_DAYS=45

# --- configuration: read the same .env as docker compose --------------------------------
env_value() { grep -E "^$1=" .env 2>/dev/null | tail -1 | cut -d= -f2- || true; }
PG_USER=$(env_value POSTGRES_USER); PG_USER=${PG_USER:-idletime}
PG_DB=$(env_value POSTGRES_DB); PG_DB=${PG_DB:-idletime}
BACKEND_PORT=$(env_value BACKEND_PORT); BACKEND_PORT=${BACKEND_PORT:-8080}
WEB_PORT=$(env_value WEB_PORT); WEB_PORT=${WEB_PORT:-3000}

BASE="http://localhost:${BACKEND_PORT}/api/v1"
ORIGIN="http://localhost:${WEB_PORT}"

if ! curl -fsS "http://localhost:${BACKEND_PORT}/readyz" >/dev/null 2>&1; then
  echo "The backend is not reachable on port ${BACKEND_PORT}. Start the stack first:" >&2
  echo "  docker compose up --build -d" >&2
  exit 1
fi

# --- 1. (re)create the user directly in the database -------------------------------------
docker compose exec -T db psql -v ON_ERROR_STOP=1 -q -U "$PG_USER" -d "$PG_DB" <<SQL
DELETE FROM users WHERE email = '${TEST_LOGIN}';
INSERT INTO users (username, email, password_hash, email_verified_at)
VALUES ('${TEST_USERNAME}', '${TEST_LOGIN}', '${TEST_PASSWORD_HASH}', now());
SQL

# --- 2. log in through the API and create the data ---------------------------------------
JAR=$(mktemp)
trap 'rm -f "$JAR"' EXIT
# The backend trusts X-Real-IP in the docker setup; a made-up value keeps the rate limiter out of the way.
api() { curl -fsS -b "$JAR" -H "X-Real-IP: 10.99.$((RANDOM % 250)).$((RANDOM % 250))" -H "Origin: ${ORIGIN}" -H "Content-Type: application/json" "$@"; }
create() { api -X POST "${BASE}$1" -d "$2" | python3 -c 'import json,sys; print(json.load(sys.stdin)["id"])'; }

curl -fsS -c "$JAR" -H "X-Real-IP: 10.99.0.1" -H "Origin: ${ORIGIN}" -H "Content-Type: application/json" \
  -X POST "${BASE}/auth/login" -d "{\"identifier\":\"${TEST_LOGIN}\",\"password\":\"${TEST_PASSWORD}\"}" >/dev/null

# Categories (the example from the readme) ...
SPORTS=$(create /categories '{"name":"Sports","multiplier":2}')
OUTDOORS=$(create /categories "{\"name\":\"Outdoors\",\"multiplier\":1,\"parentId\":${SPORTS}}")
CARE=$(create /categories '{"name":"Self-care","multiplier":1}')
LANGUAGES=$(create /categories '{"name":"Languages","multiplier":0.3}')
WORK=$(create /categories '{"name":"Side projects","multiplier":1.2}')

# ... and activities: name, category, multiplier, minimum minutes.
declare -A ACTIVITY
add_activity() { ACTIVITY["$1"]=$(create /activities "{\"name\":\"$1\",\"categoryId\":$2,\"multiplier\":$3,\"minimumMinutes\":$4}"); }
add_activity "Swimming"   "$SPORTS"    2.0 20
add_activity "Gym"        "$SPORTS"    1.5 30
add_activity "Jogging"    "$OUTDOORS"  1.0 0
add_activity "Hiking"     "$OUTDOORS"  0.5 90
add_activity "Reading"    "$CARE"      1.0 0
add_activity "Bathing"    "$CARE"      1.0 20
add_activity "Journaling" "$CARE"      1.0 0
add_activity "Spanish"    "$LANGUAGES" 1.0 0
add_activity "Italian"    "$LANGUAGES" 1.0 0
add_activity "Coding"     "$WORK"      1.0 45

# --- 3. time entries over the last weeks ---------------------------------------------------
# Deterministic pseudo-random history (same data on every run). Activities are drawn from a
# weighted list that roughly follows the targets but is deliberately a bit off, so the
# progress page shows some activities ahead and some behind.
RANDOM=42
DRAW=(Swimming Swimming Swimming Gym Gym Jogging Jogging Reading Reading Reading Reading Bathing Journaling
      Spanish Italian Coding Coding Coding Coding Hiking)
entries=0
for ((day = HISTORY_DAYS; day >= 1; day--)); do
  (( RANDOM % 100 < 30 )) && continue # rest day
  for ((n = 0; n < 1 + RANDOM % 2; n++)); do
    name=${DRAW[RANDOM % ${#DRAW[@]}]}
    minutes=$((15 + (RANDOM % 8) * 10))
    when=$(date -u -d "${day} days ago $((7 + RANDOM % 14)):$((RANDOM % 60))" +%Y-%m-%dT%H:%M:%SZ)
    api -X POST "${BASE}/time-entries" \
      -d "{\"activityId\":${ACTIVITY[$name]},\"minutes\":${minutes},\"loggedAt\":\"${when}\"}" >/dev/null
    entries=$((entries + 1))
  done
done

echo "Test data ready: 5 categories, ${#ACTIVITY[@]} activities, ${entries} time entries."
echo "Log in at ${ORIGIN} with  login: ${TEST_LOGIN}  password: ${TEST_PASSWORD}"
