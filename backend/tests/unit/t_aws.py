"""
Unit tests for the AWS connection.
"""

# --- IMPORTS ---
from src.clients.aws import Aws
from typing import Any

import asyncio


# --- GLOBALS ---
LOCALSTACK = 'http://localhost:4566'


# --- HELPERS ---
def _open(aws: Aws, **options: Any) -> tuple[str, str]:
    """Opens an unsigned client and reports where it points."""

    async def run() -> tuple[str, str]:
        client = await aws.client('cognito-idp', signed=False, **options)
        target = (client.meta.region_name, client.meta.endpoint_url)
        await aws.close()

        return target

    return asyncio.run(run())


# --- TESTS ---
def test_client_defaults_to_the_connection_region_on_aws() -> None:
    region, endpoint = _open(Aws('sa-east-1'))

    assert region == 'sa-east-1'
    assert endpoint == 'https://cognito-idp.sa-east-1.amazonaws.com'


def test_client_can_ask_for_another_region() -> None:
    region, endpoint = _open(Aws('sa-east-1'), region='us-east-1')

    assert region == 'us-east-1'
    assert endpoint == 'https://cognito-idp.us-east-1.amazonaws.com'


def test_connection_endpoint_applies_to_every_client() -> None:
    _, endpoint = _open(Aws('sa-east-1', LOCALSTACK))

    assert endpoint == LOCALSTACK


def test_client_can_ask_for_another_endpoint() -> None:
    _, endpoint = _open(Aws('sa-east-1'), endpoint_url=LOCALSTACK)

    assert endpoint == LOCALSTACK
