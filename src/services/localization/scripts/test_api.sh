#!/bin/bash

# Set the base URL
BASE_URL="http://localhost:8080/api"

# Test tenant ID
TENANT_ID="test-tenant"
MODULE="test-module"
LOCALE="en"

echo "Testing Search Messages..."
curl -X GET "${BASE_URL}/localization/messages/v1/_search?tenantId=${TENANT_ID}&module=${MODULE}&locale=${LOCALE}"

echo -e "\n\nTesting Create Messages..."
curl -X POST "${BASE_URL}/localization/messages/v1/_create" \
  -H "Content-Type: application/json" \
  -d "{
    \"tenantId\": \"${TENANT_ID}\",
    \"messages\": [
      {
        \"code\": \"test.code\",
        \"message\": \"Test Message\",
        \"module\": \"${MODULE}\",
        \"locale\": \"${LOCALE}\"
      }
    ]
  }"

echo -e "\n\nTesting Update Messages..."
curl -X PUT "${BASE_URL}/localization/messages/v1/_update" \
  -H "Content-Type: application/json" \
  -d "{
    \"tenantId\": \"${TENANT_ID}\",
    \"locale\": \"${LOCALE}\",
    \"module\": \"${MODULE}\",
    \"messages\": [
      {
        \"code\": \"test.code\",
        \"message\": \"Updated Test Message\"
      }
    ]
  }"

echo -e "\n\nTesting Upsert Messages..."
curl -X PUT "${BASE_URL}/localization/messages/v1/_upsert" \
  -H "Content-Type: application/json" \
  -d "{
    \"tenantId\": \"${TENANT_ID}\",
    \"messages\": [
      {
        \"code\": \"test.code2\",
        \"message\": \"Another Test Message\",
        \"module\": \"${MODULE}\",
        \"locale\": \"${LOCALE}\"
      }
    ]
  }"

echo -e "\n\nTesting Delete Messages..."
curl -X DELETE "${BASE_URL}/localization/messages/v1/_delete" \
  -H "Content-Type: application/json" \
  -d "{
    \"tenantId\": \"${TENANT_ID}\",
    \"messages\": [
      {
        \"code\": \"test.code\",
        \"module\": \"${MODULE}\",
        \"locale\": \"${LOCALE}\"
      }
    ]
  }"

echo -e "\n\nTesting Bust Cache..."
curl -X DELETE "${BASE_URL}/localization/messages/cache-bust" 