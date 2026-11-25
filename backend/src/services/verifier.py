"""
Human verifier service.
"""

# --- IMPORTS ---
from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey
from datetime import UTC
from datetime import datetime
from datetime import timedelta
from src.err.invalid_captcha_error import InvalidCaptchaError
from src.err.verifier_disabled_error import VerifierDisabledError
from src.schemas.claims import HumanClaimResult
from src.services.events import EventService
from src.utils.protocol import did_of
from src.utils.protocol import sign_event
from typing import Any

import altcha
import asyncio
import json
import time


# --- GLOBALS ---
# a few seconds of work in a browser, nothing for one person, a cost at scale
WORK = 20_000

# how long a check may wait between being given and being solved
CHALLENGE_TTL = 300

# the longest a human claim may live (docs/protocol.md)
CLAIM_LIFETIME = timedelta(days=730)


# --- CODE ---
class HumanVerifier:
    """
    Issues the human claim: a proof of work solved in the browser, then a
    verification.claim_created signed by this gateway's verifier key, as
    docs/verification.md describes. It identifies nobody and keeps nothing
    but the checks already used.
    """

    def __init__(self,
                 seed: bytes | None,
                 hmac_secret: str,
                 network: str,
                 events: EventService) -> None:
        """
        Initialize the verifier.

        :param seed: The 32 bytes of the verifier key; None issues nothing.
        :param hmac_secret: Signs the checks this gateway gives out.
        :param network: The protocol field the claims are signed with.
        :param events: Hands the claims to the network.

        :return: None.
        """
        self._key = (Ed25519PrivateKey.from_private_bytes(seed)
                     if seed else None)
        self._secret = hmac_secret
        self._network = network
        self._events = events

        # signatures of checks already used, until they expire
        self._used: dict[str, float] = {}


    @property
    def did(self) -> str | None:
        """
        The did:key this verifier signs as, None when it has no key.
        """
        return did_of(self._key) if self._key else None


    def challenge(self) -> dict[str, Any]:
        """
        Gives out a new check.

        :raises VerifierDisabledError: If this gateway has no verifier key.

        :return: The check, as the widget reads it.
        """
        if not self._key:
            raise VerifierDisabledError()

        given = altcha.create_challenge(
            'PBKDF2/SHA-256', WORK,
            hmac_secret=self._secret,
            expires_at=int(time.time()) + CHALLENGE_TTL,
        )

        return given.to_dict()


    async def issue(self, subject: str, solved: str) -> HumanClaimResult:
        """
        Checks the solved proof of work and issues the claim about subject.

        :param subject: The did:key the claim is about.
        :param solved: The widget's payload.

        :raises VerifierDisabledError: If this gateway has no verifier key.
        :raises InvalidCaptchaError: If the check is wrong, expired or used.
        :raises NodeUnavailableError: If no node answers.

        :return: The claim issued and what the node did with it.
        """
        if not self._key:
            raise VerifierDisabledError()

        signature = self._signature_of(solved)
        result = await asyncio.to_thread(
            altcha.verify_solution, solved, self._secret
        )

        if not result.verified or signature is None:
            raise InvalidCaptchaError(result.error or 'not verified')

        self._forget_expired()
        if signature in self._used:
            raise InvalidCaptchaError('used before')

        now = datetime.now(UTC).replace(microsecond=0)
        envelope = sign_event(self._key, self._network,
                              'verification.claim_created',
                              now.strftime('%Y-%m-%dT%H:%M:%SZ'),
                              {
                                  'claimType': 'human',
                                  'subject': subject,
                                  'expiresAt': (now + CLAIM_LIFETIME)
                                               .strftime('%Y-%m-%dT%H:%M:%SZ'),
                              })

        published = await self._events.publish(envelope)
        self._used[signature] = time.time() + CHALLENGE_TTL

        return HumanClaimResult(
            event_id=json.loads(envelope)['eventId'],
            verifier=did_of(self._key),
            outcome=published.outcome,
            reason=published.reason,
        )


    @staticmethod
    def _signature_of(solved: str) -> str | None:
        """
        The signature of the check a payload solves, which names it.

        :param solved: The widget's payload.

        :return: The signature, None when the payload cannot be read.
        """
        try:
            return altcha.Payload.from_base64(solved).challenge.signature
        except Exception:
            return None


    def _forget_expired(self) -> None:
        """
        Drops the used checks that expired: they fail on their own now.

        :return: None.
        """
        now = time.time()
        self._used = {s: t for s, t in self._used.items() if t > now}
