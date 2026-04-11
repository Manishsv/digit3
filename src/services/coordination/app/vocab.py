"""Controlled vocabulary aligned with provision MDMS seeds (extend both when adding codes)."""

ENTITY_TYPES = frozenset(
    {
        "Parcel",
        "Record",
        "Application",
        "Case",
        "Document",
        "Participant",
        "MapRef",
        "Payment",
        "Deed",
    }
)
RELATION_TYPES = frozenset(
    {
        "APPLICATION_TO_PARCEL",
        "PARCEL_TO_RECORD",
        "PARCEL_TO_MAP",
        "APPLICATION_TO_CASE",
        "PARCEL_TO_CASE",
        "DOCUMENT_TO_APPLICATION",
        "CASE_TO_TRACE",
        "APPLICATION_TO_DEED",
        "APPLICATION_TO_PAYMENT",
    }
)
EVENT_TYPES = frozenset(
    {
        "MUTATION_SUBMITTED",
        "VERIFICATION_STARTED",
        "DEED_VERIFIED",
        "MUTATION_APPROVED",
        "RECORD_UPDATED",
        "NOC_SUBMITTED",
        "NOC_APPROVED",
        "PAYMENT_RECEIVED",
        "DISPUTE_OPENED",
        "CASE_CLOSED",
    }
)
PARTICIPANT_TYPES = frozenset({"SYSTEM", "DEPARTMENT", "OFFICE", "REGISTRAR", "PARTNER"})
FACT_TYPES = frozenset(
    {
        "PARCEL_GEOMETRY",
        "LAND_RECORD",
        "REGISTRATION_STATUS",
        "PAYMENT_STATUS",
        "CASE_STATUS",
    }
)
STATUSES = frozenset({"ACTIVE", "PENDING", "RESOLVED", "AMBIGUOUS", "INACTIVE"})


def check_entity_type(code: str) -> None:
    if code not in ENTITY_TYPES:
        raise ValueError(f"Unknown entity type: {code}")


def check_relation_type(code: str) -> None:
    if code not in RELATION_TYPES:
        raise ValueError(f"Unknown relation type: {code}")


def check_event_type(code: str) -> None:
    if code not in EVENT_TYPES:
        raise ValueError(f"Unknown event type: {code}")


def check_participant_type(code: str) -> None:
    if code not in PARTICIPANT_TYPES:
        raise ValueError(f"Unknown participant type: {code}")


def check_fact_type(code: str) -> None:
    if code not in FACT_TYPES:
        raise ValueError(f"Unknown fact type: {code}")


def check_status(code: str) -> None:
    if code not in STATUSES:
        raise ValueError(f"Unknown status: {code}")

