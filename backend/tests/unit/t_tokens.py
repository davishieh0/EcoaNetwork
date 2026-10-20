"""
Unit tests for access token verification.
"""

# --- IMPORTS ---
from collections.abc import Callable
from collections.abc import Iterator
from cryptography.hazmat.primitives import serialization
from cryptography.hazmat.primitives.asymmetric import rsa
from fastapi import Depends
from fastapi import FastAPI
from fastapi.testclient import TestClient
from http.server import BaseHTTPRequestHandler
from http.server import ThreadingHTTPServer
from jwt.algorithms import RSAAlgorithm
from src.clients.tokens import TokenVerifier
from src.dependencies.auth import get_current_account
from src.err.access_token_expired_error import AccessTokenExpiredError
from src.err.auth_provider_error import AuthProviderError
from src.err.handlers import register_error_handlers
from src.err.not_authenticated_error import NotAuthenticatedError
from src.schemas.auth import AccessClaims
from src.utils.cookies import ACCESS_TOKEN_COOKIE
from threading import Thread
from typing import Any

import asyncio
import base64
import hashlib
import hmac
import json
import jwt
import pytest
import time


# --- GLOBALS ---
ISSUER = 'https://cognito-idp.sa-east-1.amazonaws.com/sa-east-1_test'
CLIENT_ID = 'test-client-id'
KID = 'pool-key-1'
SUB = '53fc7aea-8051-7059-c165-f3d7f728d9a5'

POOL_KEY = rsa.generate_private_key(public_exponent=65537, key_size=2048)
OTHER_KEY = rsa.generate_private_key(public_exponent=65537, key_size=2048)

Mint = Callable[..., str]


# --- HELPERS ---
class Jwks:
    """Serves the pool's public key the way the user pool does."""

    def __init__(self) -> None:
        jwk = RSAAlgorithm.to_jwk(POOL_KEY.public_key(), as_dict=True)
        body = json.dumps({'keys': [{**jwk, 'kid': KID, 'alg': 'RS256'}]})
        self.requests = 0
        jwks = self

        class Handler(BaseHTTPRequestHandler):
            def do_GET(self) -> None:
                jwks.requests += 1
                self.send_response(200)
                self.send_header('Content-Type', 'application/json')
                self.end_headers()
                self.wfile.write(body.encode())

            def log_message(self, *args: Any) -> None:
                pass

        self.server = ThreadingHTTPServer(('127.0.0.1', 0), Handler)
        self.uri = f'http://127.0.0.1:{self.server.server_port}/jwks.json'
        # a short poll interval, so shutting down does not stall each test
        Thread(
            target=self.server.serve_forever, args=(0.01,), daemon=True
        ).start()


def mint(
    key: Any = POOL_KEY,
    headers: dict[str, Any] | None = None,
    drop: tuple[str, ...] = (),
    **claims: Any,
) -> str:
    """Builds a token the way the user pool would, with overrides."""
    now = int(time.time())
    payload = {
        'sub': SUB,
        'iss': ISSUER,
        'client_id': CLIENT_ID,
        'token_use': 'access',
        'iat': now,
        'exp': now + 3600,
        **claims,
    }
    for name in drop:
        payload.pop(name)

    return jwt.encode(
        payload, key, algorithm='RS256', headers={'kid': KID, **(headers or {})}
    )


def _b64(data: bytes) -> str:
    return base64.urlsafe_b64encode(data).rstrip(b'=').decode()


# --- FIXTURES ---
@pytest.fixture
def jwks() -> Iterator[Jwks]:
    server = Jwks()
    yield server
    server.server.shutdown()


@pytest.fixture
def verifier(jwks: Jwks) -> TokenVerifier:
    return TokenVerifier(ISSUER, jwks.uri, CLIENT_ID)


def _verify(
    verifier: TokenVerifier, token: str, **options: Any
) -> AccessClaims:
    return asyncio.run(verifier.verify(token, **options))


# --- GENUINE TOKENS ---
def test_genuine_token_says_who_sent_it(verifier: TokenVerifier) -> None:
    claims = _verify(verifier, mint())

    assert claims.sub == SUB
    assert claims.expires_at.tzinfo is not None


def test_expired_token_is_told_apart_from_a_bad_one(
    verifier: TokenVerifier,
) -> None:
    expired = mint(exp=int(time.time()) - 60)

    with pytest.raises(AccessTokenExpiredError):
        _verify(verifier, expired)

    # refreshing needs to read who it was, so that one case may skip expiry
    assert _verify(verifier, expired, allow_expired=True).sub == SUB


def test_keys_are_fetched_once_and_reused(
    verifier: TokenVerifier, jwks: Jwks
) -> None:
    for _ in range(3):
        _verify(verifier, mint())

    assert jwks.requests == 1


# --- TOKENS THAT PROVE NOTHING ---
@pytest.mark.parametrize(
    'token',
    [
        pytest.param(mint(iss='https://evil.example/pool'), id='other issuer'),
        pytest.param(mint(token_use='id'), id='id token'),
        pytest.param(mint(client_id='another-app'), id='other app client'),
        pytest.param(mint(drop=('client_id',)), id='claim missing'),
        pytest.param(mint(key=OTHER_KEY), id='signed by another key'),
        pytest.param(mint(headers={'kid': 'unknown'}), id='unknown key id'),
        pytest.param('not-a-token', id='not a token'),
        pytest.param('', id='empty'),
    ],
)
def test_token_that_proves_nothing_is_refused(
    verifier: TokenVerifier, token: str
) -> None:
    with pytest.raises(NotAuthenticatedError):
        _verify(verifier, token)


def test_tampered_token_is_refused(verifier: TokenVerifier) -> None:
    header, _, signature = mint().split('.')
    payload = json.dumps(
        {'sub': 'someone-else', 'iss': ISSUER, 'client_id': CLIENT_ID,
         'token_use': 'access', 'iat': 0, 'exp': int(time.time()) + 3600}
    ).encode()
    tampered = f'{header}.{_b64(payload)}.{signature}'

    with pytest.raises(NotAuthenticatedError):
        _verify(verifier, tampered)


def test_unsigned_token_is_refused(verifier: TokenVerifier) -> None:
    header = _b64(json.dumps({'alg': 'none', 'kid': KID}).encode())
    payload = _b64(json.dumps(
        {'sub': SUB, 'iss': ISSUER, 'client_id': CLIENT_ID,
         'token_use': 'access', 'iat': 0, 'exp': int(time.time()) + 3600}
    ).encode())
    unsigned = f'{header}.{payload}.'

    with pytest.raises(NotAuthenticatedError):
        _verify(verifier, unsigned)


def test_token_signed_with_the_public_key_as_a_secret_is_refused(
    verifier: TokenVerifier,
) -> None:
    # the classic confusion: HS256 with the public key, which anyone has
    public_pem = POOL_KEY.public_key().public_bytes(
        serialization.Encoding.PEM,
        serialization.PublicFormat.SubjectPublicKeyInfo,
    )
    header = _b64(json.dumps({'alg': 'HS256', 'typ': 'JWT',
                              'kid': KID}).encode())
    payload = _b64(json.dumps(
        {'sub': SUB, 'iss': ISSUER, 'client_id': CLIENT_ID,
         'token_use': 'access', 'iat': 0, 'exp': int(time.time()) + 3600}
    ).encode())
    digest = hmac.new(public_pem, f'{header}.{payload}'.encode(),
                      hashlib.sha256).digest()

    with pytest.raises(NotAuthenticatedError):
        _verify(verifier, f'{header}.{payload}.{_b64(digest)}')


# --- KEYS UNAVAILABLE ---
def test_unreachable_keys_are_the_provider_failing() -> None:
    closed = TokenVerifier(ISSUER, 'http://127.0.0.1:1/jwks.json', CLIENT_ID)

    with pytest.raises(AuthProviderError):
        _verify(closed, mint())


# --- DEPENDENCY ---
def _client(verifier: TokenVerifier) -> TestClient:
    app = FastAPI()
    app.state.tokens = verifier
    register_error_handlers(app)

    @app.get('/whoami')
    async def whoami(
        account: AccessClaims = Depends(get_current_account),
    ) -> dict[str, str]:
        return {'sub': account.sub}

    return TestClient(app)


def test_route_reads_the_caller_from_the_cookie(
    verifier: TokenVerifier,
) -> None:
    client = _client(verifier)
    client.cookies.set(ACCESS_TOKEN_COOKIE, mint())

    response = client.get('/whoami')

    assert response.status_code == 200
    assert response.json() == {'sub': SUB}


@pytest.mark.parametrize(
    ('cookie', 'slug'),
    [
        (None, 'not_authenticated_error'),
        ('not-a-token', 'not_authenticated_error'),
        ('expired', 'access_token_expired_error'),
    ],
)
def test_route_refuses_a_caller_without_a_valid_token(
    verifier: TokenVerifier, cookie: str | None, slug: str
) -> None:
    client = _client(verifier)
    if cookie == 'expired':
        cookie = mint(exp=int(time.time()) - 60)
    if cookie:
        client.cookies.set(ACCESS_TOKEN_COOKIE, cookie)

    response = client.get('/whoami')

    assert response.status_code == 401
    assert response.json()['error'] == slug
