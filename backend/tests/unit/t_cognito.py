"""
Unit tests for the Cognito client.
"""

# --- IMPORTS ---
from botocore.exceptions import ClientError
from collections.abc import Awaitable
from collections.abc import Callable
from src.clients.cognito import CognitoClient
from src.err.auth_provider_error import AuthProviderError
from src.err.email_already_registered_error import EmailAlreadyRegisteredError
from src.err.invalid_password_error import InvalidPasswordError
from src.err.too_many_requests_error import TooManyRequestsError
from typing import Any

import asyncio
import pytest


# --- GLOBALS ---
EMAIL = 'user@example.com'
SUB = 'sub-123'

# HMAC-SHA256 of username + 'client-id', keyed with 'client-secret'
EMAIL_HASH = 'NpqglvUHBpz0nyO+qeiUjlRR297+e+uA7UhuwY3pkSE='


# --- HELPERS ---
class FakeIdp:
    """Stands in for the 'cognito-idp' service client."""

    def __init__(
        self,
        response: dict[str, Any] | None = None,
        error: Exception | None = None,
    ) -> None:
        self.response = response or {}
        self.error = error
        self.calls: list[tuple[str, dict[str, Any]]] = []

    def __getattr__(
        self, operation: str
    ) -> Callable[..., Awaitable[dict[str, Any]]]:
        async def call(**params: Any) -> dict[str, Any]:
            self.calls.append((operation, params))

            if self.error:
                raise self.error

            return self.response

        return call


def _failure(code: str, message: str = '') -> ClientError:
    """Builds the error Cognito raises for the given code."""
    return ClientError({'Error': {'Code': code, 'Message': message}}, 'Op')


def _cognito(idp: FakeIdp) -> CognitoClient:
    """Builds the client under test on top of a fake service client."""
    return CognitoClient(idp, 'client-id', 'client-secret')


# --- SIGN UP ---
def test_sign_up_proves_the_secret_and_returns_the_sub() -> None:
    idp = FakeIdp({'UserSub': SUB})

    assert asyncio.run(_cognito(idp).sign_up(EMAIL, 'Passw0rd!')) == SUB

    operation, params = idp.calls[0]
    assert operation == 'sign_up'
    assert params['SecretHash'] == EMAIL_HASH
    assert params['UserAttributes'] == [{'Name': 'email', 'Value': EMAIL}]


@pytest.mark.parametrize(
    ('code', 'expected'),
    [
        ('UsernameExistsException', EmailAlreadyRegisteredError),
        ('InvalidPasswordException', InvalidPasswordError),
        ('TooManyRequestsException', TooManyRequestsError),
        ('SomethingNobodyPlannedFor', AuthProviderError),
    ],
)
def test_sign_up_failures(code: str, expected: type[Exception]) -> None:
    idp = FakeIdp(error=_failure(code))

    with pytest.raises(expected):
        asyncio.run(_cognito(idp).sign_up(EMAIL, 'Passw0rd!'))
