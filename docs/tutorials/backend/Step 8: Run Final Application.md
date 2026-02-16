# Step 7: Run Final Application

It is time to test our completed application! Follow the steps on this page to test and run the application.

## Steps
To run and test our sample application, follow the below steps -

1 . Run the PGR service that we just created.
```java
mvn clean install
mvn spring-boot:run
```
If the application starts running that's great there are no compilation or runtime errors!

1. Hit access token
```bash
curl --location 'http://localhost:8095/keycloak/realms/AMARAVATI/protocol/openid-connect/token' \
--header 'Content-Type: application/x-www-form-urlencoded' \
--data-urlencode 'grant_type=password' \
--data-urlencode 'client_id=auth-server' \
--data-urlencode 'client_secret=changeme' \
--data-urlencode 'username=johndoe' \
--data-urlencode 'password=mypassword'
```
NOTE:
To upload your file use the below command (replace with your file):
```bash
curl --location 'http://localhost:8102/filestore/v1/files/upload' \
--header 'Content-Type: multipart/mixed' \
--form 'tenantId="AMARAVATI"' \
--form 'module="HCM-ADMIN-CONSOLE-CLIENT"' \
--form 'tag="test"' \
--form 'file=@"/Users/srujana/Downloads/Screenshot from 2026-02-04 17-09-55.png"'
```
NOTE: Replace the <FILESTORE_ID> below with the ID of uploaded file.

2. Hit the \_create API request to create a service request.
```bash
curl --location 'http://localhost:8083/citizen-service/create' \
--header 'Content-Type: application/json' \
--header 'x-tenant-id: SHILLONG' \
--header 'x-client-id: kuiyt' \
--data-raw '{
  "service": {
    "serviceCode": "PGR001",
    "description": "Garbage not collected",
    "accountId": "CITIZEN-123",
    "fileStoreId": <FILESTORE_ID>,
    "boundaryCode": "TESTgg010",
    "applicationStatus": "ACTIVE",
    "email": "srujana.dadi@egovernments.org",
    "mobile": "9876543210"
  },
  "workflow": {
    "action": "APPLY"
  }
  }'
```

3. Hit the \_search API request to search for the created service request.
```bash
curl --location 'http://localhost:8083/citizen-service/search?serviceRequestId=pgr-20260127-0003-WR' \
--header 'x-tenant-id: SHILLONG' \
--header 'X-client-id: iuyuy' \
--header 'Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCIgOiAiSldUIiwia2lkIiA6ICI2c2wtWUlBQUFjV3MtTlBhQTZRcTFYR2dZMnpySlZOZjVOVEJsTU1Vb0s0In0.eyJleHAiOjE3NjU0NjIyNzgsImlhdCI6MTc2NTQ1NTA3OCwianRpIjoiM2VmNTdjYmUtMjA4MC00YTIyLWFjNGQtYTliZTAzMTQzNGJmIiwiaXNzIjoiaHR0cHM6Ly9kaWdpdC1sdHMuZGlnaXQub3JnL2tleWNsb2FrL3JlYWxtcy9BQ0NPVU5ULTY3IiwiYXVkIjoiYWNjb3VudCIsInN1YiI6ImQ5MTIyOTAwLTc4NDQtNDIxOC1iYzJkLWUwM2Y0M2M4NjUzYyIsInR5cCI6IkJlYXJlciIsImF6cCI6ImF1dGgtc2VydmVyIiwic2lkIjoiZjA3YzIwZWEtZmQyNC00YmUyLTg3YWUtZjEwZGE2ZWViZmMwIiwiYWNyIjoiMSIsImFsbG93ZWQtb3JpZ2lucyI6WyIvKiJdLCJyZWFsbV9hY2Nlc3MiOnsicm9sZXMiOlsiQ1NSIiwiU1VQRVJVU0VSIiwib2ZmbGluZV9hY2Nlc3MiLCJ1bWFfYXV0aG9yaXphdGlvbiIsImRlZmF1bHQtcm9sZXMtYWNjb3VudC02NyJdfSwicmVzb3VyY2VfYWNjZXNzIjp7ImFjY291bnQiOnsicm9sZXMiOlsibWFuYWdlLWFjY291bnQiLCJtYW5hZ2UtYWNjb3VudC1saW5rcyIsInZpZXctcHJvZmlsZSJdfX0sInNjb3BlIjoicHJvZmlsZSBlbWFpbCIsImVtYWlsX3ZlcmlmaWVkIjp0cnVlLCJwcmVmZXJyZWRfdXNlcm5hbWUiOiJqb2huZG9lIiwiZW1haWwiOiJqb2huQGV4YW1wbGUuY29tIn0.sjchaQUZk-jh-kXt-qQRsjGVkJTGwUApNavZDPeQ-Hup4DPJgra49vafJyLXvB8Q07MfaWvCnyuXfdJRs6ybYPsDotXtuCu_DxnPRXqdEbCk3J6fEHLUFYdkd3kMunoFEna-xBJOX09ccPxe8ymLTMNzrKzPVL-b7I5QieumftQSlx9mXy-HCXjVUsTycqH9AD5eJZ9HywuKIfT19LJV0dZgZa5c9uUafeHUbGvh6TdMgS8D1R29_SIM6LcqQedn3qTwdYaGtLmn-yj2dKxYMJdYbx5uV2gMFOVhlU0s7nA7gnKHoBv-hibTStgJ-5GYugWZxd9QZiuGnUghv-y6Uw'
```

4. Hit \_update API request to update your service request or transition it further in the workflow by changing the actions.
```bash
curl --location 'http://localhost:8083/citizen-service/update' \
--header 'Content-Type: application/json' \
--header 'Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCIgOiAiSldUIiwia2lkIiA6ICJqdTVBb3g1Qzl2eWtoN0Zpa2I2cTNORXRRQ3NhcEZ5TzJvWi1YSmJtdzFjIn0.eyJleHAiOjE3Njk1MTc4MTMsImlhdCI6MTc2OTUxMDYxMywianRpIjoiZWNkZWU5OTEtNGYzMS00YzllLWIxM2MtMzg4Y2ZjMDFmZDM1IiwiaXNzIjoiaHR0cHM6Ly9kaWdpdC1sdHMuZGlnaXQub3JnL2tleWNsb2FrL3JlYWxtcy9TSElMTE9ORyIsImF1ZCI6ImFjY291bnQiLCJzdWIiOiI4MzcyNGVmMS04ZGUwLTQ5ZTYtOWQ1Mi1mZmJiZjMwZGU1Y2YiLCJ0eXAiOiJCZWFyZXIiLCJhenAiOiJhdXRoLXNlcnZlciIsInNpZCI6IjVmMzhhNTZkLTU3MGQtNDYwOS1hNGVkLTkxMWRhYjM3ODIyMCIsImFjciI6IjEiLCJhbGxvd2VkLW9yaWdpbnMiOlsiLyoiXSwicmVhbG1fYWNjZXNzIjp7InJvbGVzIjpbIkNTUiIsIkdSTyIsImRlZmF1bHQtcm9sZXMtc2hpbGxvbmciLCJTVVBFUlVTRVIiLCJvZmZsaW5lX2FjY2VzcyIsInVtYV9hdXRob3JpemF0aW9uIl19LCJyZXNvdXJjZV9hY2Nlc3MiOnsiYWNjb3VudCI6eyJyb2xlcyI6WyJtYW5hZ2UtYWNjb3VudCIsIm1hbmFnZS1hY2NvdW50LWxpbmtzIiwidmlldy1wcm9maWxlIl19fSwic2NvcGUiOiJwcm9maWxlIGVtYWlsIiwiZW1haWxfdmVyaWZpZWQiOnRydWUsInByZWZlcnJlZF91c2VybmFtZSI6ImpvaG5kb2UiLCJlbWFpbCI6ImpvaG5AZXhhbXBsZS5jb20ifQ.n1TydpYrjddG5UExwjF102Ns7cDsqRTkltl_g7irtBJF1bikoPzNin3P5UbWOP2kpDVOz_YIxReuBFn7jmTd4wmogMsownVi0xqFBb36YledslY2jMFe4HMZssoMGHo9YkfEhHP7wy_an6OjkI6TSPXiXGz2grzGIM0f9xj6BMkO_4beRhSNriuFgNucaJVbNat1q3so728-2WxAdCnQKS7DERAVByZcPjD8Ag2JZLm1-0JxlH81-T800fvAtwxOGuPu1f899q6DeYi_k8lIsKHFKEjPuWD7U4Kq9alnwwy4T7xh4HTTsLV5yo0CCiCu0cRBVyEJgpb4dX0hfoUVZw' \
--header 'x-tenant-id: SHILLONG' \
--header 'x-client-id: kuiyt' \
--data-raw '{
  "service": {
    "serviceCode": "PGR001",
    "serviceRequestId": "pgr-20260127-0003-WR",
    "description": "Garbage not collected",
    "accountId": "CITIZEN-123",
    "fileStoreId": <FILESTORE_ID>,
    "boundaryCode": "TESTgg010",
    "email": "srujana.dadi@egovernments.org",
    "mobile": "9876543210"
  },
  "workflow": {
    "action": "ASSIGN"
  }
  }'
```

If these tests pass fantastic, your have successfully built the PGR module on DIGIT 3.0!
