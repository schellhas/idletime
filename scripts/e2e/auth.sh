#!/usr/bin/env bash
# End-to-end checks against a running stack (make up). Configure with environment variables:
#   BASE     API base URL, default http://localhost:8080/api/v1
#   MAIL     Mailpit API,  default http://localhost:8025/api/v1
#   ORIGIN   web app origin, default http://localhost:3000
# The checks send a made-up X-Real-IP per request so they do not trip the rate limiter,
# which requires TRUST_PROXY_HEADERS=true on the backend (docker compose sets it).
set -u
BASE=${BASE:-http://localhost:8080/api/v1}
MAIL=${MAIL:-http://localhost:8025/api/v1}
O="Origin: ${ORIGIN:-http://localhost:3000}"
J='Content-Type: application/json'
SUFFIX=$RANDOM
JAR_A=$(mktemp); JAR_B=$(mktemp)

ip() { echo "X-Real-IP: 10.1.$((RANDOM%250+1)).$((RANDOM%250+1))"; }
code() { curl -s -o /dev/null -w '%{http_code}' -H "$(ip)" "$@"; }
check() { # name expected actual
  if [ "$2" = "$3" ]; then echo "ok    $1"; else echo "FAIL  $1 (expected $2, got $3)"; FAILED=1; fi
}
latest_token() { # recipient -> token from the newest mail to that address
  python3 - "$1" "$MAIL" <<'PY'
import json, re, sys, urllib.request
to, mail = sys.argv[1], sys.argv[2]
msgs = json.load(urllib.request.urlopen(mail + "/messages"))["messages"]
for m in msgs:
    if any(t["Address"] == to for t in m["To"]):
        body = json.load(urllib.request.urlopen(f"{mail}/message/{m['ID']}"))["Text"]
        found = re.search(r"token=([A-Za-z0-9_-]+)", body)
        if found:
            print(found.group(1)); break
PY
}
FAILED=0

A_USER="alice$SUFFIX"; A_MAIL="alice$SUFFIX@example.com"; A_PW="correct horse battery"
B_USER="bob$SUFFIX";   B_MAIL="bob$SUFFIX@example.com"

check "register alice"            202 "$(code -X POST $BASE/auth/register -H "$O" -H "$J" -d "{\"username\":\"$A_USER\",\"email\":\"$A_MAIL\",\"password\":\"$A_PW\"}")"
check "register again, same email (no leak)" 202 "$(code -X POST $BASE/auth/register -H "$O" -H "$J" -d "{\"username\":\"other$SUFFIX\",\"email\":\"$A_MAIL\",\"password\":\"$A_PW\"}")"
check "username taken"            409 "$(code -X POST $BASE/auth/register -H "$O" -H "$J" -d "{\"username\":\"$A_USER\",\"email\":\"x$SUFFIX@example.com\",\"password\":\"$A_PW\"}")"
check "weak password rejected"    400 "$(code -X POST $BASE/auth/register -H "$O" -H "$J" -d "{\"username\":\"weak$SUFFIX\",\"email\":\"w$SUFFIX@example.com\",\"password\":\"short\"}")"
check "login before verify"       403 "$(code -X POST $BASE/auth/login -H "$O" -H "$J" -d "{\"identifier\":\"$A_USER\",\"password\":\"$A_PW\"}")"
check "protected route, no session" 401 "$(code $BASE/categories)"

sleep 1
TOKEN=$(latest_token "$A_MAIL")
check "verification mail has a token" 1 "$([ -n "$TOKEN" ] && echo 1 || echo 0)"
check "verify email"              204 "$(code -X POST $BASE/auth/verify-email -H "$O" -H "$J" -d "{\"token\":\"$TOKEN\"}")"
check "verify token is single-use" 400 "$(code -X POST $BASE/auth/verify-email -H "$O" -H "$J" -d "{\"token\":\"$TOKEN\"}")"
check "wrong password"            401 "$(code -X POST $BASE/auth/login -H "$O" -H "$J" -d "{\"identifier\":\"$A_USER\",\"password\":\"wrong wrong wrong\"}")"
check "login with email"          200 "$(code -c $JAR_A -X POST $BASE/auth/login -H "$O" -H "$J" -d "{\"identifier\":\"$A_MAIL\",\"password\":\"$A_PW\"}")"
echo "      cookie flags: $(grep idletime_session $JAR_A | awk '{print ($1 ~ /^#HttpOnly_/ ? "HttpOnly" : "NOT-HttpOnly")}')"

check "me"                        200 "$(code -b $JAR_A $BASE/me)"
check "create category"           201 "$(code -b $JAR_A -X POST $BASE/categories -H "$O" -H "$J" -d '{"name":"Sports","multiplier":2}')"
check "foreign Origin is blocked" 403 "$(code -b $JAR_A -X POST $BASE/categories -H 'Origin: https://evil.example' -H "$J" -d '{"name":"Evil"}')"

# second user must not see or use alice's data
curl -s -o /dev/null -H "$(ip)" -X POST $BASE/auth/register -H "$O" -H "$J" -d "{\"username\":\"$B_USER\",\"email\":\"$B_MAIL\",\"password\":\"$A_PW\"}"
sleep 1
curl -s -o /dev/null -H "$(ip)" -X POST $BASE/auth/verify-email -H "$O" -H "$J" -d "{\"token\":\"$(latest_token "$B_MAIL")\"}"
check "login bob"                 200 "$(code -c $JAR_B -X POST $BASE/auth/login -H "$O" -H "$J" -d "{\"identifier\":\"$B_USER\",\"password\":\"$A_PW\"}")"
check "bob sees no categories"    "[]" "$(curl -s -b $JAR_B $BASE/categories)"
check "bob cannot nest below alice's category" 422 "$(code -b $JAR_B -X POST $BASE/categories -H "$O" -H "$J" -d '{"name":"Hack","parentId":1}')"
check "bob can use the same name" 201 "$(code -b $JAR_B -X POST $BASE/categories -H "$O" -H "$J" -d '{"name":"Sports"}')"

check "rename"                    200 "$(code -b $JAR_A -X PATCH $BASE/me -H "$O" -H "$J" -d "{\"username\":\"alice2_$SUFFIX\"}")"
check "rename to taken name"      409 "$(code -b $JAR_A -X PATCH $BASE/me -H "$O" -H "$J" -d "{\"username\":\"$B_USER\"}")"
check "export"                    200 "$(code -b $JAR_A $BASE/me/export)"
echo "      export: $(curl -s -b $JAR_A $BASE/me/export | python3 -c 'import json,sys; d=json.load(sys.stdin); print(d["user"]["username"], len(d["categories"]), "categories")')"

check "change password, wrong current" 403 "$(code -b $JAR_A -X POST $BASE/me/password -H "$O" -H "$J" -d '{"currentPassword":"nope nope nope","newPassword":"another long password"}')"
check "change password"           204 "$(code -b $JAR_A -X POST $BASE/me/password -H "$O" -H "$J" -d "{\"currentPassword\":\"$A_PW\",\"newPassword\":\"another long password\"}")"
check "current session survives"  200 "$(code -b $JAR_A $BASE/me)"
check "old password rejected"     401 "$(code -X POST $BASE/auth/login -H "$O" -H "$J" -d "{\"identifier\":\"$A_MAIL\",\"password\":\"$A_PW\"}")"

# password reset
check "forgot password (known)"   202 "$(code -X POST $BASE/auth/forgot-password -H "$O" -H "$J" -d "{\"email\":\"$A_MAIL\"}")"
check "forgot password (unknown)" 202 "$(code -X POST $BASE/auth/forgot-password -H "$O" -H "$J" -d '{"email":"nobody@example.com"}')"
sleep 1
check "reset password"            204 "$(code -X POST $BASE/auth/reset-password -H "$O" -H "$J" -d "{\"token\":\"$(latest_token "$A_MAIL")\",\"password\":\"reset by email 123\"}")"
check "reset revoked old sessions" 401 "$(code -b $JAR_A $BASE/me)"
check "login with reset password" 200 "$(code -c $JAR_A -X POST $BASE/auth/login -H "$O" -H "$J" -d "{\"identifier\":\"$A_MAIL\",\"password\":\"reset by email 123\"}")"

check "logout"                    204 "$(code -b $JAR_A -X POST $BASE/auth/logout -H "$O")"
check "after logout"              401 "$(code -b $JAR_A $BASE/me)"

check "login again"               200 "$(code -c $JAR_A -X POST $BASE/auth/login -H "$O" -H "$J" -d "{\"identifier\":\"$A_MAIL\",\"password\":\"reset by email 123\"}")"
check "delete, wrong password"    403 "$(code -b $JAR_A -X DELETE $BASE/me -H "$O" -H "$J" -d '{"password":"nope"}')"
check "delete account"            204 "$(code -b $JAR_A -X DELETE $BASE/me -H "$O" -H "$J" -d '{"password":"reset by email 123"}')"
check "deleted user cannot login" 401 "$(code -X POST $BASE/auth/login -H "$O" -H "$J" -d "{\"identifier\":\"$A_MAIL\",\"password\":\"reset by email 123\"}")"

# rate limit: hammer login until 429
LIMITED=0; for i in $(seq 1 15); do [ "$(curl -s -o /dev/null -w '%{http_code}' -H "X-Real-IP: 10.9.9.9" -X POST $BASE/auth/login -H "$O" -H "$J" -d '{"identifier":"x","password":"y"}')" = 429 ] && LIMITED=1; done
check "login is rate limited"     1 "$LIMITED"

rm -f "$JAR_A" "$JAR_B"
[ "$FAILED" = 0 ] && echo "ALL OK" || echo "SOME CHECKS FAILED"
