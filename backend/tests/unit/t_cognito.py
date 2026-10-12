"""
Unit tests for the Cognito client.
"""

# --- IMPORTS ---
from botocore.exceptions import ClientError
from botocore.exceptions import EndpointConnectionError
from collections.abc import Awaitable
from collections.abc import Callable
from src.clients.cognito import CognitoClient
from src.err.auth_provider_error import AuthProviderError
from src.err.email_already_registered_error import EmailAlreadyRegisteredError
from src.err.email_not_verified_error import EmailNotVerifiedError
from src.err.expired_code_error import ExpiredCodeError
from src.err.invalid_code_error import InvalidCodeError
from src.err.invalid_credentials_error import InvalidCredentialsError
from src.err.invalid_password_error import InvalidPasswordError
from src.err.invalid_session_error import InvalidSessionError
from src.err.too_many_requests_error import TooManyRequestsError
from typing import Any

import asyncio
import pytest


# --- GLOBALS ---
EMAIL = 'user@example.com'
SUB = 'sub-123'

# HMAC-SHA256 of username + 'client-id', keyed with 'client-secret'
EMAIL_HASH = 'NpqglvUHBpz0nyO+qeiUjlRR297+e+uA7UhuwY3pkSE='
SUB_HASH = 'XLhAQJDyIlRwnmQ1Ka/pAjmq2QYB2s+Uld8cjW/QqNk='

TOKENS = {
    'AuthenticationResult': {
        'AccessToken': 'access',
        'ExpiresIn': 3600,
        'RefreshToken': 'refresh',
    }
}


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


# --- CONFIRMATION ---
@pytest.mark.parametrize(
    ('code', 'expected'),
    [
        ('CodeMismatchException', InvalidCodeError),
        ('ExpiredCodeException', ExpiredCodeError),
        # neither says whether the address has an account
        ('UserNotFoundException', InvalidCodeError),
        ('NotAuthorizedException', InvalidCodeError),
    ],
)
def test_confirm_sign_up_failures(
    code: str, expected: type[Exception]
) -> None:
    idp = FakeIdp(error=_failure(code))

    with pytest.raises(expected):
        asyncio.run(_cognito(idp).confirm_sign_up(EMAIL, '123456'))


@pytest.mark.parametrize(
    'code', ['UserNotFoundException', 'InvalidParameterException']
)
def test_resend_succeeds_without_an_account_waiting(code: str) -> None:
    idp = FakeIdp(error=_failure(code))

    asyncio.run(_cognito(idp).resend_confirmation_code(EMAIL))


# --- SIGN IN ---
def test_sign_in_returns_the_tokens() -> None:
    idp = FakeIdp(TOKENS)

    tokens = asyncio.run(_cognito(idp).sign_in(EMAIL, 'Passw0rd!'))

    assert tokens.access_token == 'access'
    assert tokens.refresh_token == 'refresh'
    assert tokens.expires_in == 3600

    operation, params = idp.calls[0]
    assert operation == 'initiate_auth'
    assert params['AuthFlow'] == 'USER_PASSWORD_AUTH'
    assert params['AuthParameters']['SECRET_HASH'] == EMAIL_HASH


@pytest.mark.parametrize(
    ('code', 'message', 'expected'),
    [
        (
            'NotAuthorizedException',
            'Incorrect username or password.',
            InvalidCredentialsError,
        ),
        ('UserNotFoundException', '', InvalidCredentialsError),
        ('UserNotConfirmedException', '', EmailNotVerifiedError),
        (
            'NotAuthorizedException',
            'Password attempts exceeded',
            TooManyRequestsError,
        ),
        # our misconfiguration must not read as the caller's wrong password
        (
            'NotAuthorizedException',
            'Unable to verify secret hash for client client-id',
            AuthProviderError,
        ),
    ],
)
def test_sign_in_failures(
    code: str, message: str, expected: type[Exception]
) -> None:
    idp = FakeIdp(error=_failure(code, message))

    with pytest.raises(expected):
        asyncio.run(_cognito(idp).sign_in(EMAIL, 'Passw0rd!'))


def test_sign_in_rejects_a_challenge_it_cannot_answer() -> None:
    idp = FakeIdp({'ChallengeName': 'SOFTWARE_TOKEN_MFA'})

    with pytest.raises(AuthProviderError):
        asyncio.run(_cognito(idp).sign_in(EMAIL, 'Passw0rd!'))


def test_unreachable_cognito_is_a_provider_failure() -> None:
    idp = FakeIdp(error=EndpointConnectionError(endpoint_url='http://x'))

    with pytest.raises(AuthProviderError):
        asyncio.run(_cognito(idp).sign_in(EMAIL, 'Passw0rd!'))


# --- SESSION ---
def test_refresh_hashes_the_secret_against_the_sub() -> None:
    idp = FakeIdp({'AuthenticationResult': {'AccessToken': 'new',
                                            'ExpiresIn': 3600}})

    tokens = asyncio.run(_cognito(idp).refresh('refresh', SUB))

    assert tokens.access_token == 'new'
    assert tokens.refresh_token is None

    _, params = idp.calls[0]
    assert params['AuthFlow'] == 'REFRESH_TOKEN_AUTH'
    assert params['AuthParameters']['SECRET_HASH'] == SUB_HASH


def test_refresh_with_a_dead_token_ends_the_session() -> None:
    idp = FakeIdp(error=_failure('NotAuthorizedException', 'Token revoked'))

    with pytest.raises(InvalidSessionError):
        asyncio.run(_cognito(idp).refresh('refresh', SUB))
