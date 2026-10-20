"""
Unit tests for the CORS configuration.
"""

# --- IMPORTS ---
from fastapi.testclient import TestClient
from src.middleware.cors import origin_regex

import re


ACAO = 'access-control-allow-origin'
ACAC = 'access-control-allow-credentials'


# --- INTEGRATION ---
def test_localhost_any_port_is_allowed(client: TestClient) -> None:
    for origin in ['http://localhost:5173', 'http://localhost:61234']:
        response = client.get('/api/system/health', headers={'Origin': origin})
        assert response.headers[ACAO] == origin


def test_unknown_origin_gets_no_cors_headers(client: TestClient) -> None:
    response = client.get(
        '/api/system/health', headers={'Origin': 'https://evil.com'}
    )
    assert ACAO not in response.headers


def test_localhost_lookalike_domain_is_rejected(client: TestClient) -> None:
    response = client.get(
        '/api/system/health',
        headers={'Origin': 'http://localhost.evil.com:3000'},
    )
    assert ACAO not in response.headers


def test_preflight_from_allowed_origin_is_answered(client: TestClient) -> None:
    response = client.options(
        '/api/system/health',
        headers={
            'Origin': 'http://localhost:5173',
            'Access-Control-Request-Method': 'GET',
        },
    )
    assert response.status_code == 200
    assert response.headers[ACAO] == 'http://localhost:5173'


def test_allowed_origin_may_send_and_receive_cookies(
    client: TestClient,
) -> None:
    origin = 'http://localhost:5173'

    preflight = client.options(
        '/api/system/health',
        headers={'Origin': origin, 'Access-Control-Request-Method': 'POST'},
    )
    request = client.get('/api/system/health', headers={'Origin': origin})

    for response in (preflight, request):
        assert response.headers[ACAO] == origin
        assert response.headers[ACAC] == 'true'


def test_unknown_origin_cannot_read_a_response_made_with_cookies(
    client: TestClient,
) -> None:
    response = client.get(
        '/api/system/health', headers={'Origin': 'https://evil.com'}
    )

    # the browser only hands the page a response that names its origin;
    # allow-credentials alone, which Starlette always sends, grants nothing
    assert ACAO not in response.headers


# --- UNIT (origin_regex) ---
def test_origin_regex_is_none_without_wildcards() -> None:
    assert origin_regex(['https://ecoa.example']) is None


def test_origin_regex_requires_an_explicit_port() -> None:
    pattern = origin_regex(['http://localhost:*'])
    assert pattern is not None
    assert re.fullmatch(pattern, 'http://localhost:80')
    assert not re.fullmatch(pattern, 'http://localhost')
