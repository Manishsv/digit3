local http = require "resty.http"
local cjson = require "cjson"

local KeycloakRbacHandler = {}

KeycloakRbacHandler.PRIORITY = 700  -- Lower priority than JWT plugin (1005) and header-enrichment (800)
KeycloakRbacHandler.VERSION = "2.0.0"

-- Function to convert base64url to base64
local function base64url_to_base64(str)
  -- Replace URL-safe characters
  str = str:gsub('-', '+'):gsub('_', '/')
  -- Add padding if needed
  local padding = 4 - (#str % 4)
  if padding ~= 4 then
    str = str .. string.rep('=', padding)
  end
  return str
end

-- Function to extract realm from JWT payload
local function extract_realm_from_payload(payload)
  if not payload then
    return nil
  end
  
  -- Extract realm from issuer like "http://localhost:8080/realms/acme-corp"
  if payload.iss then
    local realm = payload.iss:match("/realms/([^/]+)")
    return realm
  end
  
  return nil
end

-- Function to call Keycloak Authorization Services
local function check_keycloak_authorization(conf, token, realm, resource_path, method)
  kong.log.debug("Making authorization call to Keycloak for realm: " .. realm)
  
  local httpc = http.new()
  httpc:set_timeout(conf.timeout or 5000)
  
  -- Keycloak Authorization Services endpoint
  local authz_url = conf.keycloak_base_url .. "/realms/" .. realm .. "/protocol/openid-connect/token"
  
  kong.log.debug("Authorization URL: " .. authz_url)
  kong.log.debug("Checking access for path: " .. resource_path .. " method: " .. method)
  
  -- Use actual HTTP method as scope
  local scope = method:lower()  -- get, post, put, delete, upsert, etc.
  
  -- Let Keycloak handle the URL directly - no hardcoded mapping
  -- Keycloak will match this against its configured resources
  local auth_request_body = {
    grant_type = "urn:ietf:params:oauth:grant-type:uma-ticket",
    audience = conf.client_id,
    permission = resource_path .. "#" .. scope,
    permission_resource_format = "uri",
    response_mode = "decision"
  }
  
  -- Convert to URL-encoded form data
  local form_data = {}
  for key, value in pairs(auth_request_body) do
    table.insert(form_data, key .. "=" .. ngx.escape_uri(tostring(value)))
  end
  local body = table.concat(form_data, "&")
  
  kong.log.debug("Authorization request body: " .. body)
  kong.log.err("KEYCLOAK DEBUG: Making request to: " .. authz_url)
  kong.log.err("KEYCLOAK DEBUG: Request body: " .. body)
  kong.log.err("KEYCLOAK DEBUG: Token: " .. token:sub(1, 50) .. "...")
  
  local res, err = httpc:request_uri(authz_url, {
    method = "POST",
    headers = {
      ["Content-Type"] = "application/x-www-form-urlencoded",
      ["Authorization"] = "Bearer " .. token,
      ["Accept"] = "application/json"
    },
    body = body
  })
  
  if not res then
    kong.log.err("Failed to call Keycloak authorization: " .. tostring(err))
    return false, "Failed to call Keycloak authorization: " .. tostring(err)
  end
  
  kong.log.debug("Keycloak authorization response status: " .. res.status)
  kong.log.debug("Keycloak authorization response body: " .. (res.body or "empty"))
  kong.log.err("KEYCLOAK DEBUG: Response status: " .. res.status)
  kong.log.err("KEYCLOAK DEBUG: Response body: " .. (res.body or "empty"))
  
  -- Check response
  if res.status == 200 then
    -- Permission granted - Keycloak returned 200, so access is allowed
    kong.log.debug("Keycloak GRANTED access for " .. resource_path .. "#" .. scope .. " (HTTP 200)")
    return true, nil
  elseif res.status == 403 or res.status == 401 then
    -- Permission denied
    kong.log.debug("Keycloak DENIED access for " .. resource_path .. "#" .. scope)
    return false, "Access denied by Keycloak authorization service"
  else
    -- Error
    kong.log.err("Keycloak authorization service error: " .. res.status .. " - " .. (res.body or ""))
    return false, "Keycloak authorization service error: " .. res.status
  end
end

-- Replace dynamic path segments (ids, uuids, etc.) with a wildcard so the
-- authorization check happens against generalized routes.
local function sanitize_path_for_rbac(path)
  if not path or path == "" then
    return path
  end

  local segments = {}
  for segment in path:gmatch("[^/]+") do
    if segment:match("^%d+$") -- purely numeric ids
        or segment:match("^[0-9a-fA-F%-]+$") -- uuid / hex identifiers
    then
      table.insert(segments, "*")
    else
      table.insert(segments, segment)
    end
  end

  local sanitized = (path:sub(1, 1) == "/" and "/" or "") .. table.concat(segments, "/")
  if path:sub(-1) == "/" and sanitized ~= "/" then
    sanitized = sanitized .. "/"
  end

  return sanitized
end

function KeycloakRbacHandler:access(conf)
  kong.log.debug("Keycloak RBAC plugin access phase started")
  
  local payload = nil
  local token = nil
  
  -- First, try to get JWT payload from Kong context
  local jwt_token = ngx.ctx.authenticated_jwt_token or ngx.ctx.jwt_token or ngx.ctx.jwt
  if jwt_token and type(jwt_token) == "table" and jwt_token.payload then
    kong.log.debug("Found JWT payload in Kong context")
    payload = jwt_token.payload
  end
  
  -- Also get the raw token for authorization call
  local auth_header = kong.request.get_header("Authorization")
  if auth_header then
    token = auth_header:match("Bearer%s+(.+)")
  end
  
  if not payload then
    kong.log.debug("No JWT payload found, trying manual decoding")
    
    if token then
      -- Parse JWT manually to get payload
      local header_b64, payload_b64, signature = token:match("([^%.]+)%.([^%.]+)%.([^%.]+)")
      if payload_b64 then
        local json = require "cjson"
        
        -- Convert base64url to regular base64
        local payload_b64_regular = base64url_to_base64(payload_b64)
        
        local ok, payload_json = pcall(function()
          return ngx.decode_base64(payload_b64_regular)
        end)
        
        if ok and payload_json then
          local ok2, decoded_payload = pcall(json.decode, payload_json)
          if ok2 and decoded_payload then
            kong.log.debug("Successfully decoded JWT payload manually for RBAC")
            payload = decoded_payload
          end
        end
      end
    end
  end
  
  if not payload or not token then
    kong.log.debug("No JWT token found")
    return kong.response.exit(401, { message = "No JWT token provided" })
  end
  
  -- Extract realm from JWT payload
  local realm = extract_realm_from_payload(payload)
  if not realm then
    kong.log.err("Could not extract realm from JWT payload")
    return kong.response.exit(401, { message = "Invalid JWT - no realm found" })
  end
  
  kong.log.debug("Extracted realm: " .. realm)
  
  -- Get the requested resource path and method
  local original_resource_path = kong.request.get_path()
  local resource_path = sanitize_path_for_rbac(original_resource_path)
  local method = kong.request.get_method()
  
  kong.log.debug("Checking authorization for: " .. method .. " " .. resource_path)
  kong.log.debug("Original request path retained: " .. original_resource_path)
  
  -- Make authorization call to Keycloak
  local authorized, err = check_keycloak_authorization(conf, token, realm, resource_path, method)
  
  if not authorized then
    kong.log.debug("Keycloak authorization failed: " .. tostring(err))
    return kong.response.exit(403, { 
      message = "Access denied",
      details = err
    })
  end
  
  kong.log.debug("Keycloak RBAC plugin access phase completed - access granted by Keycloak")
end

return KeycloakRbacHandler 
