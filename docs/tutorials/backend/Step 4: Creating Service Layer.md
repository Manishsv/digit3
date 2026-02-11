# Service Layer

The service layer is where the core business logic is implemented. DIGIT services follow a consistent lifecycle — enrichment, validation, persistence, and orchestration with platform services. This structure is shared across DIGIT services, making them easier to understand and maintain.

The Service Layer will look like this on completion: 

### 1. RegistryService.java
```java
package com.example.pgrown30.service;

import com.example.pgrown30.web.models.CitizenService;
import org.digit.services.registry.model.RegistryDataResponse;

/**
 * Service interface for Registry operations.
 * Registry acts as the persistence layer replacing database operations.
 */
public interface RegistryService {

    /**
     * Saves a PGR service to registry after enrichment and validation.
     * This replaces the database save operation.
     *
     * @param citizenService the citizen service to save
     * @return the saved citizen service with registry response data
     */
    CitizenService save(CitizenService citizenService);

    /**
     * Creates registry data for a PGR service (internal method).
     *
     * @param citizenService the citizen service for which to create registry data
     * @return the registry response from the registry service
     */
    RegistryDataResponse createPgrRegistryData(CitizenService citizenService);

    /**
     * Searches for a PGR service by service request ID using registry search.
     *
     * @param serviceRequestId the service request ID to search for
     * @return the registry response containing the service data
     */
    RegistryDataResponse searchPgrRegistryData(String serviceRequestId);

    /**
     * Searches for a PGR service by service request ID and converts to CitizenService.
     *
     * @param serviceRequestId the service request ID to search for
     * @param tenantId the tenant ID for additional filtering (optional)
     * @return the citizen service if found, null otherwise
     */
    CitizenService findByServiceRequestId(String serviceRequestId, String tenantId);

    /**
     * Updates a PGR service in registry using registryId.
     *
     * @param citizenService the citizen service to update
     * @param registryId the registry ID to update
     * @return the updated citizen service
     */
    CitizenService update(CitizenService citizenService, String registryId);

    /**
     * Updates registry data for a PGR service (internal method).
     *
     * @param citizenService the citizen service for which to update registry data
     * @param registryId the registry ID to update
     * @return the registry response from the registry service
     */
    RegistryDataResponse updatePgrRegistryData(CitizenService citizenService, String registryId);
}
```

### 2. ServiceService.java
```java
package com.example.pgrown30.service;

import com.example.pgrown30.web.models.ServiceResponse;
import com.example.pgrown30.web.models.ServiceWrapper;

import java.util.List;

public interface ServiceService {

    // create and update keep the wrapper + roles signature
    ServiceResponse createService(ServiceWrapper wrapper, List<String> roles);

    ServiceResponse updateService(ServiceWrapper wrapper, List<String> roles);

    ServiceResponse searchServicesById(String serviceRequestId, String tenantId);
}
```

### 3. ServiceServiceImpl.java
```java
package com.example.pgrown30.service.impl;

import com.example.pgrown30.client.NotificationServiceClient;
import com.example.pgrown30.client.WorkflowServiceClient;
import com.example.pgrown30.validation.CreateServiceValidator;
import com.example.pgrown30.validation.UpdateServiceValidator;
import com.example.pgrown30.web.models.AuditDetails;
import com.example.pgrown30.web.models.CitizenService;
import com.example.pgrown30.web.models.ServiceResponse;
import com.example.pgrown30.web.models.ServiceWrapper;
import org.digit.services.idgen.IdGenClient;
import org.digit.services.idgen.model.IdGenGenerateRequest;
import org.digit.services.workflow.model.WorkflowTransitionResponse;
import com.example.pgrown30.service.ServiceService;
import com.example.pgrown30.service.RegistryService;
import org.digit.services.registry.model.RegistryDataResponse;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;
// Removed @Transactional - using registry instead of database

import java.time.Instant;
import java.util.*;

@Slf4j
@Service
@RequiredArgsConstructor
public class ServiceServiceImpl implements ServiceService {

    private final IdGenClient idGenClient;
    private final WorkflowServiceClient workflowServiceClient;
    private final NotificationServiceClient notificationServiceClient;
    private final CreateServiceValidator createServiceValidator;
    private final UpdateServiceValidator updateServiceValidator;
    private final RegistryService registryService;


    @Value("${idgen.templateCode}")
    private String templateCode;

    // ----------------------- CREATE SERVICE -----------------------
    @Override
    public ServiceResponse createService(ServiceWrapper wrapper, List<String> roles) {
        CitizenService citizenService = wrapper.getService();
        log.debug("createService: incoming DTO = {}", citizenService);

        // Initialize new service (ID generation, audit details, default values)
        initializeNewService(citizenService);

        // Validate service creation
        createServiceValidator.validate(citizenService);

        // Start workflow
        WorkflowTransitionResponse wfResp = workflowServiceClient.startWorkflow(citizenService, "APPLY", roles);
        if (wfResp != null) {
            workflowServiceClient.updateCitizenServiceWithWorkflow(citizenService, wfResp);
        }

        // Save to registry (replaces database persistence)
        CitizenService saved = registryService.save(citizenService);
        log.debug("Saved citizen service to registry with id={}", saved.getServiceRequestId());

        // Prepare and return response
        return buildServiceResponse(saved, wfResp, wrapper.getWorkflow(), null);
    }

    // ----------------------- UPDATE SERVICE -----------------------
    @Override
    public ServiceResponse updateService(ServiceWrapper wrapper, List<String> roles) {
        CitizenService incoming = wrapper.getService();
        log.debug("updateService called for requestId={} tenant={}", incoming.getServiceRequestId(), incoming.getTenantId());

        String serviceRequestId = incoming.getServiceRequestId();
        String tenantId = incoming.getTenantId();

        // Validate service request ID
        updateServiceValidator.validateServiceRequestId(serviceRequestId);

        // Validate service exists
        CitizenService existing = updateServiceValidator.validateServiceExists(serviceRequestId, tenantId);

        log.debug("Found existing entity id={} status={}", existing.getServiceRequestId(), existing.getApplicationStatus());

        // Apply updates
        applyPartialUpdates(existing, incoming);

        // Handle workflow transition if requested
        WorkflowTransitionResponse workflowResp = workflowServiceClient.handleWorkflowTransition(wrapper, existing, roles);

        // Update using registry service
        CitizenService saved = registryService.update(existing, existing.getRegistryId());
        log.debug("Updated service in registry: id={}", saved.getServiceRequestId());

        // Prepare and return response
        String workflowAction = workflowResp != null ? workflowResp.getAction() : null;
        return buildServiceResponse(saved, workflowResp, wrapper.getWorkflow(), workflowAction);
    }

    // ----------------------- SEARCH SERVICE BY ID -----------------------
    @Override
    public ServiceResponse searchServicesById(String serviceRequestId, String tenantId) {
        log.debug("Searching for service with serviceRequestId: {} and tenantId: {}", serviceRequestId, tenantId);

        try {
            // Search using registry service
            CitizenService foundService = registryService.findByServiceRequestId(serviceRequestId, tenantId);
            
            if (foundService != null) {
                log.info("Found service with serviceRequestId: {}", serviceRequestId);
                
                // Build clean response with only services array
                return ServiceResponse.builder()
                        .services(List.of(foundService))
                        .serviceWrappers(Collections.emptyList())
                        .build();
            } else {
                log.warn("No service found with serviceRequestId: {} and tenantId: {}", serviceRequestId, tenantId);
                
                // Return empty response
                return ServiceResponse.builder()
                        .services(Collections.emptyList())
                        .serviceWrappers(Collections.emptyList())
                        .build();
            }
        } catch (Exception e) {
            log.error("Error searching for service with serviceRequestId: {} and tenantId: {}", 
                    serviceRequestId, tenantId, e);
            
            // Return error response with message in services array
            return ServiceResponse.builder()
                    .services(Collections.emptyList())
                    .serviceWrappers(Collections.emptyList())
                    .build();
        }
    }



    // ----------------------- HELPERS -----------------------
    /**
     * Initializes a new service with ID, audit details, and default values.
     */
    private void initializeNewService(CitizenService citizenService) {
        // Generate service request ID
        String serviceRequestId = generateServiceRequestId();
        citizenService.setServiceRequestId(serviceRequestId);

        // Set audit details
        AuditDetails auditDetails = createAuditDetails();
        citizenService.setAuditDetails(auditDetails);

        // Set default source if not provided
        setDefaultSource(citizenService);
    }

    /**
     * Generates a new service request ID using IdGen service.
     */
    private String generateServiceRequestId() {
        IdGenGenerateRequest request = IdGenGenerateRequest.builder()
                .templateCode(templateCode)
                .variables(Map.of("ORG", "pgr"))
                .build();

        log.info("Requesting ID from IdGen with templateCode={} and orgCode={}", templateCode, "pgr");
        return idGenClient.generateId(request);
    }

    /**
     * Creates audit details with current timestamp for both created and last modified times.
     */
    private AuditDetails createAuditDetails() {
        long now = Instant.now().toEpochMilli();
        AuditDetails auditDetails = new AuditDetails();
        auditDetails.setCreatedTime(now);
        auditDetails.setLastModifiedTime(now);
        return auditDetails;
    }

    /**
     * Sets default source to "Citizen" if not provided.
     */
    private void setDefaultSource(CitizenService citizenService) {
        if (citizenService.getSource() == null || citizenService.getSource().isBlank()) {
            citizenService.setSource("Citizen");
        }
    }

    /**
     * Builds and returns a service response with workflow information and notifications.
     */
    private ServiceResponse buildServiceResponse(CitizenService saved, WorkflowTransitionResponse workflowResp,
                                                  org.digit.services.workflow.model.Workflow workflow, String workflowAction) {
        // Enrich response with workflow information
        CitizenService responseDto = workflowServiceClient.enrichResponseWithWorkflow(saved, workflowResp);

        // Send notifications
        notificationServiceClient.sendNotificationIfNeeded(saved, workflowAction);

        // Build and return clean response with only services array
        return ServiceResponse.builder()
                .services(List.of(responseDto))
                .serviceWrappers(Collections.emptyList())
                .build();
    }

    private void applyPartialUpdates(CitizenService existing, CitizenService incoming) {
        if (incoming.getDescription() != null) existing.setDescription(incoming.getDescription());
        if (incoming.getAddress() != null) existing.setAddress(incoming.getAddress());
        if (incoming.getEmail() != null) existing.setEmail(incoming.getEmail());
        if (incoming.getMobile() != null) existing.setMobile(incoming.getMobile());

        if (incoming.getFileStoreId() != null && !incoming.getFileStoreId().equals(existing.getFileStoreId())) {
            updateServiceValidator.validateFileStore(existing, incoming.getFileStoreId());
        }

        if (incoming.getBoundaryCode() != null && !incoming.getBoundaryCode().equals(existing.getBoundaryCode())) {
            updateServiceValidator.validateBoundary(existing, incoming.getBoundaryCode());
        }

        // Individual validation commented out
        // Store individualId if provided and different from existing (without validation)
        if (incoming.getIndividualId() != null && !incoming.getIndividualId().equals(existing.getIndividualId())) {
            existing.setIndividualId(incoming.getIndividualId());
            // updateServiceValidator.validateIndividual(existing, incoming.getIndividualId());
        }

        // Audit details are automatically handled by @PreUpdate lifecycle callback in CitizenService entity
    }

    // Registry persistence is now handled directly in the save method

}
```

### 4. RegistryServiceImpl.java
```java
package com.example.pgrown30.service.impl;

import com.example.pgrown30.service.RegistryService;
import com.example.pgrown30.web.models.CitizenService;
import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.digit.services.registry.RegistryClient;
import org.digit.services.registry.model.RegistryData;
import org.digit.services.registry.model.RegistryDataResponse;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;

/**
 * Registry-based persistence implementation for PGR services.
 * This replaces database operations with registry service calls.
 */
@Slf4j
@Service
@RequiredArgsConstructor
public class RegistryServiceImpl implements RegistryService {

    private final RegistryClient registryClient;
    private final ObjectMapper objectMapper;

    @Value("${pgr.registry.schema-code}")
    private String pgrSchemaCode;

    @Override
    public CitizenService save(CitizenService citizenService) {
        log.info("Saving PGR service to registry: {}", citizenService.getServiceRequestId());

        // Create registry data with complete CitizenService object
        RegistryDataResponse response = createPgrRegistryData(citizenService);

        if (response != null && Boolean.TRUE.equals(response.getSuccess())) {
            try {
                // Convert the response data to JsonNode and extract registryId
                JsonNode dataNode = objectMapper.valueToTree(response.getData());
                String registryId = dataNode.get("registryId").asText();
                
                log.info("Successfully saved PGR service to registry: {}. Registry ID: {}", 
                        citizenService.getServiceRequestId(), registryId);
                
                // Store the registry ID in the citizen service for future searches
                citizenService.setRegistryId(registryId);
                
                return citizenService;
            } catch (Exception e) {
                log.error("Error extracting registry ID from response: {}", e.getMessage(), e);
                throw new RuntimeException("Failed to extract registry ID from response", e);
            }
        } else {
            log.error("Failed to save PGR service to registry: {}. Response: {}", 
                    citizenService.getServiceRequestId(), response);
            throw new RuntimeException("Failed to save PGR service to registry: " + 
                    (response != null ? response.getError() : "Unknown error"));
        }
    }

    @Override
    public RegistryDataResponse createPgrRegistryData(CitizenService citizenService) {
        log.debug("Creating registry data for PGR service: {}", citizenService.getServiceRequestId());

        try {
            // Configure ObjectMapper to exclude null values
            ObjectMapper registryMapper = objectMapper.copy();
            registryMapper.setSerializationInclusion(com.fasterxml.jackson.annotation.JsonInclude.Include.NON_NULL);
            
            // Convert CitizenService object to registry data, excluding null values
            RegistryData registryData = RegistryData.builder()
                    .version(1)
                    .data(registryMapper.valueToTree(citizenService))
                    .build();

            // Call registry service to persist the data
            RegistryDataResponse response = registryClient.createRegistryData(pgrSchemaCode, registryData);
            
            // Log the response
            log.info("Registry data creation response for PGR service {}: {}", 
                    citizenService.getServiceRequestId(), response);
            
            return response;
        } catch (Exception e) {
            log.error("Error creating registry data for PGR service: {}", 
                    citizenService.getServiceRequestId(), e);
            throw new RuntimeException("Failed to create registry data", e);
        }
    }

    @Override
    public RegistryDataResponse searchPgrRegistryData(String serviceRequestId) {
        log.debug("Searching registry data with service request ID: {}", serviceRequestId);

        try {
            // Call registry service to search the data by serviceRequestId using key-value search
            RegistryDataResponse response = registryClient.searchRegistryData(pgrSchemaCode, "serviceRequestId", serviceRequestId);
            
            log.info("Registry data search response for service request ID {}: Success={}, Data={}, Error={}", 
                    serviceRequestId, response != null ? response.getSuccess() : "null", 
                    response != null ? response.getData() : "null",
                    response != null ? response.getError() : "null");
            
            return response;
        } catch (Exception e) {
            log.error("Error searching registry data for service request ID: {}", serviceRequestId, e);
            throw new RuntimeException("Failed to search registry data", e);
        }
    }

    @Override
    public CitizenService findByServiceRequestId(String serviceRequestId, String tenantId) {
        log.debug("Searching for PGR service by serviceRequestId: {} and tenantId: {}", serviceRequestId, tenantId);

        try {
            RegistryDataResponse response = searchPgrRegistryData(serviceRequestId);
            
            if (response != null && Boolean.TRUE.equals(response.getSuccess()) && response.getData() != null) {
                // Convert the response data to JsonNode
                JsonNode responseDataNode = objectMapper.valueToTree(response.getData());
                
                // Debug: Log the entire response structure
                log.debug("Registry response structure: {}", responseDataNode.toString());
                
                // The registry returns an array of data, so we need to get the first element
                if (responseDataNode.isArray() && responseDataNode.size() > 0) {
                    JsonNode firstElement = responseDataNode.get(0);
                    log.debug("First element structure: {}", firstElement.toString());
                    
                    // Extract the actual data field from the first element
                    JsonNode actualDataNode = firstElement.get("data");
                    if (actualDataNode != null) {
                        log.debug("Data node found: {}", actualDataNode.toString());
                        CitizenService citizenService = objectMapper.treeToValue(actualDataNode, CitizenService.class);
                        
                        // Set the registry ID from the response
                        JsonNode registryIdNode = firstElement.get("registryId");
                        if (registryIdNode != null) {
                            citizenService.setRegistryId(registryIdNode.asText());
                        }
                        
                        log.info("Successfully found PGR service with serviceRequestId: {}", serviceRequestId);
                        return citizenService;
                    } else {
                        log.warn("No data field found in registry response for serviceRequestId: {}. Available fields: {}", 
                                serviceRequestId, firstElement.fieldNames());
                        
                        // Let's try to list all field names for debugging
                        StringBuilder fields = new StringBuilder();
                        firstElement.fieldNames().forEachRemaining(field -> fields.append(field).append(", "));
                        log.warn("Available fields in first element: {}", fields.toString());
                        
                        return null;
                    }
                } else {
                    log.warn("Registry response is not an array or is empty for serviceRequestId: {}. Response type: {}", 
                            serviceRequestId, responseDataNode.getNodeType());
                    return null;
                }
            } else {
                log.warn("No service found with serviceRequestId: {}. Success: {}, Data: {}", 
                        serviceRequestId, response != null ? response.getSuccess() : "null", 
                        response != null ? response.getData() : "null");
                return null;
            }
        } catch (Exception e) {
            log.error("Error searching for service with serviceRequestId: {}", serviceRequestId, e);
            throw new RuntimeException("Failed to search for service by serviceRequestId", e);
        }
    }

    @Override
    public CitizenService update(CitizenService citizenService, String registryId) {
        log.info("Updating PGR service in registry: {} with registryId: {}", 
                citizenService.getServiceRequestId(), registryId);

        // Update registry data with complete CitizenService object
        RegistryDataResponse response = updatePgrRegistryData(citizenService, registryId);

        if (response != null && Boolean.TRUE.equals(response.getSuccess())) {
            try {
                // Convert the response data to JsonNode and extract registryId
                JsonNode dataNode = objectMapper.valueToTree(response.getData());
                String updatedRegistryId = dataNode.get("registryId").asText();
                
                log.info("Successfully updated PGR service in registry: {}. Registry ID: {}", 
                        citizenService.getServiceRequestId(), updatedRegistryId);
                
                // Store the registry ID in the citizen service
                citizenService.setRegistryId(updatedRegistryId);
                
                return citizenService;
            } catch (Exception e) {
                log.error("Error extracting registry ID from update response: {}", e.getMessage(), e);
                throw new RuntimeException("Failed to extract registry ID from update response", e);
            }
        } else {
            log.error("Failed to update PGR service in registry: {}. Response: {}", 
                    citizenService.getServiceRequestId(), response);
            throw new RuntimeException("Failed to update PGR service in registry: " + 
                    (response != null ? response.getError() : "Unknown error"));
        }
    }

    @Override
    public RegistryDataResponse updatePgrRegistryData(CitizenService citizenService, String registryId) {
        log.debug("Updating registry data for PGR service: {} with registryId: {}", 
                citizenService.getServiceRequestId(), registryId);

        try {
            // Configure ObjectMapper to exclude null values
            ObjectMapper registryMapper = objectMapper.copy();
            registryMapper.setSerializationInclusion(JsonInclude.Include.NON_NULL);
            
            // Convert CitizenService object to registry data, excluding null values
            RegistryData registryData = RegistryData.builder()
                    .version(1) // Version will be handled by the registry client
                    .data(registryMapper.valueToTree(citizenService))
                    .build();

            // Call registry service to update the data
            RegistryDataResponse response = registryClient.updateRegistryData(pgrSchemaCode, registryData, registryId);
            
            // Log the response
            log.info("Registry data update response for PGR service {} with registryId {}: {}", 
                    citizenService.getServiceRequestId(), registryId, response);
            
            return response;
        } catch (Exception e) {
            log.error("Error updating registry data for PGR service: {} with registryId: {}", 
                    citizenService.getServiceRequestId(), registryId, e);
            throw new RuntimeException("Failed to update registry data", e);
        }
    }
}
```

The Service Layer is now ready!




