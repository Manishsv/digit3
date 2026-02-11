# Client Layer

To interact with platform capabilities, the service uses DIGIT client libraries. These clients abstract access to shared services such as registry, workflow, notification, and boundary management. From the service’s perspective, these are simple method calls — not complex integrations.

### 1. NotificationServiceClient.java
```java
package com.example.pgrown30.client;

import org.digit.services.notification.NotificationClient;
import org.digit.services.notification.model.SendEmailRequest;
import com.example.pgrown30.web.models.CitizenService;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Component;

import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * Wrapper class for notification-related operations.
 * Encapsulates notification client interactions to reduce clutter in service layer.
 */
@Slf4j
@Component
@RequiredArgsConstructor
public class NotificationServiceClient {

    private final NotificationClient notificationClient;

    /**
     * Sends email notification if the citizen service has an email address.
     *
     * @param citizenService  the citizen service
     * @param workflowAction  the workflow action (null for create, non-null for update)
     */
    public void sendNotificationIfNeeded(CitizenService citizenService, String workflowAction) {
        if (citizenService.getEmail() == null || citizenService.getEmail().isBlank()) {
            return;
        }

        Map<String, Object> emailPayload = createEmailPayload(citizenService, workflowAction);

        SendEmailRequest request = SendEmailRequest.builder()
                .version("v1")
                .templateId("my-template")
                .emailIds(List.of(citizenService.getEmail()))
                .enrich(false)
                .payload(emailPayload)
                .build();

        notificationClient.sendEmail(request);
        String notificationType = workflowAction != null ? "update" : "create";
        log.info("Triggered {} email notification for {}", notificationType, citizenService.getServiceRequestId());
    }

    /**
     * Creates the email payload for notifications.
     *
     * @param citizenService  the citizen service
     * @param workflowAction  the workflow action
     * @return map containing email payload data
     */
    private Map<String, Object> createEmailPayload(CitizenService citizenService, String workflowAction) {
        Map<String, Object> payload = new HashMap<>();
        payload.put("applicationNo", citizenService.getServiceRequestId());
        payload.put("citizenName", citizenService.getAccountId() != null ? citizenService.getAccountId() : "");
        payload.put("serviceName", citizenService.getDescription() != null ? citizenService.getDescription() : "");
        payload.put("statusLabel", citizenService.getApplicationStatus());
        payload.put("trackUrl", "https://pgr.digit.org/track/" + citizenService.getServiceRequestId());

        if (workflowAction != null) {
            payload.put("action", workflowAction);
        }

        return payload;
    }
}
```


### 2. WorkflowServiceClient.java
```java
package com.example.pgrown30.client;

import org.digit.services.workflow.WorkflowClient;
import org.digit.services.workflow.model.WorkflowTransitionRequest;
import org.digit.services.workflow.model.WorkflowTransitionResponse;
import com.example.pgrown30.web.models.CitizenService;
import com.example.pgrown30.web.models.ServiceWrapper;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;

import java.util.Collections;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * Wrapper class for workflow-related operations.
 * Encapsulates workflow client interactions to reduce clutter in service layer.
 */
@Slf4j
@Component
@RequiredArgsConstructor
public class WorkflowServiceClient {

    private final WorkflowClient workflowClient;

    @Value("${pgr.workflow.processCode}")
    private String processCode;

    /**
     * Starts a workflow for a new citizen service.
     *
     * @param service the citizen service
     * @param action  the workflow action (e.g., "APPLY")
     * @param roles   list of user roles
     * @return workflow transition response, or null if workflow start fails
     */
    public WorkflowTransitionResponse startWorkflow(CitizenService service, String action, List<String> roles) {
        try {
            Map<String, List<String>> attributes = new HashMap<>();
            attributes.put("roles", roles != null ? roles : Collections.emptyList());
            attributes.put("tenantId", Collections.singletonList(service.getTenantId()));

            WorkflowTransitionRequest request = WorkflowTransitionRequest.builder()
                    .processId(workflowClient.getProcessByCode(processCode))
                    .entityId(service.getServiceRequestId())
                    .action(action)
                    .comment("Complaint submitted")
                    .attributes(attributes)
                    .build();

            return workflowClient.executeTransition(request);
        } catch (Exception ex) {
            log.error("Failed to start workflow for {}: {}", service.getServiceRequestId(), ex.getMessage(), ex);
            return null;
        }
    }

    /**
     * Handles workflow transition for service updates.
     *
     * @param wrapper  the service wrapper containing workflow action
     * @param existing the existing citizen service
     * @param roles    list of user roles
     * @return workflow transition response, or null if no action specified
     * @throws RuntimeException if workflow transition fails
     */
    public WorkflowTransitionResponse handleWorkflowTransition(ServiceWrapper wrapper, CitizenService existing, List<String> roles) {
        String workflowAction = (wrapper.getWorkflow() != null) ? wrapper.getWorkflow().getAction() : null;

        if (workflowAction == null || workflowAction.isBlank()) {
            return null;
        }

        try {
            Map<String, List<String>> data = new HashMap<>();
            data.put("roles", roles != null ? roles : Collections.emptyList());

            if (wrapper.getWorkflow() != null && wrapper.getWorkflow().getAssignes() != null) {
                data.put("assignes", wrapper.getWorkflow().getAssignes());
            }

            WorkflowTransitionRequest request = WorkflowTransitionRequest.builder()
                    .processId(workflowClient.getProcessByCode(processCode))
                    .entityId(existing.getServiceRequestId())
                    .action(workflowAction)
                    .comment("Updating service request")
                    .attributes(data)
                    .build();

            return workflowClient.executeTransition(request);
        } catch (Exception e) {
            log.error("Workflow transition failed for {} action={}: {}", existing.getServiceRequestId(), workflowAction, e.getMessage(), e);
            throw new RuntimeException("Workflow transition failed", e);
        }
    }

    /**
     * Updates the citizen service with workflow response data.
     *
     * @param service the citizen service to update
     * @param wfResp  the workflow transition response
     */
    public void updateCitizenServiceWithWorkflow(CitizenService service, WorkflowTransitionResponse wfResp) {
        service.setWorkflowInstanceId(wfResp.getId());
        service.setProcessId(wfResp.getProcessId());
        try {
            service.setApplicationStatus(wfResp.getCurrentState());
        } catch (IllegalArgumentException e) {
            log.warn("Unknown workflow state '{}', defaulting to INITIATED", wfResp.getCurrentState());
        }
    }

    /**
     * Enriches the response DTO with workflow information.
     *
     * @param saved  the saved citizen service
     * @param wfResp the workflow transition response
     * @return the citizen service with workflow info added
     */
    public CitizenService enrichResponseWithWorkflow(CitizenService saved, WorkflowTransitionResponse wfResp) {
        if (wfResp != null) {
            saved.setWorkflowInstanceId(wfResp.getId());
            saved.setAction(wfResp.getAction());
        }
        return saved;
    }
}
```

Now the Client Layer is ready!
