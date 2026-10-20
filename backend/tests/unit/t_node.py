"""
Unit tests for the node connection.
"""

# --- IMPORTS ---
from src.clients.node import NodeClient
from src.err.node_unavailable_error import NodeUnavailableError
from src.grpc.proto import node_api_pb2
from typing import Any

import asyncio
import grpc
import pytest


# --- GLOBALS ---
ENVELOPE = b'{"eventId":"evt_1"}'
ACCEPTED = node_api_pb2.PublishResponse(outcome=node_api_pb2.ACCEPTED)


# --- FAKES ---
class FakeStub:
    """Stands in for one node: answers, or fails with a status code."""

    def __init__(self, failure: grpc.StatusCode | None = None) -> None:
        self.failure = failure
        self.events: list[node_api_pb2.Event] = []

    async def Publish(  # noqa: N802
        self, event: node_api_pb2.Event, timeout: float
    ) -> node_api_pb2.PublishResponse:
        self.events.append(event)
        if self.failure:
            raise grpc.aio.AioRpcError(
                self.failure, grpc.aio.Metadata(), grpc.aio.Metadata()
            )
        return ACCEPTED


# --- HELPERS ---
def _client(*stubs: FakeStub) -> NodeClient:
    async def build() -> NodeClient:
        node = NodeClient([])
        node._stubs = list(stubs)  # type: ignore[arg-type]
        return node

    return asyncio.run(build())


def _publish(node: NodeClient) -> Any:
    return asyncio.run(node.publish(ENVELOPE))


# --- PUBLISH ---
def test_publish_hands_the_envelope_to_the_first_node() -> None:
    first, second = FakeStub(), FakeStub()

    response = _publish(_client(first, second))

    assert response == ACCEPTED
    assert [e.envelope for e in first.events] == [ENVELOPE]
    assert second.events == []


@pytest.mark.parametrize(
    'failure',
    [grpc.StatusCode.UNAVAILABLE, grpc.StatusCode.DEADLINE_EXCEEDED],
)
def test_publish_falls_back_to_the_next_node(
    failure: grpc.StatusCode,
) -> None:
    first, second = FakeStub(failure), FakeStub()

    response = _publish(_client(first, second))

    assert response == ACCEPTED
    assert [e.envelope for e in second.events] == [ENVELOPE]


def test_publish_with_no_node_answering_names_each_failure() -> None:
    node = _client(FakeStub(grpc.StatusCode.UNAVAILABLE),
                   FakeStub(grpc.StatusCode.DEADLINE_EXCEEDED))

    with pytest.raises(NodeUnavailableError) as raised:
        _publish(node)

    assert raised.value.args[1] == ['UNAVAILABLE', 'DEADLINE_EXCEEDED']


def test_publish_does_not_retry_a_failure_of_the_call_itself() -> None:
    first, second = FakeStub(grpc.StatusCode.RESOURCE_EXHAUSTED), FakeStub()

    with pytest.raises(grpc.aio.AioRpcError):
        _publish(_client(first, second))

    assert second.events == []


def test_publish_with_no_nodes_is_unavailable() -> None:
    with pytest.raises(NodeUnavailableError):
        _publish(_client())
