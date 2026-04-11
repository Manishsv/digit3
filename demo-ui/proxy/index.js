require('dotenv').config();

const express = require('express');
const cors = require('cors');
const { createProxyMiddleware } = require('http-proxy-middleware');

const app = express();

const port = Number(process.env.PORT || 3847);
const webOrigin = process.env.WEB_ORIGIN || '';
const allowAnyLocalhostOrigin = !webOrigin;

app.use(
  cors({
    origin: (origin, cb) => {
      if (!origin) return cb(null, true);
      if (allowAnyLocalhostOrigin) {
        if (/^http:\/\/(127\.0\.0\.1|localhost):\d+$/.test(origin)) return cb(null, true);
        return cb(new Error(`CORS blocked origin: ${origin}`));
      }
      return cb(null, origin === webOrigin);
    },
    credentials: true,
    allowedHeaders: ['Content-Type', 'Authorization', 'X-Tenant-ID', 'X-Client-ID'],
    methods: ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'OPTIONS'],
  }),
);

app.get('/api/health', (_req, res) => res.json({ status: 'ok' }));

const targets = {
  keycloak: process.env.KEYCLOAK_BASE_URL || 'http://127.0.0.1:8080',
  studio: process.env.STUDIO_BASE_URL || 'http://127.0.0.1:8107',
  governance: process.env.GOVERNANCE_BASE_URL || 'http://127.0.0.1:8098',
  coordination: process.env.COORDINATION_BASE_URL || 'http://127.0.0.1:8090',
  registry: process.env.REGISTRY_BASE_URL || 'http://127.0.0.1:8104',
  workflow: process.env.WORKFLOW_BASE_URL || 'http://127.0.0.1:8085',
  mdms: process.env.MDMS_BASE_URL || 'http://127.0.0.1:8099',
  idgen: process.env.IDGEN_BASE_URL || 'http://127.0.0.1:8100',
  boundary: process.env.BOUNDARY_BASE_URL || 'http://127.0.0.1:8093',
  account: process.env.ACCOUNT_BASE_URL || 'http://127.0.0.1:8094',
};

function mount(service, pathRewritePrefix) {
  const target = targets[service];
  if (!target) throw new Error(`Unknown proxy target ${service}`);
  app.use(
    `/api/${service}`,
    createProxyMiddleware({
      target,
      changeOrigin: true,
      secure: false,
      logLevel: process.env.PROXY_LOG_LEVEL || 'warn',
      pathRewrite: (path) => path.replace(new RegExp(`^/api/${pathRewritePrefix}`), ''),
      onProxyReq: (proxyReq, req) => {
        // Preserve tenant/client headers; normalize casing.
        if (req.headers['x-tenant-id']) proxyReq.setHeader('X-Tenant-ID', req.headers['x-tenant-id']);
        if (req.headers['x-client-id']) proxyReq.setHeader('X-Client-ID', req.headers['x-client-id']);
      },
    }),
  );
}

mount('keycloak', 'keycloak');
mount('studio', 'studio');
mount('governance', 'governance');
mount('coordination', 'coordination');
mount('registry', 'registry');
mount('workflow', 'workflow');
mount('mdms', 'mdms');
mount('idgen', 'idgen');
mount('boundary', 'boundary');
mount('account', 'account');

app.listen(port, () => {
  // eslint-disable-next-line no-console
  console.log(`demo-ui proxy listening on http://127.0.0.1:${port} (allowOrigin=${webOrigin})`);
});

