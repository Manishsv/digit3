-- Flyway Migration V1: Create initial schema
-- This migration creates the initial workflow database schema

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Table for Process Definitions
CREATE TABLE IF NOT EXISTS processes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(64) NOT NULL,
    name VARCHAR(128) NOT NULL,
    code VARCHAR(128) NOT NULL,
    description VARCHAR(512),
    version VARCHAR(32),
    sla BIGINT,
    created_by VARCHAR(64) NOT NULL,
    created_at BIGINT NOT NULL,
    modified_by VARCHAR(64) NOT NULL,
    modified_at BIGINT NOT NULL,
    UNIQUE(tenant_id, code)
);

-- Table for States within a Process
CREATE TABLE IF NOT EXISTS states (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(64) NOT NULL,
    process_id UUID NOT NULL REFERENCES processes(id) ON DELETE CASCADE,
    code VARCHAR(64) NOT NULL,
    name VARCHAR(128) NOT NULL,
    description VARCHAR(512),
    sla BIGINT,
    is_initial BOOLEAN DEFAULT FALSE,
    is_parallel BOOLEAN DEFAULT FALSE,
    is_join BOOLEAN DEFAULT FALSE,
    branch_states JSONB,
    created_by VARCHAR(64) NOT NULL,
    created_at BIGINT NOT NULL,
    modified_by VARCHAR(64) NOT NULL,
    modified_at BIGINT NOT NULL,
    UNIQUE(process_id, code)
);

-- Table for Attribute-based Guard Validations
CREATE TABLE IF NOT EXISTS attribute_validations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(64) NOT NULL,
    attributes JSONB,
    assignee_check BOOLEAN DEFAULT FALSE,
    created_by VARCHAR(64) NOT NULL,
    created_at BIGINT NOT NULL,
    modified_by VARCHAR(64) NOT NULL,
    modified_at BIGINT NOT NULL
);

-- Table for Actions (Transitions) between States
CREATE TABLE IF NOT EXISTS actions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(64) NOT NULL,
    name VARCHAR(64) NOT NULL,
    label VARCHAR(128),
    current_state_id UUID NOT NULL REFERENCES states(id) ON DELETE CASCADE,
    next_state_id UUID NOT NULL REFERENCES states(id) ON DELETE CASCADE,
    roles JSONB,
    attribute_validation_id UUID REFERENCES attribute_validations(id) ON DELETE SET NULL,
    created_by VARCHAR(64) NOT NULL,
    created_at BIGINT NOT NULL,
    modified_by VARCHAR(64) NOT NULL,
    modified_at BIGINT NOT NULL
);

-- Table for Process Instances (runtime tracking)
CREATE TABLE IF NOT EXISTS process_instances (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(64) NOT NULL,
    process_id UUID NOT NULL REFERENCES processes(id) ON DELETE RESTRICT,
    entity_id VARCHAR(256) NOT NULL,
    action VARCHAR(128),
    status VARCHAR(64) NOT NULL,
    comment VARCHAR(512),
    documents JSONB,
    assigner VARCHAR(64),
    assignees JSONB,
    current_state_id UUID NOT NULL REFERENCES states(id) ON DELETE RESTRICT,
    state_sla BIGINT,
    process_sla BIGINT,
    attributes JSONB,
    created_by VARCHAR(64) NOT NULL,
    created_at BIGINT NOT NULL,
    modified_by VARCHAR(64) NOT NULL,
    modified_at BIGINT NOT NULL,
    UNIQUE(tenant_id, entity_id, process_id)
); 