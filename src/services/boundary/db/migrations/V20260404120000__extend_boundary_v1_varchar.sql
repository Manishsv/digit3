-- boundary_v1 originally used VARCHAR(64) for several fields; long JWT sub, realm codes,
-- or boundary codes from account/provision can exceed that and cause SQLSTATE 22001.
ALTER TABLE boundary_v1 ALTER COLUMN tenantid TYPE VARCHAR(255);
ALTER TABLE boundary_v1 ALTER COLUMN code TYPE VARCHAR(255);
ALTER TABLE boundary_v1 ALTER COLUMN id TYPE VARCHAR(128);
ALTER TABLE boundary_v1 ALTER COLUMN createdby TYPE VARCHAR(255);
ALTER TABLE boundary_v1 ALTER COLUMN lastmodifiedby TYPE VARCHAR(255);
