"""
Node connection.
"""

# --- IMPORTS ---
from src.err.node_unavailable_error import NodeUnavailableError
from src.grpc.proto import node_api_pb2
from src.grpc.proto import node_api_pb2_grpc

import grpc


# --- GLOBALS ---
CALL_TIMEOUT = 5
RETRYABLE = {grpc.StatusCode.UNAVAILABLE, grpc.StatusCode.DEADLINE_EXCEEDED}


# --- CODE ---
class NodeClient:
    """
    Holds one channel per node and publishes to the first that answers.
    """

    def __init__(self, addrs: list[str]) -> None:
        """
        Initialize the node connection.

        Nothing connects here: each channel opens on its first call.

        :param addrs: The nodes, as 'host:port', primary first.

        :return: None.
        """
        self._channels = [grpc.aio.insecure_channel(a) for a in addrs]
        self._stubs = [
            node_api_pb2_grpc.NodeServiceStub(c) for c in self._channels
        ]


    async def publish(self, envelope: bytes) -> node_api_pb2.PublishResponse:
        """
        Submits a signed envelope to the first node that answers.

        :param envelope: The envelope, byte for byte as it was signed.

        :raises NodeUnavailableError: If no node in the list answers.

        :return: What the node did with the event.
        """
        event = node_api_pb2.Event(envelope=envelope)

        # status code of each node that did not answer
        failures = []

        for stub in self._stubs:
            try:
                response: node_api_pb2.PublishResponse = await stub.Publish(
                    event, timeout=CALL_TIMEOUT
                )
                return response

            # node down or too slow: try the next one
            except grpc.aio.AioRpcError as error:
                if error.code() not in RETRYABLE:
                    raise
                failures.append(error.code().name)

        raise NodeUnavailableError(failures)


    async def close(self) -> None:
        """
        Closes every channel.

        :return: None.
        """
        for channel in self._channels:
            await channel.close()
