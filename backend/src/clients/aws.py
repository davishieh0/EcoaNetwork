"""
AWS connection.
"""

# --- IMPORTS ---
from aiobotocore.config import AioConfig
from botocore import UNSIGNED
from contextlib import AsyncExitStack
from typing import Any

import aioboto3


# --- GLOBALS ---
CONNECT_TIMEOUT = 5
READ_TIMEOUT = 10
MAX_ATTEMPTS = 3


# --- CODE ---
class Aws:
    """
    Opens the AWS service clients the backend uses, and closes them together.
    """

    def __init__(self, region: str, endpoint_url: str = '') -> None:
        """
        Initialize the AWS connection.

        :param region: Region every client uses unless it asks for another.
        :param endpoint_url: Endpoint every client uses instead of AWS unless
                             it asks for another, e.g. LocalStack. Empty to
                             talk to AWS itself.

        :return: None.
        """
        self._session = aioboto3.Session(region_name=region)
        self._endpoint_url = endpoint_url.strip() or None
        self._stack = AsyncExitStack()


    async def client(
        self,
        service: str,
        *,
        region: str | None = None,
        endpoint_url: str | None = None,
        signed: bool = True,
    ) -> Any:
        """
        Opens a client for one AWS service, kept open until close().

        :param service: The service name, as botocore spells it.
        :param region: Region for this client, or None for the default one.
        :param endpoint_url: Endpoint for this client, or None for the default
                             one.
        :param signed: False for services called through their public
                       operations, which need no AWS credentials.

        :return: The open service client.
        """
        return await self._stack.enter_async_context(
            self._session.client(
                service,
                region_name=region,
                endpoint_url=endpoint_url or self._endpoint_url,
                config=AioConfig(
                    signature_version=None if signed else UNSIGNED,
                    connect_timeout=CONNECT_TIMEOUT,
                    read_timeout=READ_TIMEOUT,
                    retries={'max_attempts': MAX_ATTEMPTS, 'mode': 'standard'},
                ),
            )
        )


    async def close(self) -> None:
        """
        Closes every client opened so far.

        :return: None.
        """
        await self._stack.aclose()
