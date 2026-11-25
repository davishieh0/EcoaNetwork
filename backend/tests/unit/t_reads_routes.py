"""
Unit tests for the read routes: they pass the indexer's answer on as it is.
"""

# --- IMPORTS ---
from collections.abc import Iterator
from fastapi.testclient import TestClient
from src.app import app
from src.dependencies.indexer import get_indexer
from src.err.indexer_unavailable_error import IndexerUnavailableError
from src.schemas.reads import IndexerAnswer
from typing import Any

import pytest

# wires routes, handlers, middleware and the lifespan onto the app
import src.main  # noqa: F401


# --- GLOBALS ---
TRICKY = (b'{"envelope":"{\\"z\\":1,  \\"a\\":\\"caf\\u00e9\\"}",'
          b'"derived":{},"observed":{}}')


# --- FAKES ---
class FakeIndexer:
    """Stands in for the indexer connection: records each method called,
    with its arguments, and answers the same."""

    def __init__(self) -> None:
        self.calls: list[tuple[str, dict[str, Any]]] = []
        self.answer = IndexerAnswer(status=200, body=TRICKY)
        self.failure: Exception | None = None

    def __getattr__(self, method: str) -> Any:
        async def call(**arguments: Any) -> IndexerAnswer:
            self.calls.append((method, arguments))
            if self.failure:
                raise self.failure
            return self.answer

        return call


# --- FIXTURES ---
@pytest.fixture
def indexer() -> FakeIndexer:
    return FakeIndexer()


@pytest.fixture
def client(indexer: FakeIndexer) -> Iterator[TestClient]:
    app.dependency_overrides[get_indexer] = lambda: indexer
    with TestClient(app) as test_client:
        yield test_client
    app.dependency_overrides.clear()


# --- ROUTES ---
@pytest.mark.parametrize(('url', 'method', 'arguments'), [
    ('/api/entities?q=joao&limit=5&after=x',
     'search_entities', {'q': 'joao', 'limit': '5', 'after': 'x',
                         'category': None, 'moderators': None}),
    ('/api/entities?category=retail',
     'search_entities', {'q': None, 'limit': None, 'after': None,
                         'category': 'retail', 'moderators': None}),
    ('/api/entities',
     'search_entities', {'q': None, 'limit': None, 'after': None,
                         'category': None, 'moderators': None}),
    ('/api/entities/cnpj:12345678000199',
     'get_entity', {'entity': 'cnpj:12345678000199', 'moderators': None}),
    ('/api/entities/cnpj:12345678000199/reviews?labeled=show&limit=2',
     'list_entity_reviews', {'entity': 'cnpj:12345678000199', 'limit': '2',
                             'after': None, 'labeled': 'show',
                             'moderators': None}),
    ('/api/reviews/evt_a',
     'get_review', {'event_id': 'evt_a', 'moderators': None}),
    ('/api/reviews/evt_a/responses?after=y',
     'list_review_responses', {'event_id': 'evt_a', 'limit': None,
                               'after': 'y', 'labeled': None,
                               'moderators': None}),
    ('/api/events/evt_a',
     'get_event', {'event_id': 'evt_a', 'moderators': None}),
    ('/api/profiles?ids=did:key:a,did:key:b',
     'list_profiles', {'ids': 'did:key:a,did:key:b', 'moderators': None}),
    ('/api/profiles/did:key:a',
     'get_profile', {'did': 'did:key:a', 'moderators': None}),
    ('/api/identities/did:key:a', 'get_identity', {'did': 'did:key:a'}),
    ('/api/reviews/evt_a?moderators=did:key:m',
     'get_review', {'event_id': 'evt_a', 'moderators': 'did:key:m'}),
])
def test_each_route_calls_its_indexer_method(
    client: TestClient,
    indexer: FakeIndexer,
    url: str,
    method: str,
    arguments: dict[str, Any],
) -> None:
    # Act
    response = client.get(url)

    # Assert: the route knows the method, never the indexer's paths
    assert response.status_code == 200
    assert indexer.calls == [(method, arguments)]


def test_a_route_passes_only_the_parameters_it_takes(
    client: TestClient, indexer: FakeIndexer,
) -> None:
    # Act
    client.get('/api/reviews/evt_a?limit=1&admin=true&role=x')

    # Assert: nothing a client adds reaches the indexer
    assert indexer.calls == [('get_review', {'event_id': 'evt_a',
                                             'moderators': None})]


def test_the_body_passes_on_byte_for_byte(
    client: TestClient, indexer: FakeIndexer,
) -> None:
    # Act
    response = client.get('/api/events/evt_a')

    # Assert
    assert response.content == TRICKY
    assert response.headers['content-type'] == 'application/json'


@pytest.mark.parametrize('status', [400, 404, 410, 451])
def test_a_failure_the_indexer_explains_passes_on(
    client: TestClient, indexer: FakeIndexer, status: int,
) -> None:
    # Arrange
    body = b'{"error":"review_withdrawn_error","message":"Withdrawn."}'
    indexer.answer = IndexerAnswer(status=status, body=body)

    # Act
    response = client.get('/api/reviews/evt_a')

    # Assert
    assert (response.status_code, response.content) == (status, body)


def test_an_indexer_that_does_not_answer_is_a_503(
    client: TestClient, indexer: FakeIndexer,
) -> None:
    # Arrange
    indexer.failure = IndexerUnavailableError('connection refused')

    # Act
    response = client.get('/api/entities')

    # Assert: the backend's own envelope, nothing of the indexer's details
    assert response.status_code == 503
    assert response.json() == {
        'error': 'indexer_unavailable_error',
        'message': 'Service temporarily unavailable!',
    }


def test_a_path_segment_cannot_reach_another_indexer_route(
    client: TestClient, indexer: FakeIndexer,
) -> None:
    # Act: an encoded slash is decoded before routing, so it matches no route
    response = client.get('/api/entities/..%2Fhealth')

    # Assert: refused here; the indexer is never asked
    assert response.status_code == 404
    assert indexer.calls == []


def test_reads_need_no_account(
    client: TestClient, indexer: FakeIndexer,
) -> None:
    # Act: no session cookie at all
    response = client.get('/api/entities')

    # Assert
    assert response.status_code == 200
