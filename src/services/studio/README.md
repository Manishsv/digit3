# Studio service (tenant configuration control plane)

`studio` is a tenant-admin control-plane service to configure services/modules by managing:

- service directory entries
- versioned configuration bundles
- provisioning jobs (audit trail)

It is designed to orchestrate DIGIT shared services (Registry, IdGen, MDMS, Workflow) and governance service.

## Local dev (docker-compose)

Exposes port `8080` inside the container.

