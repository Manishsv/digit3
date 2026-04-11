package audit

import (
	"context"
	"fmt"
)

type noopAuditor struct{}

func NewNoopAuditor() Auditor {
	return &noopAuditor{}
}

func (n *noopAuditor) LogDataEvent(ctx context.Context, event DataEvent) error {
	return nil
}

func (n *noopAuditor) LogSchemaEvent(ctx context.Context, event SchemaEvent) error {
	return nil
}

func (n *noopAuditor) VerifySignature(ctx context.Context, tenantID string, digest []byte, signature string) (bool, error) {
	return false, fmt.Errorf("signature verification not supported")
}
