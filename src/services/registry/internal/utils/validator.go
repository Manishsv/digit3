package utils

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

type SchemaValidator struct {
	compiler *jsonschema.Compiler
}

func NewSchemaValidator() *SchemaValidator {
	compiler := jsonschema.NewCompiler()
	compiler.Draft = jsonschema.Draft2020
	return &SchemaValidator{
		compiler: compiler,
	}
}

func (sv *SchemaValidator) ValidateSchema(schemaDefinition json.RawMessage) error {
	var schemaMap map[string]interface{}
	if err := json.Unmarshal(schemaDefinition, &schemaMap); err != nil {
		return fmt.Errorf("invalid JSON schema: %w", err)
	}

	// Ensure the schema uses the correct draft version
	if draft, exists := schemaMap["$schema"]; exists {
		if draftStr, ok := draft.(string); ok {
			if draftStr != "https://json-schema.org/draft/2020-12/schema" {
				return fmt.Errorf("schema must use JSON Schema Draft 2020-12")
			}
		}
	} else {
		// Set the draft version if not present
		schemaMap["$schema"] = "https://json-schema.org/draft/2020-12/schema"
	}

	// Add the schema to the compiler
	schemaBytes, _ := json.Marshal(schemaMap)
	if err := sv.compiler.AddResource("schema.json", bytes.NewReader(schemaBytes)); err != nil {
		return fmt.Errorf("failed to add schema resource: %w", err)
	}

	// Compile the schema to validate it
	_, err := sv.compiler.Compile("schema.json")
	if err != nil {
		return fmt.Errorf("invalid schema definition: %w", err)
	}

	return nil
}

func (sv *SchemaValidator) ValidateData(data json.RawMessage, schemaDefinition json.RawMessage) error {
	var schemaMap map[string]interface{}
	if err := json.Unmarshal(schemaDefinition, &schemaMap); err != nil {
		return fmt.Errorf("invalid JSON schema: %w", err)
	}

	// Ensure the schema has the correct draft version
	if _, exists := schemaMap["$schema"]; !exists {
		schemaMap["$schema"] = "https://json-schema.org/draft/2020-12/schema"
	}

	// Create a new compiler for validation to avoid conflicts
	compiler := jsonschema.NewCompiler()
	compiler.Draft = jsonschema.Draft2020

	// Add the schema to the compiler
	schemaBytes, _ := json.Marshal(schemaMap)
	if err := compiler.AddResource("schema.json", bytes.NewReader(schemaBytes)); err != nil {
		return fmt.Errorf("failed to add schema resource: %w", err)
	}

	schema, err := compiler.Compile("schema.json")
	if err != nil {
		return fmt.Errorf("failed to compile schema: %w", err)
	}

	var dataMap interface{}
	if err := json.Unmarshal(data, &dataMap); err != nil {
		return fmt.Errorf("invalid JSON data: %w", err)
	}

	if err := schema.Validate(dataMap); err != nil {
		return fmt.Errorf("data validation failed: %w", err)
	}

	return nil
}