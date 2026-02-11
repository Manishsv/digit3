ALTER TABLE eg_filestoremap_v2 ADD COLUMN createdby character varying(64);
ALTER TABLE eg_filestoremap_v2 ADD COLUMN lastmodifiedby character varying(64);
ALTER TABLE eg_filestoremap_v2 ADD COLUMN createdtime bigint;
ALTER TABLE eg_filestoremap_v2 ADD COLUMN lastmodifiedtime bigint;
