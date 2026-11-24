"""
Tests for signing an event in the gateway.
"""

# --- IMPORTS ---
from base64 import b32encode
from base64 import urlsafe_b64decode
from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey
from src.utils.protocol import PREFIX
from src.utils.protocol import canonicalize
from src.utils.protocol import did_of
from src.utils.protocol import sign_event

import hashlib
import json
import pytest


# --- GLOBALS ---
SEED = hashlib.sha256(b'ecoa-dev-human-verifier').digest()


# --- CODE ---
def test_canonicalize_orders_keys_and_writes_no_space() -> None:
    # Act
    text = canonicalize({'b': 1, 'a': {'d': None, 'c': True}, 'é': 'ç'})

    # Assert
    assert text == '{"a":{"c":true,"d":null},"b":1,"é":"ç"}'


def test_canonicalize_refuses_a_float() -> None:
    with pytest.raises(TypeError):
        canonicalize({'rating': 4.5})


def test_did_of_is_the_did_key_of_the_key() -> None:
    # the development verifier, as infra/local lists it
    key = Ed25519PrivateKey.from_private_bytes(SEED)

    assert did_of(key) == ('did:key:z6MkuoUeru2RSnwM5Bpm1B2THsFp6CgYjFr'
                           'fV4uw9TAubctb')


def test_sign_event_names_and_signs_what_it_says() -> None:
    # Arrange
    key = Ed25519PrivateKey.from_private_bytes(SEED)

    # Act
    envelope = json.loads(sign_event(key, 'ecoa-dev',
                                     'verification.claim_created',
                                     '2026-11-24T22:00:00Z',
                                     {'claimType': 'human',
                                      'subject': 'did:key:z6Mkx',
                                      'expiresAt': '2028-11-24T22:00:00Z'}))

    # Assert: the eventId is the hash of the signing input, the signature
    # its key's
    unsigned = {k: v for k, v in envelope.items()
                if k not in ('eventId', 'signature')}
    signing_input = PREFIX + canonicalize(unsigned).encode()
    digest = hashlib.sha256(signing_input).digest()

    assert envelope['eventId'] == ('evt_' + b32encode(digest).decode()
                                   .lower().rstrip('='))
    assert envelope['author'] == did_of(key)

    signature = urlsafe_b64decode(envelope['signature'] + '==')
    key.public_key().verify(signature, signing_input)
