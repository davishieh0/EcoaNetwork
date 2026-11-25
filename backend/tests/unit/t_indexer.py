"""
Unit tests for IndexerClient: an indexer stands in through httpx's mock
transport.
"""

# --- IMPORTS ---
from collections.abc import Awaitable
from collections.abc import Callable
from src.clients.indexer import IndexerClient
from src.err.indexer_unavailable_error import IndexerUnavailableError
from src.schemas.reads import IndexerAnswer
from typing import Any

import asyncio
import httpx
import pytest


# --- GLOBALS ---
# what a JSON parser on the way would change: spacing, an escape, key order
TRICKY = (b'{"envelope":"{\\"z\\":1,  \\"a\\":\\"caf\\u00e9\\"}",'
          b'"derived":{},"observed":{}}')

ENTITY = 'cnpj:12345678000199'


# --- HELPERS ---
def client(handler: Callable[[httpx.Request], httpx.Response]) -> IndexerClient:
    """An IndexerClient whose indexer is handler."""
    return IndexerClient('http://indexer',
                         transport=httpx.MockTransport(handler))


def run(coro: Awaitable[Any]) -> Any:
    """Runs one coroutine to its end."""
    async def wait() -> Any:
        return await coro

    return asyncio.run(wait())


def recorder() -> tuple[IndexerClient, list[httpx.Request]]:
    """An IndexerClient whose indexer records each request and answers {}."""
    seen: list[httpx.Request] = []

    def handler(request: httpx.Request) -> httpx.Response:
        seen.append(request)
        return httpx.Response(200, content=b'{}')

    return client(handler), seen


# --- ROUTES ---
@pytest.mark.parametrize(('call', 'path', 'params'), [
    (lambda c: c.search_entities(q='joao', limit='5', after=None),
     b'/entities', {'q': 'joao', 'limit': '5'}),
    (lambda c: c.search_entities(q=None, limit=None, after=None,
                                 category='retail'),
     b'/entities', {'category': 'retail'}),
    (lambda c: c.search_entities(q=None, limit=None, after=None),
     b'/entities', {}),
    (lambda c: c.get_entity(ENTITY),
     b'/entities/' + ENTITY.encode(), {}),
    (lambda c: c.list_entity_reviews(ENTITY, limit='2', after='x',
                                     labeled='show'),
     b'/entities/' + ENTITY.encode() + b'/reviews',
     {'limit': '2', 'after': 'x', 'labeled': 'show'}),
    (lambda c: c.get_review('evt_a'), b'/reviews/evt_a', {}),
    (lambda c: c.list_review_responses('evt_a', limit=None, after='y',
                                       labeled=None),
     b'/reviews/evt_a/responses', {'after': 'y'}),
    (lambda c: c.get_event('evt_a'), b'/events/evt_a', {}),
    (lambda c: c.list_profiles('did:key:a,did:key:b'),
     b'/profiles', {'ids': 'did:key:a,did:key:b'}),
    (lambda c: c.get_profile('did:key:a'), b'/profiles/did:key:a', {}),
    (lambda c: c.get_identity('did:key:a'), b'/identities/did:key:a', {}),
])
def test_each_method_reads_its_route(
    call: Callable[[IndexerClient], Awaitable[IndexerAnswer]],
    path: bytes,
    params: dict[str, str],
) -> None:
    # Arrange
    indexer, seen = recorder()

    # Act
    run(call(indexer))

    # Assert: the route, and only the parameters given
    assert seen[0].url.raw_path.split(b'?')[0] == path
    assert dict(seen[0].url.params) == params


# --- ANSWERS ---
def test_the_body_comes_back_byte_for_byte() -> None:
    # Arrange
    indexer = client(lambda request: httpx.Response(200, content=TRICKY))

    # Act
    answer = run(indexer.get_event('evt_a'))

    # Assert
    assert (answer.status, answer.body) == (200, TRICKY)


@pytest.mark.parametrize('status', [400, 404, 410, 451])
def test_a_failure_the_indexer_explains_comes_back_as_an_answer(
    status: int,
) -> None:
    # Arrange
    body = b'{"error":"review_withdrawn_error","message":"Withdrawn."}'
    indexer = client(lambda request: httpx.Response(status, content=body))

    # Act
    answer = run(indexer.get_review('evt_a'))

    # Assert
    assert (answer.status, answer.body) == (status, body)


# --- FAILURES ---
@pytest.mark.parametrize('status', [500, 503])
def test_an_indexer_failing_on_its_side_raises(status: int) -> None:
    # Arrange
    indexer = client(lambda request: httpx.Response(status, content=b'{}'))

    # Act / Assert
    with pytest.raises(IndexerUnavailableError):
        run(indexer.get_entity(ENTITY))


@pytest.mark.parametrize('failure', [
    httpx.ConnectError('refused'),
    httpx.ReadTimeout('too slow'),
])
def test_an_indexer_that_does_not_answer_raises(failure: Exception) -> None:
    # Arrange
    def handler(request: httpx.Request) -> httpx.Response:
        raise failure

    # Act / Assert
    with pytest.raises(IndexerUnavailableError):
        run(client(handler).get_entity(ENTITY))


# --- PATHS ---
@pytest.mark.parametrize(('entity', 'escaped'), [
    ('cnpj:12345678000199', b'cnpj:12345678000199'),
    ('did:key:z6Mk', b'did:key:z6Mk'),
    ('domain:a.com.br', b'domain:a.com.br'),
    ('../health', b'..%2Fhealth'),
    ('..', b'%2E%2E'),
    ('.', b'%2E'),
    ('a b?c', b'a%20b%3Fc'),
])
def test_an_id_stays_one_segment_whatever_it_holds(
    entity: str, escaped: bytes,
) -> None:
    # Arrange
    indexer, seen = recorder()

    # Act
    run(indexer.list_entity_reviews(entity, limit=None, after=None,
                                    labeled=None))

    # Assert: escaped, colons kept
    assert seen[0].url.raw_path == b'/entities/' + escaped + b'/reviews'
