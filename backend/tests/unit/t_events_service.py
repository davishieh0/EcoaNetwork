"""
Unit tests for the event service.
"""

# --- IMPORTS ---
from src.err.node_unavailable_error import NodeUnavailableError
from src.grpc.proto import node_api_pb2
from src.services.events import EventService
from typing import cast

import asyncio
import pytest


# --- GLOBALS ---
ENVELOPE = b'{"eventId":"evt_1"}'


# --- FAKES ---
class FakeNode:
    """Stands in for the node client: answers with a set response."""

    def __init__(self) -> None:
        self.envelopes: list[bytes] = []
        self.response = node_api_pb2.PublishResponse(
            outcome=node_api_pb2.ACCEPTED
        )

    async def publish(self, envelope: bytes) -> node_api_pb2.PublishResponse:
        self.envelopes.append(envelope)
        return self.response


# --- FIXTURES ---
@pytest.fixture
def node() -> FakeNode:
    return FakeNode()


@pytest.fixture
def events(node: FakeNode) -> EventService:
    return EventService(node)  # type: ignore[arg-type]


# --- PUBLISH ---
@pytest.mark.parametrize(
    ('outcome', 'name'),
    [
        (node_api_pb2.ACCEPTED, 'accepted'),
        (node_api_pb2.ALREADY_EXISTS, 'already_exists'),
        (node_api_pb2.ORPHAN, 'orphan'),
    ],
)
def test_publish_reports_the_node_outcome(
    events: EventService, node: FakeNode,
    outcome: node_api_pb2.Outcome, name: str,
) -> None:
    node.response = node_api_pb2.PublishResponse(outcome=outcome)

    result = asyncio.run(events.publish(ENVELOPE))

    assert (result.outcome, result.reason) == (name, None)
    assert node.envelopes == [ENVELOPE]


def test_publish_reports_why_the_node_rejected(
    events: EventService, node: FakeNode
) -> None:
    node.response = node_api_pb2.PublishResponse(
        outcome=node_api_pb2.REJECTED, reason='bad signature'
    )

    result = asyncio.run(events.publish(ENVELOPE))

    assert (result.outcome, result.reason) == ('rejected', 'bad signature')


@pytest.mark.parametrize(
    'outcome',
    [node_api_pb2.OUTCOME_UNSPECIFIED, cast(node_api_pb2.Outcome, 99)],
    ids=['unspecified', 'unknown'],
)
def test_outcome_outside_the_contract_is_a_failure(
    events: EventService, node: FakeNode, outcome: node_api_pb2.Outcome
) -> None:
    node.response = node_api_pb2.PublishResponse(outcome=outcome)

    with pytest.raises(NodeUnavailableError):
        asyncio.run(events.publish(ENVELOPE))
