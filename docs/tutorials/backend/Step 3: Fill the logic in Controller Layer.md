# Fill the logic in Controller Layer

The controller layer serves as the entry point for all external interactions, translating HTTP requests into domain-specific operations. It is responsible for orchestrating request handling, delegating processing to the appropriate services, and returning standardized responses.

ServiceController.java has been generated for you, only few module specific changes are to be done.

The controller layer will look like this:

**1. ServiceController.java**
```java
package com.example.pgrown30.web.controllers;

import com.example.pgrown30.service.ServiceService;
import com.example.pgrown30.web.models.CitizenService;
import com.example.pgrown30.web.models.ServiceResponse;
import com.example.pgrown30.web.models.ServiceWrapper;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.http.ResponseEntity;
import org.springframework.security.core.annotation.AuthenticationPrincipal;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.web.bind.annotation.*;
import jakarta.validation.Valid;
import jakarta.validation.ConstraintViolationException;
import org.springframework.web.bind.MethodArgumentNotValidException;
import org.springframework.web.bind.annotation.RestControllerAdvice;

import java.net.URI;
import java.util.Collections;
import java.util.List;
import java.util.Map;
import java.util.regex.Matcher;
import java.util.regex.Pattern;
import java.util.stream.Collectors;

@RestController
@RequestMapping("/citizen-service")
@RequiredArgsConstructor
@Slf4j
@RestControllerAdvice
public class ServiceController {

    private final ServiceService serviceService;

    private static final Pattern REALM_PATTERN = Pattern.compile(".*/realms/([^/]+)/*$");

    private List<String> getRealmRoles(Jwt jwt) {
        if (jwt == null) return List.of();
        Map<String, Object> realmAccess = jwt.getClaim("realm_access");
        if (realmAccess == null) return List.of();

        @SuppressWarnings("unchecked")
        List<String> roles = (List<String>) realmAccess.getOrDefault("roles", List.of());
        return roles;
    }

    private static String getTenantIdFromIssuer(Jwt jwt) {
        if (jwt == null || jwt.getIssuer() == null) return null;

        String iss = jwt.getIssuer().toString();
        Matcher m = REALM_PATTERN.matcher(iss);

        if (m.matches()) return m.group(1);

        try {
            String path = URI.create(iss).getPath();
            Matcher m2 = REALM_PATTERN.matcher(path);
            if (m2.matches()) return m2.group(1);
        } catch (Exception ignored) {}

        return null;
    }

    // ----------------------- CREATE -----------------------
    @PostMapping("/create")
    public ResponseEntity<ServiceResponse> create(
            @Valid @RequestBody ServiceWrapper wrapper,
            @AuthenticationPrincipal Jwt jwt
    ) {
        List<String> roles = getRealmRoles(jwt);
        String tenantId = getTenantIdFromIssuer(jwt);

        if (tenantId != null) wrapper.getService().setTenantId(tenantId);
        else log.warn("create: tenantId could not be resolved from JWT issuer");

        log.debug("create: roles={}, tenant={}", roles, tenantId);

        ServiceResponse response = serviceService.createService(wrapper, roles);
        return ResponseEntity.ok(response);
    }

    // ----------------------- UPDATE -----------------------
    @PostMapping("/update")
    public ResponseEntity<ServiceResponse> update(
            @Valid @RequestBody ServiceWrapper wrapper,
            @AuthenticationPrincipal Jwt jwt
    ) {
        List<String> roles = getRealmRoles(jwt);
        String tenantId = getTenantIdFromIssuer(jwt);

        if (tenantId != null) wrapper.getService().setTenantId(tenantId);
        else log.warn("update: tenantId could not be resolved from JWT issuer");

        log.debug("update: requestId={}, roles={}, tenant={}",
                wrapper.getService().getServiceRequestId(), roles, tenantId);

        ServiceResponse response = serviceService.updateService(wrapper, roles);
        return ResponseEntity.ok(response);
    }

    // ----------------------- UPDATE BY REGISTRY ID -----------------------
    @PutMapping("/update/{registryId}")
    public ResponseEntity<ServiceResponse> updateByRegistryId(
            @PathVariable String registryId,
            @Valid @RequestBody ServiceWrapper wrapper,
            @AuthenticationPrincipal Jwt jwt
    ) {
        if (registryId == null || registryId.isBlank()) {
            log.warn("updateByRegistryId: missing registryId");
            return ResponseEntity.badRequest().body(new ServiceResponse(List.of(), List.of()));
        }

        List<String> roles = getRealmRoles(jwt);
        String tenantId = getTenantIdFromIssuer(jwt);

        if (tenantId != null) wrapper.getService().setTenantId(tenantId);
        else log.warn("updateByRegistryId: tenantId could not be resolved from JWT issuer");

        // Set the registryId in the service object
        wrapper.getService().setRegistryId(registryId);

        log.debug("updateByRegistryId: registryId={}, requestId={}, roles={}, tenant={}",
                registryId, wrapper.getService().getServiceRequestId(), roles, tenantId);

        ServiceResponse response = serviceService.updateService(wrapper, roles);
        return ResponseEntity.ok(response);
    }

    // ----------------------- SEARCH -----------------------
    @GetMapping("/search")
    public ResponseEntity<ServiceResponse> search(
            @RequestParam(value = "serviceRequestId", required = true) String serviceRequestId,
            @AuthenticationPrincipal Jwt jwt
    ) {
        if (serviceRequestId == null || serviceRequestId.isBlank()) {
            log.warn("search: missing serviceRequestId");
            return ResponseEntity.badRequest().body(new ServiceResponse(List.of(), List.of()));
        }

        List<String> roles = getRealmRoles(jwt);
        String tenantId = getTenantIdFromIssuer(jwt);

        if (tenantId == null) log.warn("search: tenantId could not be resolved from JWT issuer");

        log.debug("search: serviceRequestId={}, tenant={}, roles={}",
                serviceRequestId, tenantId, roles);

        ServiceResponse response = serviceService.searchServicesById(serviceRequestId, tenantId);
        return ResponseEntity.ok(response);
    }

    // ----------------------- VALIDATION HANDLERS -----------------------
    @ExceptionHandler(MethodArgumentNotValidException.class)
    public ResponseEntity<ServiceResponse> handleValidationExceptions(MethodArgumentNotValidException ex) {
        log.warn("Validation error: {}", ex.getMessage());

        String errors = ex.getBindingResult()
            .getFieldErrors()
            .stream()
            .map(fieldError -> String.format("%s %s",
                    fieldError.getField(),
                    fieldError.getDefaultMessage()))
            .collect(Collectors.joining(", "));

        return ResponseEntity.badRequest().body(
            ServiceResponse.builder()
                .services(Collections.emptyList())
                .serviceWrappers(Collections.singletonList(
                    ServiceWrapper.builder()
                        .service(CitizenService.builder()
                            .description("Validation error: " + errors)
                            .build())
                        .build()
                ))
                .build()
        );
    }

    @ExceptionHandler(ConstraintViolationException.class)
    public ResponseEntity<ServiceResponse> handleConstraintViolation(ConstraintViolationException ex) {
        log.warn("Constraint violation: {}", ex.getMessage());

        String errors = ex.getConstraintViolations()
            .stream()
            .map(v -> String.format("%s %s", v.getPropertyPath(), v.getMessage()))
            .collect(Collectors.joining(", "));

        return ResponseEntity.badRequest().body(
            ServiceResponse.builder()
                .services(Collections.emptyList())
                .serviceWrappers(Collections.singletonList(
                    ServiceWrapper.builder()
                        .service(CitizenService.builder()
                            .description("Validation error: " + errors)
                            .build())
                        .build()
                ))
                .build()
        );
    }
}

```
