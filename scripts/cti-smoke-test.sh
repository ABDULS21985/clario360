#!/usr/bin/env bash
# =============================================================================
# Clario 360 — CTI Module Smoke Test
# Tests all CTI endpoints against a running cyber-service via the API gateway.
# =============================================================================
set -euo pipefail

BASE_URL="${CTI_BASE_URL:-http://localhost:8080/api/v1/cyber/cti}"
IAM_URL="${IAM_URL:-http://localhost:8081}"
EMAIL="${CLARIO360_SMOKE_EMAIL:-admin@clario.dev}"
PASSWORD="${CLARIO360_SMOKE_PASSWORD:-Cl@rio360Dev!}"
RUN_ID="${CLARIO360_SMOKE_RUN_ID:-$(date +%s)}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

PASS=0
FAIL=0

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

check() {
  local label="$1"
  local expected_status="$2"
  local actual_status="$3"
  local body="$4"

  if [ "$actual_status" = "$expected_status" ]; then
    echo -e "  ${GREEN}✓${NC} $label (HTTP $actual_status)"
    PASS=$((PASS + 1))
  else
    echo -e "  ${RED}✗${NC} $label — expected $expected_status, got $actual_status"
    echo "    Body: $(echo "$body" | head -c 200)"
    FAIL=$((FAIL + 1))
  fi
}

api_get() {
  local url="$1"
  local resp
  resp=$(curl -s -w "\n%{http_code}" -H "Authorization: Bearer $TOKEN" "$url" 2>/dev/null)
  local status=$(echo "$resp" | tail -1)
  local body=$(echo "$resp" | sed '$d')
  echo "$status|$body"
}

api_post() {
  local url="$1"
  local data="$2"
  local resp
  resp=$(curl -s -w "\n%{http_code}" -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" "$url" -d "$data" 2>/dev/null)
  local status=$(echo "$resp" | tail -1)
  local body=$(echo "$resp" | sed '$d')
  echo "$status|$body"
}

api_put() {
  local url="$1"
  local data="$2"
  local resp
  resp=$(curl -s -w "\n%{http_code}" -X PUT -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" "$url" -d "$data" 2>/dev/null)
  local status=$(echo "$resp" | tail -1)
  local body=$(echo "$resp" | sed '$d')
  echo "$status|$body"
}

api_patch() {
  local url="$1"
  local data="$2"
  local resp
  resp=$(curl -s -w "\n%{http_code}" -X PATCH -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" "$url" -d "$data" 2>/dev/null)
  local status=$(echo "$resp" | tail -1)
  local body=$(echo "$resp" | sed '$d')
  echo "$status|$body"
}

api_delete() {
  local url="$1"
  local resp
  resp=$(curl -s -w "\n%{http_code}" -X DELETE -H "Authorization: Bearer $TOKEN" "$url" 2>/dev/null)
  local status=$(echo "$resp" | tail -1)
  local body=$(echo "$resp" | sed '$d')
  echo "$status|$body"
}

extract_id() {
  echo "$1" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('data',{}).get('id',''))" 2>/dev/null || echo ""
}

# ---------------------------------------------------------------------------
# 1. Authenticate
# ---------------------------------------------------------------------------

echo -e "${YELLOW}=== CTI Smoke Test ===${NC}"
echo ""
echo "Authenticating as $EMAIL..."

LOGIN_RESP=$(curl -s -X POST "$IAM_URL/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}" 2>/dev/null)

TOKEN=$(echo "$LOGIN_RESP" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('data',{}).get('access_token',d.get('access_token','')))" 2>/dev/null || echo "")

if [ -z "$TOKEN" ] || [ "$TOKEN" = "" ]; then
  echo -e "${RED}Failed to authenticate. Response:${NC}"
  echo "$LOGIN_RESP" | head -c 300
  echo ""
  exit 1
fi
echo -e "${GREEN}Authenticated successfully${NC}"
echo ""

# ---------------------------------------------------------------------------
# 2. Reference Data
# ---------------------------------------------------------------------------

echo -e "${YELLOW}--- Reference Data ---${NC}"

IFS='|' read -r status body <<< "$(api_get "$BASE_URL/severity-levels")"
check "List severity levels" "200" "$status" "$body"

IFS='|' read -r status body <<< "$(api_get "$BASE_URL/categories")"
check "List categories" "200" "$status" "$body"

IFS='|' read -r status body <<< "$(api_get "$BASE_URL/regions")"
check "List regions" "200" "$status" "$body"

IFS='|' read -r status body <<< "$(api_get "$BASE_URL/sectors")"
check "List sectors" "200" "$status" "$body"

IFS='|' read -r status body <<< "$(api_get "$BASE_URL/data-sources")"
check "List data sources" "200" "$status" "$body"

# ---------------------------------------------------------------------------
# 3. Threat Events CRUD
# ---------------------------------------------------------------------------

echo ""
echo -e "${YELLOW}--- Threat Events ---${NC}"

IFS='|' read -r status body <<< "$(api_post "$BASE_URL/events" '{
  "event_type": "attack_attempt",
  "title": "Smoke test: SSH brute force from 10.99.1.1 '${RUN_ID}'",
  "severity_code": "high",
  "confidence_score": 0.85,
  "origin_country_code": "ru",
  "origin_city": "Moscow",
  "ioc_type": "ip",
  "ioc_value": "10.99.1.1",
  "tags": ["smoke-test", "brute-force"]
}')"
check "Create threat event" "201" "$status" "$body"
EVENT_ID=$(extract_id "$body")

IFS='|' read -r status body <<< "$(api_get "$BASE_URL/events?page=1&per_page=5")"
check "List threat events (page 1)" "200" "$status" "$body"

IFS='|' read -r status body <<< "$(api_get "$BASE_URL/events?severity=critical&per_page=5")"
check "List threat events (severity=critical)" "200" "$status" "$body"

if [ -n "$EVENT_ID" ]; then
  IFS='|' read -r status body <<< "$(api_get "$BASE_URL/events/$EVENT_ID")"
  check "Get threat event" "200" "$status" "$body"

  IFS='|' read -r status body <<< "$(api_put "$BASE_URL/events/$EVENT_ID" '{"description": "Updated via smoke test"}')"
  check "Update threat event" "200" "$status" "$body"

  IFS='|' read -r status body <<< "$(api_get "$BASE_URL/events/$EVENT_ID/tags")"
  check "Get event tags" "200" "$status" "$body"

  IFS='|' read -r status body <<< "$(api_post "$BASE_URL/events/$EVENT_ID/tags" '{"tags":["extra-tag"]}')"
  check "Add event tags" "200" "$status" "$body"

  IFS='|' read -r status body <<< "$(api_post "$BASE_URL/events/$EVENT_ID/false-positive" '{}')"
  check "Mark event false positive" "200" "$status" "$body"

  IFS='|' read -r status body <<< "$(api_delete "$BASE_URL/events/$EVENT_ID")"
  check "Delete threat event" "204" "$status" "$body"
fi

# ---------------------------------------------------------------------------
# 4. Threat Actors CRUD
# ---------------------------------------------------------------------------

echo ""
echo -e "${YELLOW}--- Threat Actors ---${NC}"

IFS='|' read -r status body <<< "$(api_post "$BASE_URL/actors" '{
  "name": "SMOKE TEST APT '${RUN_ID}'",
  "actor_type": "state_sponsored",
  "sophistication_level": "advanced",
  "primary_motivation": "espionage",
  "risk_score": 75.0
}')"
check "Create threat actor" "201" "$status" "$body"
ACTOR_ID=$(extract_id "$body")

IFS='|' read -r status body <<< "$(api_get "$BASE_URL/actors?per_page=5")"
check "List threat actors" "200" "$status" "$body"

if [ -n "$ACTOR_ID" ]; then
  IFS='|' read -r status body <<< "$(api_get "$BASE_URL/actors/$ACTOR_ID")"
  check "Get threat actor" "200" "$status" "$body"

  IFS='|' read -r status body <<< "$(api_put "$BASE_URL/actors/$ACTOR_ID" '{"description": "Updated via smoke test"}')"
  check "Update threat actor" "200" "$status" "$body"

  IFS='|' read -r status body <<< "$(api_delete "$BASE_URL/actors/$ACTOR_ID")"
  check "Delete threat actor" "204" "$status" "$body"
fi

# ---------------------------------------------------------------------------
# 5. Campaigns CRUD
# ---------------------------------------------------------------------------

echo ""
echo -e "${YELLOW}--- Campaigns ---${NC}"

IFS='|' read -r status body <<< "$(api_post "$BASE_URL/campaigns" '{
  "campaign_code": "C-TEST-'${RUN_ID}'",
  "name": "SMOKE TEST CAMPAIGN '${RUN_ID}'",
  "status": "active",
  "severity_code": "high",
  "first_seen_at": "2026-04-01T00:00:00Z"
}')"
check "Create campaign" "201" "$status" "$body"
CAMPAIGN_ID=$(extract_id "$body")

IFS='|' read -r status body <<< "$(api_get "$BASE_URL/campaigns?per_page=5")"
check "List campaigns" "200" "$status" "$body"

if [ -n "$CAMPAIGN_ID" ]; then
  IFS='|' read -r status body <<< "$(api_get "$BASE_URL/campaigns/$CAMPAIGN_ID")"
  check "Get campaign" "200" "$status" "$body"

  IFS='|' read -r status body <<< "$(api_put "$BASE_URL/campaigns/$CAMPAIGN_ID" '{"description": "Updated via smoke test"}')"
  check "Update campaign" "200" "$status" "$body"

  IFS='|' read -r status body <<< "$(api_patch "$BASE_URL/campaigns/$CAMPAIGN_ID/status" '{"status": "monitoring"}')"
  check "Update campaign status" "200" "$status" "$body"

  # Campaign IOCs
  IFS='|' read -r status body <<< "$(api_post "$BASE_URL/campaigns/$CAMPAIGN_ID/iocs" '{
    "ioc_type": "domain",
    "ioc_value": "smoke-test-'${RUN_ID}'.example.net",
    "confidence_score": 0.90
  }')"
  check "Create campaign IOC" "201" "$status" "$body"

  IFS='|' read -r status body <<< "$(api_get "$BASE_URL/campaigns/$CAMPAIGN_ID/iocs")"
  check "List campaign IOCs" "200" "$status" "$body"

  IFS='|' read -r status body <<< "$(api_get "$BASE_URL/campaigns/$CAMPAIGN_ID/events")"
  check "List campaign events" "200" "$status" "$body"

  IFS='|' read -r status body <<< "$(api_delete "$BASE_URL/campaigns/$CAMPAIGN_ID")"
  check "Delete campaign" "204" "$status" "$body"
fi

# ---------------------------------------------------------------------------
# 6. Brand Abuse
# ---------------------------------------------------------------------------

echo ""
echo -e "${YELLOW}--- Brand Abuse ---${NC}"

IFS='|' read -r status body <<< "$(api_get "$BASE_URL/brands")"
check "List monitored brands" "200" "$status" "$body"

# Get first brand ID for abuse incident test
BRAND_ID=$(echo "$body" | python3 -c "import sys,json; d=json.load(sys.stdin); items=d.get('data',[]); print(items[0]['id'] if items else '')" 2>/dev/null || echo "")

if [ -n "$BRAND_ID" ]; then
  IFS='|' read -r status body <<< "$(api_post "$BASE_URL/brand-abuse" "{
    \"brand_id\": \"$BRAND_ID\",
    \"malicious_domain\": \"smoke-test-phish.example.net\",
    \"abuse_type\": \"credential_phishing\",
    \"risk_level\": \"high\"
  }")"
  check "Create brand abuse incident" "201" "$status" "$body"
  INCIDENT_ID=$(extract_id "$body")

  IFS='|' read -r status body <<< "$(api_get "$BASE_URL/brand-abuse?per_page=5")"
  check "List brand abuse incidents" "200" "$status" "$body"

  if [ -n "$INCIDENT_ID" ]; then
    IFS='|' read -r status body <<< "$(api_get "$BASE_URL/brand-abuse/$INCIDENT_ID")"
    check "Get brand abuse incident" "200" "$status" "$body"

    IFS='|' read -r status body <<< "$(api_patch "$BASE_URL/brand-abuse/$INCIDENT_ID/takedown-status" '{"status": "reported"}')"
    check "Update takedown status" "200" "$status" "$body"
  fi
fi

# ---------------------------------------------------------------------------
# 7. Dashboard
# ---------------------------------------------------------------------------

echo ""
echo -e "${YELLOW}--- Dashboard ---${NC}"

IFS='|' read -r status body <<< "$(api_get "$BASE_URL/dashboard/threat-map?period=7d")"
check "Global threat map (7d)" "200" "$status" "$body"

IFS='|' read -r status body <<< "$(api_get "$BASE_URL/dashboard/sectors?period=30d")"
check "Sector threat overview (30d)" "200" "$status" "$body"

IFS='|' read -r status body <<< "$(api_get "$BASE_URL/dashboard/executive")"
check "Executive dashboard" "200" "$status" "$body"

# ---------------------------------------------------------------------------
# 8. Admin
# ---------------------------------------------------------------------------

echo ""
echo -e "${YELLOW}--- Admin ---${NC}"

IFS='|' read -r status body <<< "$(api_post "$BASE_URL/admin/refresh-aggregations" '{}')"
check "Refresh aggregations" "200" "$status" "$body"

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------

echo ""
echo "============================================"
TOTAL=$((PASS + FAIL))
echo -e "Results: ${GREEN}$PASS passed${NC}, ${RED}$FAIL failed${NC} / $TOTAL total"
echo "============================================"

if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
