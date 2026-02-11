package handlers

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	localizationv1 "localization/api/proto/localization/v1"
	"localization/internal/core/domain"
	"localization/internal/core/ports"
)

// GRPCServer implements the gRPC server for the localization service
type GRPCServer struct {
	localizationv1.UnimplementedLocalizationServiceServer
	service ports.MessageService
}

// NewGRPCServer creates a new gRPC server
func NewGRPCServer(service ports.MessageService) *GRPCServer {
	return &GRPCServer{
		service: service,
	}
}

// getTenantIDFromContext extracts the tenant ID from the gRPC context metadata.
func getTenantIDFromContext(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.InvalidArgument, "missing metadata")
	}

	tenantID := md.Get("x-tenant-id")
	if len(tenantID) == 0 || tenantID[0] == "" {
		return "", status.Error(codes.InvalidArgument, "x-tenant-id header is required")
	}

	return tenantID[0], nil
}

// getUserIDFromContext extracts the user ID from the gRPC context metadata.
func getUserIDFromContext(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	userID := md.Get("x-user-id")
	if len(userID) == 0 {
		return ""
	}
	return userID[0]
}

// SearchMessages implements the SearchMessages gRPC endpoint
func (s *GRPCServer) SearchMessages(ctx context.Context, req *localizationv1.SearchMessagesRequest) (*localizationv1.SearchMessagesResponse, error) {
	tenantID, err := getTenantIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	var messages []domain.Message
	if len(req.Codes) > 0 {
		messages, err = s.service.SearchMessagesByCodes(ctx, tenantID, req.Locale, req.Codes)
	} else {
		messages, err = s.service.SearchMessages(ctx, tenantID, req.Module, req.Locale, int(req.Limit), int(req.Offset))
	}

	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to search messages: %v", err))
	}

	response := &localizationv1.SearchMessagesResponse{
		Messages: make([]*localizationv1.Message, len(messages)),
	}

	for i, msg := range messages {
		response.Messages[i] = &localizationv1.Message{
			Uuid:    msg.UUID,
			Code:    msg.Code,
			Message: msg.Message,
			Module:  msg.Module,
			Locale:  msg.Locale,
		}
	}

	return response, nil
}

// CreateMessages implements the CreateMessages gRPC endpoint
func (s *GRPCServer) CreateMessages(ctx context.Context, req *localizationv1.CreateMessagesRequest) (*localizationv1.CreateMessagesResponse, error) {
	tenantID, err := getTenantIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	userID := getUserIDFromContext(ctx)

	if len(req.Messages) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one message is required")
	}

	// Convert proto messages to domain messages
	domainMessages := make([]domain.Message, len(req.Messages))
	for i, msg := range req.Messages {
		domainMessages[i] = domain.Message{
			UUID:    msg.Uuid,
			Code:    msg.Code,
			Message: msg.Message,
			Module:  msg.Module,
			Locale:  msg.Locale,
		}
	}

	messages, err := s.service.CreateMessages(ctx, tenantID, userID, domainMessages)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to create messages: %v", err))
	}

	response := &localizationv1.CreateMessagesResponse{
		Messages: make([]*localizationv1.Message, len(messages)),
	}

	for i, msg := range messages {
		response.Messages[i] = &localizationv1.Message{
			Uuid:    msg.UUID,
			Code:    msg.Code,
			Message: msg.Message,
			Module:  msg.Module,
			Locale:  msg.Locale,
		}
	}

	return response, nil
}

// UpdateMessages implements the UpdateMessages gRPC endpoint
func (s *GRPCServer) UpdateMessages(ctx context.Context, req *localizationv1.UpdateMessagesRequest) (*localizationv1.UpdateMessagesResponse, error) {
	tenantID, err := getTenantIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	userID := getUserIDFromContext(ctx)

	if len(req.Messages) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one message is required")
	}

	// Convert proto messages to domain messages
	domainMessages := make([]domain.Message, len(req.Messages))
	for i, msg := range req.Messages {
		domainMessages[i] = domain.Message{
			UUID:    msg.Uuid,
			Message: msg.Message,
		}
	}

	messages, err := s.service.UpdateMessages(ctx, tenantID, userID, domainMessages)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to update messages: %v", err))
	}

	response := &localizationv1.UpdateMessagesResponse{
		Messages: make([]*localizationv1.Message, len(messages)),
	}

	for i, msg := range messages {
		response.Messages[i] = &localizationv1.Message{
			Uuid:    msg.UUID,
			Code:    msg.Code,
			Message: msg.Message,
			Module:  msg.Module,
			Locale:  msg.Locale,
		}
	}

	return response, nil
}

// DeleteMessages implements the DeleteMessages gRPC endpoint
func (s *GRPCServer) DeleteMessages(ctx context.Context, req *localizationv1.DeleteMessagesRequest) (*localizationv1.DeleteMessagesResponse, error) {
	tenantID, err := getTenantIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if len(req.Uuids) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one uuid is required")
	}

	err = s.service.DeleteMessages(ctx, tenantID, req.Uuids)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to delete messages: %v", err))
	}

	return &localizationv1.DeleteMessagesResponse{
		Success: true,
	}, nil
}

// BustCache implements the BustCache gRPC endpoint
func (s *GRPCServer) BustCache(ctx context.Context, req *localizationv1.BustCacheRequest) (*localizationv1.BustCacheResponse, error) {
	tenantID, err := getTenantIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	err = s.service.BustCache(ctx, tenantID, req.Module, req.Locale)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to bust cache: %v", err))
	}

	return &localizationv1.BustCacheResponse{
		Message: "Cache bust operation completed successfully",
		Success: true,
	}, nil
}

// FindMissingMessages implements the FindMissingMessages gRPC endpoint
func (s *GRPCServer) FindMissingMessages(ctx context.Context, req *localizationv1.FindMissingMessagesRequest) (*localizationv1.FindMissingMessagesResponse, error) {
	tenantID, err := getTenantIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	missingMessages, err := s.service.FindMissingMessages(ctx, tenantID, req.Module)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find missing messages: %v", err)
	}

	// Convert the service response to the gRPC response format
	response := &localizationv1.FindMissingMessagesResponse{
		MissingMessagesByModule: make(map[string]*localizationv1.LocaleToMissingCodes),
	}

	for module, localeMap := range missingMessages {
		grpcLocaleMap := &localizationv1.LocaleToMissingCodes{
			Locales: make(map[string]*localizationv1.MissingCodes),
		}
		for locale, codes := range localeMap {
			grpcLocaleMap.Locales[locale] = &localizationv1.MissingCodes{Codes: codes}
		}
		response.MissingMessagesByModule[module] = grpcLocaleMap
	}

	return response, nil
}
