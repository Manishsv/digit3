-- Widen string columns (were VARCHAR(64)); long JWT sub / realm / template codes caused SQLSTATE 22001.
ALTER TABLE idgen_templates ALTER COLUMN templatecode TYPE VARCHAR(255);
ALTER TABLE idgen_templates ALTER COLUMN tenantid TYPE VARCHAR(255);
ALTER TABLE idgen_templates ALTER COLUMN createdby TYPE VARCHAR(255);
ALTER TABLE idgen_templates ALTER COLUMN lastmodifiedby TYPE VARCHAR(255);

ALTER TABLE idgen_sequence_resets ALTER COLUMN templatecode TYPE VARCHAR(255);
ALTER TABLE idgen_sequence_resets ALTER COLUMN tenantid TYPE VARCHAR(255);

ALTER TABLE idgen_sequence_lookup ALTER COLUMN tenantid TYPE VARCHAR(255);
ALTER TABLE idgen_sequence_lookup ALTER COLUMN templatecode TYPE VARCHAR(255);
