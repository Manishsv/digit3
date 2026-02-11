CREATE TABLE IF NOT EXISTS eg_doc_metadata_v2 (
    id bigint NOT NULL,
    tenantId character varying(256) NOT NULL,
    type character varying(256) NOT NULL,
    code character varying(256) NOT NULL,
    allowedFormats Jsonb,
    minSize character varying(256),
    maxSize character varying(256),
    isSensitive boolean,
    description character varying(256) not null,
    isActive boolean,
    auditDetail Jsonb
);


ALTER TABLE eg_doc_metadata_v2 ADD CONSTRAINT pk_doc_metadata_v2 PRIMARY KEY (id); 
