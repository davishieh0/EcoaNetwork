"""
Unit tests for the session cookies.
"""

# --- IMPORTS ---
from fastapi import FastAPI
from fastapi import Request
from fastapi import Response
from fastapi.testclient import TestClient
from src.schemas.auth import Tokens
from src.utils.cookies import ACCESS_TOKEN_COOKIE
from src.utils.cookies import REFRESH_TOKEN_COOKIE
from src.utils.cookies import SESSION_MAX_AGE
from src.utils.cookies import clear_session_cookies
from src.utils.cookies import set_session_cookies


# --- HELPERS ---
def _client() -> TestClient:
    """Builds an app that signs in, refreshes, signs out and echoes cookies."""
    app = FastAPI()

    @app.post('/api/auth/login')
    def login(response: Response) -> None:
        set_session_cookies(
            response,
            Tokens(access_token='a1', expires_in=3600, refresh_token='r1'),
        )

    @app.post('/api/auth/refresh')
    def refresh(response: Response) -> None:
        set_session_cookies(
            response, Tokens(access_token='a2', expires_in=3600)
        )

    @app.post('/api/auth/logout')
    def logout(response: Response) -> None:
        clear_session_cookies(response)

    @app.get('/api/auth/cookies')
    def auth_cookies(request: Request) -> dict[str, str]:
        return dict(request.cookies)

    @app.get('/api/reviews/cookies')
    def other_cookies(request: Request) -> dict[str, str]:
        return dict(request.cookies)

    # https, as in production: the cookies are Secure
    return TestClient(app, base_url='https://testserver')


def _set_cookie_headers(client: TestClient, path: str) -> dict[str, str]:
    """Maps each cookie name to the Set-Cookie header that set it."""
    response = client.post(path)

    return {
        header.split('=', 1)[0]: header
        for header in response.headers.get_list('set-cookie')
    }


# --- SETTING ---
def test_sign_in_sets_both_cookies_out_of_reach_of_scripts() -> None:
    headers = _set_cookie_headers(_client(), '/api/auth/login')

    access = headers[ACCESS_TOKEN_COOKIE]
    refresh = headers[REFRESH_TOKEN_COOKIE]

    for header in (access, refresh):
        assert 'HttpOnly' in header
        assert 'Secure' in header
        assert 'SameSite=lax' in header
        assert f'Max-Age={SESSION_MAX_AGE}' in header

    assert 'Path=/api;' in access
    assert 'Path=/api/auth;' in refresh


def test_refresh_replaces_the_access_token_and_keeps_the_refresh_one() -> None:
    client = _client()
    client.post('/api/auth/login')

    headers = _set_cookie_headers(client, '/api/auth/refresh')

    assert set(headers) == {ACCESS_TOKEN_COOKIE}
    assert client.get('/api/auth/cookies').json() == {
        ACCESS_TOKEN_COOKIE: 'a2',
        REFRESH_TOKEN_COOKIE: 'r1',
    }


# --- WHERE THE BROWSER SENDS THEM ---
def test_refresh_token_only_travels_to_the_auth_routes() -> None:
    client = _client()
    client.post('/api/auth/login')

    assert client.get('/api/auth/cookies').json() == {
        ACCESS_TOKEN_COOKIE: 'a1',
        REFRESH_TOKEN_COOKIE: 'r1',
    }
    assert client.get('/api/reviews/cookies').json() == {
        ACCESS_TOKEN_COOKIE: 'a1'
    }


def test_secure_cookies_are_never_sent_over_plain_http() -> None:
    client = _client()
    client.post('/api/auth/login')

    insecure = TestClient(client.app, base_url='http://testserver')
    insecure.cookies = client.cookies

    assert insecure.get('/api/auth/cookies').json() == {}


# --- CLEARING ---
def test_sign_out_removes_both_cookies() -> None:
    client = _client()
    client.post('/api/auth/login')

    headers = _set_cookie_headers(client, '/api/auth/logout')

    # a removal only works with the path the cookie was set with
    assert 'Max-Age=0' in headers[ACCESS_TOKEN_COOKIE]
    assert 'Path=/api;' in headers[ACCESS_TOKEN_COOKIE]
    assert 'Max-Age=0' in headers[REFRESH_TOKEN_COOKIE]
    assert 'Path=/api/auth;' in headers[REFRESH_TOKEN_COOKIE]
    assert client.get('/api/auth/cookies').json() == {}
