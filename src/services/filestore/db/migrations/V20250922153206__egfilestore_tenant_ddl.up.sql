CREATE TABLE IF NOT EXISTS eg_filestoremap_v2 (
    id bigint NOT NULL,
    filestoreid character varying(36) NOT NULL,
    filename character varying(100) NOT NULL,
    contenttype character varying(100),
    module character varying(256),
    tag character varying(256),
    tenantid character varying(256) not null,
    version bigint
);
CREATE SEQUENCE IF NOT EXISTS seq_eg_filestoremap_v2
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER TABLE eg_filestoremap_v2 ADD CONSTRAINT pk_filestoremap_v2 PRIMARY KEY (id); 
ALTER TABLE eg_filestoremap_v2 ADD CONSTRAINT uk_filestoremap_v2_filestoreid UNIQUE (filestoreid);
alter table eg_filestoremap_v2 add constraint uk_filestoremap_v2_fsid_tenant unique (filestoreid,tenantid);
