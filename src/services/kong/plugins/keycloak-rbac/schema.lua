local typedefs = require "kong.db.schema.typedefs"

return {
  name = "keycloak-rbac",
  fields = {
    { config = {
        type = "record",
        fields = {
          {
            keycloak_base_url = {
              type = "string",
              required = true,
              default = "http://localhost:8080",
              description = "Base URL of Keycloak server"
            }
          },
          {
            client_id = {
              type = "string",
              required = true,
              description = "Keycloak client ID for authorization requests"
            }
          },
          {
            timeout = {
              type = "number",
              default = 5000,
              description = "Timeout for Keycloak authorization calls in milliseconds"
            }
          },
          {
            cache_ttl = {
              type = "number",
              default = 300,
              description = "Cache TTL for authorization decisions in seconds"
            }
          }
        },
      },
    },
  },
} 