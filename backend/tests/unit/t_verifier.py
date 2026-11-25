"""
Tests for the human verifier.
"""

# --- IMPORTS ---
from collections.abc import Iterator
from fastapi.testclient import TestClient
from src.app import app
from src.dependencies.verifier import get_verifier
from src.err.invalid_captcha_error import InvalidCaptchaError
from src.err.verifier_disabled_error import VerifierDisabledError
from src.schemas.events import PublishResult
from src.services.verifier import HumanVerifier
from typing import Any

import altcha
import asyncio
import hashlib
import json
import pytest

# wires routes, handlers, middleware and the lifespan onto the app
import src.main  # noqa: F401


# --- GLOBALS ---
SEED = hashlib.sha256(b'ecoa-dev-human-verifier').digest()
SUBJECT = 'did:key:z6MknyZhxYKizGbyDyDNBsvHU9Mq7uCkC4qV3oVxHhZdBjGh'


# --- CODE ---
class FakeEvents:
    """
    Stands in for the event service, keeping what it was handed.
    """

    def __init__(self) -> None:
        self.published: list[dict[str, Any]] = []


    async def publish(self, envelope: bytes) -> PublishResult:
        self.published.append(json.loads(envelope))
        return PublishResult(outcome='accepted')


def solved(verifier: HumanVerifier) -> str:
    """
    Solves one of verifier's checks, as the widget does in a browser.
    """
    challenge = altcha.v2.Challenge.from_dict(verifier.challenge())
    solution = altcha.solve_challenge(challenge)
    assert solution is not None

    return altcha.Payload(challenge=challenge, solution=solution).to_base64()


@pytest.fixture
def events() -> FakeEvents:
    return FakeEvents()


@pytest.fixture
def verifier(events: FakeEvents) -> HumanVerifier:
    return HumanVerifier(SEED, 'secret', 'ecoa-dev', events)  # type: ignore[arg-type]


def test_a_solved_check_issues_a_human_claim(verifier: HumanVerifier,
                                             events: FakeEvents) -> None:
    # Act
    result = asyncio.run(verifier.issue(SUBJECT, solved(verifier)))

    # Assert
    assert result.outcome == 'accepted'
    claim = events.published[0]
    assert claim['eventType'] == 'verification.claim_created'
    assert claim['author'] == verifier.did
    assert claim['payload']['claimType'] == 'human'
    assert claim['payload']['subject'] == SUBJECT
    assert 'entity' not in claim['payload']
    assert 'evidenceRef' not in claim['payload']
    assert result.event_id == claim['eventId']


def test_a_check_is_used_once(verifier: HumanVerifier) -> None:
    # Arrange
    payload = solved(verifier)
    asyncio.run(verifier.issue(SUBJECT, payload))

    # Act / Assert
    with pytest.raises(InvalidCaptchaError):
        asyncio.run(verifier.issue(SUBJECT, payload))


def test_a_failed_publish_does_not_burn_the_check() -> None:
    # Arrange
    class FlakyEvents(FakeEvents):
        def __init__(self) -> None:
            super().__init__()
            self.fail_next = True

        async def publish(self, envelope: bytes) -> PublishResult:
            if self.fail_next:
                self.fail_next = False
                raise RuntimeError('node unavailable')
            return await super().publish(envelope)

    events = FlakyEvents()
    verifier = HumanVerifier(SEED, 'secret', 'ecoa-dev', events)  # type: ignore[arg-type]
    payload = solved(verifier)

    # Act: first attempt fails to publish, second attempt succeeds with same solved check
    with pytest.raises(RuntimeError):
        asyncio.run(verifier.issue(SUBJECT, payload))
    result = asyncio.run(verifier.issue(SUBJECT, payload))

    # Assert
    assert result.outcome == 'accepted'
    assert len(events.published) == 1


def test_a_check_signed_by_another_gateway_fails(events: FakeEvents) -> None:
    # Arrange: solved against another secret
    other = HumanVerifier(SEED, 'another', 'ecoa-dev', events)  # type: ignore[arg-type]
    payload = solved(other)
    verifier = HumanVerifier(SEED, 'secret', 'ecoa-dev', events)  # type: ignore[arg-type]

    # Act / Assert
    with pytest.raises(InvalidCaptchaError):
        asyncio.run(verifier.issue(SUBJECT, payload))
    assert events.published == []


def test_garbage_fails(verifier: HumanVerifier) -> None:
    with pytest.raises(InvalidCaptchaError):
        asyncio.run(verifier.issue(SUBJECT, 'not a payload'))


def test_without_a_key_nothing_is_issued(events: FakeEvents) -> None:
    verifier = HumanVerifier(None, 'secret', 'ecoa-dev', events)  # type: ignore[arg-type]

    with pytest.raises(VerifierDisabledError):
        verifier.challenge()


@pytest.fixture
def client(verifier: HumanVerifier) -> Iterator[TestClient]:
    app.dependency_overrides[get_verifier] = lambda: verifier
    with TestClient(app) as test_client:
        yield test_client
    app.dependency_overrides.clear()


def test_the_routes_give_a_check_and_issue_the_claim(
    client: TestClient,
    verifier: HumanVerifier,
) -> None:
    # Act
    given = client.get('/api/claims/human/challenge')
    challenge = altcha.v2.Challenge.from_dict(given.json())
    solution = altcha.solve_challenge(challenge)
    assert solution is not None
    payload = altcha.Payload(challenge=challenge, solution=solution)
    issued = client.post('/api/claims/human',
                         json={'subject': SUBJECT,
                               'altcha': payload.to_base64()})

    # Assert
    assert given.status_code == 200
    assert issued.status_code == 200
    assert issued.json()['outcome'] == 'accepted'
    assert issued.json()['eventId'].startswith('evt_')
    assert issued.json()['verifier'] == verifier.did


def test_the_route_refuses_a_subject_that_is_not_a_did(
    client: TestClient,
) -> None:
    response = client.post('/api/claims/human',
                           json={'subject': 'someone', 'altcha': 'x'})

    assert response.status_code == 422
