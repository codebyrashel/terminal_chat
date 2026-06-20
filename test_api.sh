#!/bin/bash
BASE="http://localhost:8080"
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}======================================${NC}"
echo -e "${BLUE}  Terminal Chat - API Test${NC}"
echo -e "${BLUE}======================================${NC}"

echo -e "\n${GREEN}1. Register user Alice${NC}"
REG1=$(curl -s -X POST $BASE/api/register \
  -H "Content-Type: application/json" \
  -d '{"username":"testalice","password":"alice123"}')
echo "$REG1" | jq 2>/dev/null || echo "$REG1"

echo -e "\n${GREEN}2. Register user Bob${NC}"
REG2=$(curl -s -X POST $BASE/api/register \
  -H "Content-Type: application/json" \
  -d '{"username":"testbob","password":"bob123"}')
echo "$REG2" | jq 2>/dev/null || echo "$REG2"

echo -e "\n${GREEN}3. Login as Alice${NC}"
ALICE=$(curl -s -X POST $BASE/api/login \
  -H "Content-Type: application/json" \
  -d '{"username":"testalice","password":"alice123"}')
echo "$ALICE" | jq 2>/dev/null || echo "$ALICE"
ALICE_TOKEN=$(echo "$ALICE" | jq -r '.token')
ALICE_ID=$(echo "$ALICE" | jq -r '.id')

echo -e "\n${GREEN}4. Login as Bob${NC}"
BOB=$(curl -s -X POST $BASE/api/login \
  -H "Content-Type: application/json" \
  -d '{"username":"testbob","password":"bob123"}')
echo "$BOB" | jq 2>/dev/null || echo "$BOB"
BOB_TOKEN=$(echo "$BOB" | jq -r '.token')
BOB_ID=$(echo "$BOB" | jq -r '.id')

echo -e "\n${GREEN}5. Alice sends friend request to Bob${NC}"
REQ=$(curl -s -X POST $BASE/api/friends/request \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ALICE_TOKEN" \
  -d '{"username":"testbob"}')
echo "$REQ" | jq 2>/dev/null || echo "$REQ"

echo -e "\n${GREEN}6. Bob checks pending requests${NC}"
REQUESTS=$(curl -s $BASE/api/friends/requests \
  -H "Authorization: Bearer $BOB_TOKEN")
echo "$REQUESTS" | jq 2>/dev/null || echo "$REQUESTS"
REQUEST_ID=$(echo "$REQUESTS" | jq -r '.[0].id')

echo -e "\n${GREEN}7. Bob accepts friend request${NC}"
ACCEPT=$(curl -s -X POST $BASE/api/friends/accept \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $BOB_TOKEN" \
  -d "{\"request_id\":\"$REQUEST_ID\"}")
echo "$ACCEPT" | jq 2>/dev/null || echo "$ACCEPT"

echo -e "\n${GREEN}8. Alice checks friends list${NC}"
curl -s $BASE/api/friends \
  -H "Authorization: Bearer $ALICE_TOKEN" | jq

echo -e "\n${GREEN}9. Bob checks friends list${NC}"
curl -s $BASE/api/friends \
  -H "Authorization: Bearer $BOB_TOKEN" | jq

echo -e "\n${GREEN}10. Search for users (Bob searches 'test')${NC}"
curl -s "$BASE/api/users/search?q=test" \
  -H "Authorization: Bearer $BOB_TOKEN" | jq

echo -e "\n${GREEN}11. Bob removes Alice from friends${NC}"
curl -s -X POST $BASE/api/friends/remove \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $BOB_TOKEN" \
  -d "{\"friend_id\":\"$ALICE_ID\"}" | jq

echo -e "\n${GREEN}12. Verify Bob's friends list is empty${NC}"
curl -s $BASE/api/friends \
  -H "Authorization: Bearer $BOB_TOKEN" | jq

echo -e "\n${GREEN}13. Verify Alice's friends list is empty${NC}"
curl -s $BASE/api/friends \
  -H "Authorization: Bearer $ALICE_TOKEN" | jq

echo -e "\n${BLUE}======================================${NC}"
echo -e "${BLUE}  API Tests Complete${NC}"
echo -e "${BLUE}======================================${NC}"