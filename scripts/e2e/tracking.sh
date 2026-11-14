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
JAR=$(mktemp); JAR_B=$(mktemp)
FAILED=0

ip() { echo "X-Real-IP: 10.2.$((RANDOM%250+1)).$((RANDOM%250+1))"; }
code() { curl -s -o /dev/null -w '%{http_code}' -H "$(ip)" "$@"; }
body() { curl -s -H "$(ip)" "$@"; }
check() { if [ "$2" = "$3" ]; then echo "ok    $1"; else echo "FAIL  $1 (expected $2, got $3)"; FAILED=1; fi; }
jq_() { python3 -c "import json,sys; d=json.load(sys.stdin); print($1)"; }
latest_token() {
  python3 - "$1" "$MAIL" <<'PY'
import json, re, sys, urllib.request
to, mail = sys.argv[1], sys.argv[2]
for m in json.load(urllib.request.urlopen(mail + "/messages"))["messages"]:
    if any(t["Address"] == to for t in m["To"]):
        text = json.load(urllib.request.urlopen(f"{mail}/message/{m['ID']}"))["Text"]
        f = re.search(r"token=([A-Za-z0-9_-]+)", text)
        if f: print(f.group(1)); break
PY
}
signup() { # name jar
  curl -s -o /dev/null -H "$(ip)" -X POST $BASE/auth/register -H "$O" -H "$J" -d "{\"username\":\"$1\",\"email\":\"$1@example.com\",\"password\":\"correct horse battery\"}"
  sleep 1
  curl -s -o /dev/null -H "$(ip)" -X POST $BASE/auth/verify-email -H "$O" -H "$J" -d "{\"token\":\"$(latest_token "$1@example.com")\"}"
  curl -s -o /dev/null -c "$2" -H "$(ip)" -X POST $BASE/auth/login -H "$O" -H "$J" -d "{\"identifier\":\"$1\",\"password\":\"correct horse battery\"}"
}
post() { curl -s -H "$(ip)" -b "$JAR" -X POST "$BASE$1" -H "$O" -H "$J" -d "$2"; }

signup "track$SUFFIX" "$JAR"
signup "other$SUFFIX" "$JAR_B"

# --- categories: nesting, edit, cycle protection
SPORTS=$(post /categories '{"name":"Sports","multiplier":2}' | jq_ 'd["id"]')
CARE=$(post /categories '{"name":"Self-care","multiplier":1}' | jq_ 'd["id"]')
LANG=$(post /categories '{"name":"Languages","multiplier":0.3}' | jq_ 'd["id"]')
OUTDOOR=$(post /categories "{\"name\":\"Outdoor\",\"parentId\":$SPORTS}" | jq_ 'd["id"]')
check "rename category"           200 "$(code -b $JAR -X PATCH $BASE/categories/$OUTDOOR -H "$O" -H "$J" -d '{"name":"Outdoors","multiplier":1.5}')"
check "cycle is refused"          400 "$(code -b $JAR -X PATCH $BASE/categories/$SPORTS -H "$O" -H "$J" -d "{\"parentId\":$OUTDOOR}")"
check "move to top level"         200 "$(code -b $JAR -X PATCH $BASE/categories/$OUTDOOR -H "$O" -H "$J" -d '{"parentId":null}')"
check "move back below sports"    200 "$(code -b $JAR -X PATCH $BASE/categories/$OUTDOOR -H "$O" -H "$J" -d "{\"parentId\":$SPORTS}")"

# --- activities
SWIM=$(post /activities "{\"categoryId\":$SPORTS,\"name\":\"Swimming\",\"multiplier\":2,\"minimumMinutes\":30}" | jq_ 'd["id"]')
JOG=$(post /activities "{\"categoryId\":$OUTDOOR,\"name\":\"Jogging\",\"multiplier\":1}" | jq_ 'd["id"]')
READ=$(post /activities "{\"categoryId\":$CARE,\"name\":\"Reading\"}" | jq_ 'd["id"]')
SPAN=$(post /activities "{\"categoryId\":$LANG,\"name\":\"Spanish\"}" | jq_ 'd["id"]')
check "duplicate activity name"   409 "$(code -b $JAR -X POST $BASE/activities -H "$O" -H "$J" -d "{\"categoryId\":$SPORTS,\"name\":\"swimming\"}")"
check "activity in foreign category" 422 "$(code -b $JAR_B -X POST $BASE/activities -H "$O" -H "$J" -d "{\"categoryId\":$SPORTS,\"name\":\"Hack\"}")"
check "invalid minimum"           400 "$(code -b $JAR -X POST $BASE/activities -H "$O" -H "$J" -d "{\"categoryId\":$SPORTS,\"name\":\"X\",\"minimumMinutes\":-5}")"
check "edit activity"             200 "$(code -b $JAR -X PATCH $BASE/activities/$SWIM -H "$O" -H "$J" -d '{"minimumMinutes":20}')"
check "other user cannot edit"    404 "$(code -b $JAR_B -X PATCH $BASE/activities/$SWIM -H "$O" -H "$J" -d '{"name":"Stolen"}')"
check "other user sees none"      "[]" "$(body -b $JAR_B $BASE/activities)"

# --- recommendation (nothing tracked: heaviest branch first)
REC=$(body -b $JAR "$BASE/recommendation")
check "first recommendation"      "Swimming" "$(echo "$REC" | jq_ 'd["activity"]["name"]')"
check "recommendation path"       "Sports" "$(echo "$REC" | jq_ '"/".join(d["path"])')"
check "time slot excludes swimming" "Jogging" "$(body -b $JAR "$BASE/recommendation?availableMinutes=10" | jq_ 'd["activity"]["name"]')"
check "skip moves on"             "Jogging" "$(body -b $JAR "$BASE/recommendation?skip=$SWIM" | jq_ 'd["activity"]["name"]')"
check "skip everything"           404 "$(code -b $JAR "$BASE/recommendation?skip=$SWIM,$JOG,$READ,$SPAN")"
check "scope to self-care"        "Reading" "$(body -b $JAR "$BASE/recommendation?categoryId=$CARE" | jq_ 'd["activity"]["name"]')"
check "unknown category"          404 "$(code -b $JAR "$BASE/recommendation?categoryId=999999")"

# --- time entries
check "log time manually"         201 "$(code -b $JAR -X POST $BASE/time-entries -H "$O" -H "$J" -d "{\"activityId\":$SWIM,\"minutes\":60,\"note\":\"pool\"}")"
check "log in the past"           201 "$(code -b $JAR -X POST $BASE/time-entries -H "$O" -H "$J" -d "{\"activityId\":$READ,\"minutes\":30,\"loggedAt\":\"$(date -u -d '2 days ago' +%Y-%m-%dT%H:%M:%SZ)\"}")"
check "no future entries"         400 "$(code -b $JAR -X POST $BASE/time-entries -H "$O" -H "$J" -d "{\"activityId\":$SWIM,\"minutes\":5,\"loggedAt\":\"$(date -u -d '2 hours' +%Y-%m-%dT%H:%M:%SZ)\"}")"
check "zero minutes"              400 "$(code -b $JAR -X POST $BASE/time-entries -H "$O" -H "$J" -d "{\"activityId\":$SWIM,\"minutes\":0}")"
check "foreign activity"          422 "$(code -b $JAR_B -X POST $BASE/time-entries -H "$O" -H "$J" -d "{\"activityId\":$SWIM,\"minutes\":5}")"
ENTRY=$(body -b $JAR "$BASE/time-entries?limit=1" | jq_ 'd[0]["id"]')
check "edit entry"                200 "$(code -b $JAR -X PATCH $BASE/time-entries/$ENTRY -H "$O" -H "$J" -d '{"minutes":45}')"
check "tracked minutes appear"    "45" "$(body -b $JAR $BASE/activities | jq_ '[a["trackedMinutes"] for a in d if a["name"]=="Swimming"][0]')"

# swimming 45 min (22.5 per weight), reading 30, spanish 0: spanish is furthest behind
check "recommendation adapts"     "Spanish" "$(body -b $JAR "$BASE/recommendation" | jq_ 'd["activity"]["name"]')"

# --- timer
check "no timer yet"              "None" "$(body -b $JAR $BASE/timer | jq_ 'd')"
check "start timer"               201 "$(code -b $JAR -X POST $BASE/timer -H "$O" -H "$J" -d "{\"activityId\":$JOG}")"
check "second timer refused"      409 "$(code -b $JAR -X POST $BASE/timer -H "$O" -H "$J" -d "{\"activityId\":$READ}")"
check "timer is running"          "$JOG" "$(body -b $JAR $BASE/timer | jq_ 'd["activityId"]')"
check "stop timer logs >= 1 min"  "True" "$(post /timer/stop '' | jq_ 'd["activityId"] == '$JOG' and d["minutes"] >= 1')"
check "stop without timer"        404 "$(code -b $JAR -X POST $BASE/timer/stop -H "$O")"
check "start and cancel"          201 "$(code -b $JAR -X POST $BASE/timer -H "$O" -H "$J" -d "{\"activityId\":$READ}")"
check "cancel"                    204 "$(code -b $JAR -X DELETE $BASE/timer -H "$O")"
check "foreign activity timer"    422 "$(code -b $JAR_B -X POST $BASE/timer -H "$O" -H "$J" -d "{\"activityId\":$SWIM}")"

# --- stats
STATS=$(body -b $JAR "$BASE/stats?days=7&tz=Europe/Berlin")
check "stats: days"               7 "$(echo "$STATS" | jq_ 'len(d["daily"])')"
check "stats: total"              "True" "$(echo "$STATS" | jq_ 'd["totalMinutes"] >= 75 and d["periodMinutes"] >= 75')"
check "stats: sports share target" "True" "$(echo "$STATS" | jq_ 'abs([s for s in d["shares"] if s["name"]=="Sports"][0]["targetShare"] - 2/3.3) < 0.001')"
check "stats: bad zone"           400 "$(code -b $JAR "$BASE/stats?tz=Nope/Nowhere")"
check "stats: bad days"           400 "$(code -b $JAR "$BASE/stats?days=0x")"
check "stats: one activity"       "True" "$(body -b $JAR "$BASE/stats?days=7&activityId=$SWIM" | jq_ 'd["periodMinutes"] == 45')"
check "stats: category incl. children" "True" "$(body -b $JAR "$BASE/stats?days=7&categoryId=$SPORTS" | jq_ 'd["periodMinutes"] >= 46')"
check "stats: other category"     "True" "$(body -b $JAR "$BASE/stats?days=7&categoryId=$CARE" | jq_ 'd["periodMinutes"] == 30')"
check "stats: scope keeps shares" "True" "$(body -b $JAR "$BASE/stats?days=7&activityId=$SWIM" | jq_ 'len(d["shares"]) == 8')"
check "stats: category and activity" 400 "$(code -b $JAR "$BASE/stats?categoryId=$SPORTS&activityId=$SWIM")"
check "stats: unknown activity"   404 "$(code -b $JAR "$BASE/stats?activityId=999999")"
check "stats: foreign activity"   404 "$(code -b $JAR_B "$BASE/stats?activityId=$SWIM")"

# --- export and deletion cascade
EXPORT=$(body -b $JAR $BASE/me/export)
check "export has everything"     "True" "$(echo "$EXPORT" | jq_ 'len(d["categories"])==4 and len(d["activities"])==4 and len(d["timeEntries"])>=3')"
check "delete entry"              204 "$(code -b $JAR -X DELETE $BASE/time-entries/$ENTRY -H "$O")"
check "delete category cascades"  204 "$(code -b $JAR -X DELETE $BASE/categories/$SPORTS -H "$O")"
check "activities below are gone" "False" "$(body -b $JAR $BASE/activities | jq_ 'any(a["name"] in ("Swimming","Jogging") for a in d)')"
check "delete activity"           204 "$(code -b $JAR -X DELETE $BASE/activities/$READ -H "$O")"
check "delete unknown activity"   404 "$(code -b $JAR -X DELETE $BASE/activities/$READ -H "$O")"

rm -f "$JAR" "$JAR_B"
[ "$FAILED" = 0 ] && echo "ALL OK" || echo "SOME CHECKS FAILED"
