"""
Event service.
"""

# --- IMPORTS ---
from src.clients.node import NodeClient
from src.err.node_unavailable_error import NodeUnavailableError
from src.grpc.proto import node_api_pb2
from src.schemas.events import Outcome
from src.schemas.events import PublishResult


# --- GLOBALS ---
OUTCOMES: dict[int, Outcome] = {
    node_api_pb2.ACCEPTED: 'accepted',
    node_api_pb2.ALREADY_EXISTS: 'already_exists',
    node_api_pb2.ORPHAN: 'orphan',
    node_api_pb2.REJECTED: 'rejected',
}


# --- CODE ---
class EventService:
    """
    Signed events: hands them to the network and reports what became of them.
    """

    def __init__(self, node: NodeClient) -> None:
        """
        Initialize the service.

        :param node: The node connection.

        :return: None.
        """
        self._node = node


    async def publish(self, envelope: bytes) -> PublishResult:
        """
        Submits a signed envelope to a node.

        :param envelope: The envelope, byte for byte as it was signed.

        :raises NodeUnavailableError: If no node answers, or one answers with
            an outcome outside the contract.

        :return: What the node did with the event.
        """
        response = await self._node.publish(envelope)

        # an outcome this gateway does not know
        if response.outcome not in OUTCOMES:
            raise NodeUnavailableError(response.outcome)

        return PublishResult(
            outcome=OUTCOMES[response.outcome],
            reason=response.reason or None,
        )
