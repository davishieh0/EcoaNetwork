"""
Unit tests for the event endpoints: the HTTP side only.
"""

# --- IMPORTS ---
from collections.abc import Iterator
from fastapi.testclient import TestClient
from src.app import app
from src.dependencies.events import MAX_ENVELOPE
from src.dependencies.events import get_events
from src.err.node_unavailable_error import NodeUnavailableError
from src.schemas.events import Outcome
from src.schemas.events import PublishResult

import pytest

# wires routes, handlers, middleware and the lifespan onto the app
import src.main  # noqa: F401


# --- GLOBALS ---
ENVELOPE = b'{"eventId":"evt_1","eventType":"review.created"}'


# --- FAKES ---
class FakeEvents:
    """Stands in for the service: records the envelopes it is handed."""

    def __init__(self) -> None:
        self.envelopes: list[bytes] = []
        self.outcome: Outcome = 'accepted'
        self.reason: str | None = None
        self.failure: Exception | None = None

    async def publish(self, envelope: bytes) -> PublishResult:
        self.envelopes.append(envelope)
        if self.failure:
            raise self.failure
        return PublishResult(outcome=self.outcome, reason=self.reason)


# --- FIXTURES ---
@pytest.fixture
def events() -> FakeEvents:
    return FakeEvents()


@pytest.fixture
def client(events: FakeEvents) -> Iterator[TestClient]:
    app.dependency_overrides[get_events] = lambda: events

    with TestClient(app) as test_client:
        yield test_client

    app.dependency_overrides.clear()


# --- OUTCOMES ---
@pytest.mark.parametrize(
    ('outcome', 'reason', 'status'),
    [
        ('accepted', None, 200),
        ('already_exists', None, 200),
        ('orphan', None, 202),
        ('rejected', 'bad signature', 400),
    ],
)
def test_outcome_sets_the_status_and_the_body(
    client: TestClient, events: FakeEvents,
    outcome: Outcome, reason: str | None, status: int,
) -> None:
    events.outcome, events.reason = outcome, reason

    response = client.post('/api/events', content=ENVELOPE)

    assert response.status_code == status
    assert response.json() == {'outcome': outcome, 'reason': reason}


def test_envelope_reaches_the_service_byte_for_byte(
    client: TestClient, events: FakeEvents
) -> None:
    envelope = b'{ "z":1,\n  "eventId" :"evt_2",  "a":"\\u00e9" }'

    client.post('/api/events', content=envelope)

    assert events.envelopes == [envelope]


def test_no_node_answering_is_a_503(
    client: TestClient, events: FakeEvents
) -> None:
    events.failure = NodeUnavailableError(['UNAVAILABLE'])

    response = client.post('/api/events', content=ENVELOPE)

    assert response.status_code == 503
    assert response.json()['error'] == 'node_unavailable_error'


# --- BODIES THAT NEVER REACH THE SERVICE ---
@pytest.mark.parametrize(
    'body',
    [b'not json', b'', b'"' + b'x' * MAX_ENVELOPE + b'"'],
    ids=['not json', 'empty', 'over the ceiling'],
)
def test_bad_body_never_reaches_the_service(
    client: TestClient, events: FakeEvents, body: bytes
) -> None:
    response = client.post('/api/events', content=body)

    assert response.status_code == 400
    assert response.json()['error'] == 'invalid_request_error'
    assert events.envelopes == []


def test_body_at_the_ceiling_reaches_the_service(
    client: TestClient, events: FakeEvents
) -> None:
    body = b'"' + b'x' * (MAX_ENVELOPE - 2) + b'"'

    response = client.post('/api/events', content=body)

    assert response.status_code == 200
    assert events.envelopes == [body]
