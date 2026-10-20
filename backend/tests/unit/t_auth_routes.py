"""
Unit tests for the authentication endpoints: the HTTP side only.
"""

# --- IMPORTS ---
from collections.abc import Iterator
from fastapi.testclient import TestClient
from src.app import app
from src.dependencies.auth import get_auth
from src.dependencies.auth import get_tokens
from src.err.email_already_registered_error import EmailAlreadyRegisteredError
from src.err.email_not_verified_error import EmailNotVerifiedError
from src.err.handle_taken_error import HandleTakenError
from src.err.invalid_credentials_error import InvalidCredentialsError
from src.err.invalid_session_error import InvalidSessionError
from src.err.not_authenticated_error import NotAuthenticatedError
from src.schemas.auth import Account
from src.schemas.auth import Session
from src.schemas.auth import Tokens
from src.utils.cookies import ACCESS_TOKEN_COOKIE
from src.utils.cookies import REFRESH_TOKEN_COOKIE
from tests.unit.t_auth_service import FakeTokens
from typing import Any

import pytest

# wires routes, handlers, middleware and the lifespan onto the app
import src.main  # noqa: F401


# --- GLOBALS ---
SUB = 'sub-1'
EMAIL = 'user@example.com'
PASSWORD = 'Passw0rd!'
ACCOUNT = Account(sub=SUB, email=EMAIL, handle='eduardo')
SIGNED_IN = Session(
    tokens=Tokens(access_token=f'access-{SUB}', expires_in=3600,
                  refresh_token='refresh-1'),
    account=ACCOUNT,
)


# --- FAKES ---
class FakeAuth:
    """Stands in for the service: records what the routes hand it."""

    def __init__(self) -> None:
        self.calls: list[tuple[str, tuple[Any, ...]]] = []
        self.failure: Exception | None = None

    def __getattr__(self, method: str) -> Any:
        async def call(*args: Any) -> Any:
            self.calls.append((method, args))
            if self.failure:
                raise self.failure
            if method == 'register':
                return ACCOUNT
            if method == 'login':
                return SIGNED_IN
            if method == 'refresh':
                return Tokens(access_token='renewed', expires_in=3600)
            if method == 'me':
                return Account(sub=args[0], email=EMAIL, handle='eduardo')
            return None

        return call


# --- FIXTURES ---
@pytest.fixture
def auth() -> FakeAuth:
    return FakeAuth()


@pytest.fixture
def client(auth: FakeAuth) -> Iterator[TestClient]:
    app.dependency_overrides[get_auth] = lambda: auth
    app.dependency_overrides[get_tokens] = FakeTokens

    # https, as in production: the session cookies are Secure
    with TestClient(app, base_url='https://testserver') as test_client:
        yield test_client

    app.dependency_overrides.clear()


def _set_cookies(response: Any) -> dict[str, str]:
    return {
        header.split('=', 1)[0]: header
        for header in response.headers.get_list('set-cookie')
    }


# --- REQUESTS REACH THE SERVICE ---
def test_register_returns_the_account_created(
    client: TestClient, auth: FakeAuth
) -> None:
    response = client.post('/api/auth/register', json={
        'handle': 'eduardo', 'email': EMAIL, 'password': PASSWORD,
    })

    assert response.status_code == 201
    assert response.json() == ACCOUNT.model_dump()
    assert auth.calls == [('register', ('eduardo', EMAIL, PASSWORD))]


@pytest.mark.parametrize(
    ('path', 'body', 'call'),
    [
        ('/verify', {'email': EMAIL, 'code': '123456'},
         ('verify', (EMAIL, '123456'))),
        ('/verify/resend', {'email': EMAIL},
         ('resend_verification', (EMAIL,))),
        ('/forgot-password', {'email': EMAIL},
         ('forgot_password', (EMAIL,))),
        ('/reset-password',
         {'email': EMAIL, 'code': '123456', 'password': 'N3w-passw0rd'},
         ('reset_password', (EMAIL, '123456', 'N3w-passw0rd'))),
    ],
)
def test_route_hands_the_request_to_the_service(
    client: TestClient, auth: FakeAuth,
    path: str, body: dict[str, str], call: tuple[str, tuple[str, ...]],
) -> None:
    response = client.post(f'/api/auth{path}', json=body)

    assert response.status_code == 204
    assert response.content == b''
    assert auth.calls == [call]


@pytest.mark.parametrize(
    ('path', 'body'),
    [
        ('/register', {'handle': 'Eduardo', 'email': EMAIL,
                       'password': PASSWORD}),
        ('/register', {'handle': 'edu.k', 'email': EMAIL,
                       'password': PASSWORD}),
        ('/register', {'handle': 'eduardo', 'email': 'not-an-email',
                       'password': PASSWORD}),
        ('/register', {'handle': 'eduardo', 'email': EMAIL,
                       'password': 'short'}),
        ('/register', {'handle': 'eduardo', 'email': EMAIL}),
        ('/verify', {'email': EMAIL, 'code': '12345'}),
        ('/verify', {'email': EMAIL, 'code': 'abcdef'}),
        ('/login', {'email': EMAIL, 'password': ''}),
        ('/reset-password', {'email': EMAIL, 'code': '123456',
                             'password': 'short'}),
    ],
    ids=['uppercase handle', 'dotted handle', 'bad email', 'short password',
         'no password', 'five digit code', 'letters in code',
         'empty password', 'short new password'],
)
def test_malformed_request_never_reaches_the_service(
    client: TestClient, auth: FakeAuth, path: str, body: dict[str, str]
) -> None:
    response = client.post(f'/api/auth{path}', json=body)

    assert response.status_code == 422
    assert auth.calls == []


@pytest.mark.parametrize(
    ('failure', 'status', 'slug'),
    [
        (HandleTakenError(), 409, 'handle_taken_error'),
        (EmailAlreadyRegisteredError(), 409, 'email_already_registered_error'),
    ],
)
def test_service_error_becomes_its_status(
    client: TestClient, auth: FakeAuth,
    failure: Exception, status: int, slug: str,
) -> None:
    auth.failure = failure

    response = client.post('/api/auth/register', json={
        'handle': 'eduardo', 'email': EMAIL, 'password': PASSWORD,
    })

    assert response.status_code == status
    assert response.json()['error'] == slug


# --- COOKIES ---
def test_login_stores_the_session_and_returns_the_account(
    client: TestClient, auth: FakeAuth
) -> None:
    response = client.post('/api/auth/login',
                           json={'email': EMAIL, 'password': PASSWORD})

    assert response.status_code == 200
    assert response.json() == ACCOUNT.model_dump()
    assert auth.calls == [('login', (EMAIL, PASSWORD))]
    assert set(_set_cookies(response)) == {ACCESS_TOKEN_COOKIE,
                                           REFRESH_TOKEN_COOKIE}


@pytest.mark.parametrize(
    ('failure', 'status'),
    [(InvalidCredentialsError(), 401), (EmailNotVerifiedError(), 403)],
)
def test_login_that_fails_sets_no_cookie(
    client: TestClient, auth: FakeAuth, failure: Exception, status: int
) -> None:
    auth.failure = failure

    response = client.post('/api/auth/login',
                           json={'email': EMAIL, 'password': PASSWORD})

    assert response.status_code == status
    assert _set_cookies(response) == {}


def test_refresh_hands_over_both_cookies_and_replaces_the_access_one(
    client: TestClient, auth: FakeAuth
) -> None:
    client.cookies.set(ACCESS_TOKEN_COOKIE, f'expired-{SUB}', path='/api')
    client.cookies.set(REFRESH_TOKEN_COOKIE, 'refresh-1', path='/api/auth')

    response = client.post('/api/auth/refresh')

    assert response.status_code == 204
    assert auth.calls == [('refresh', (f'expired-{SUB}', 'refresh-1'))]
    assert set(_set_cookies(response)) == {ACCESS_TOKEN_COOKIE}


def test_refresh_without_cookies_hands_over_nothing(
    client: TestClient, auth: FakeAuth
) -> None:
    auth.failure = InvalidSessionError()

    response = client.post('/api/auth/refresh')

    assert response.status_code == 401
    assert auth.calls == [('refresh', (None, None))]
    assert _set_cookies(response) == {}


def test_logout_revokes_and_removes_the_cookies(
    client: TestClient, auth: FakeAuth
) -> None:
    client.post('/api/auth/login', json={'email': EMAIL, 'password': PASSWORD})

    response = client.post('/api/auth/logout')

    assert response.status_code == 204
    assert auth.calls[-1] == ('logout', ('refresh-1',))
    assert all('Max-Age=0' in h for h in _set_cookies(response).values())
    assert not client.cookies


# --- ME ---
def test_me_returns_the_account_of_the_verified_caller(
    client: TestClient, auth: FakeAuth
) -> None:
    client.cookies.set(ACCESS_TOKEN_COOKIE, f'access-{SUB}', path='/api')

    response = client.get('/api/auth/me')

    assert response.status_code == 200
    assert response.json() == {'sub': SUB, 'email': EMAIL,
                               'handle': 'eduardo'}
    assert auth.calls == [('me', (SUB, f'access-{SUB}'))]


@pytest.mark.parametrize(
    ('cookie', 'slug'),
    [
        (None, 'not_authenticated_error'),
        (f'expired-{SUB}', 'access_token_expired_error'),
        ('forged', 'not_authenticated_error'),
    ],
)
def test_me_never_reaches_the_service_without_a_valid_token(
    client: TestClient, auth: FakeAuth, cookie: str | None, slug: str
) -> None:
    if cookie:
        client.cookies.set(ACCESS_TOKEN_COOKIE, cookie, path='/api')

    response = client.get('/api/auth/me')

    assert response.status_code == 401
    assert response.json()['error'] == slug
    assert auth.calls == []


def test_me_of_a_signed_out_session(
    client: TestClient, auth: FakeAuth
) -> None:
    auth.failure = NotAuthenticatedError('revoked')
    client.cookies.set(ACCESS_TOKEN_COOKIE, f'access-{SUB}', path='/api')

    response = client.get('/api/auth/me')

    assert response.status_code == 401
