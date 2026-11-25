"""
Signing an event, for the one the gateway signs itself: the human claim it
issues as a verifier. Everything else it only passes on.
"""

# --- IMPORTS ---
from base64 import b32encode
from base64 import urlsafe_b64encode
from cryptography.hazmat.primitives import serialization
from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey
from typing import Any

import hashlib
import json


# --- GLOBALS ---
# the signing input's domain separation, as docs/protocol.md fixes it
PREFIX = b'ecoa-event-v1\x00'

# the multicodec of an Ed25519 public key
ED25519_PREFIX = bytes([0xED, 0x01])

BASE58 = '123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz'


# --- CODE ---
def canonicalize(value: Any) -> str:
    """
    Writes a value as JCS (RFC 8785) does, for the values a claim holds:
    objects, strings, integers, booleans and null. Floats are refused, as the
    protocol refuses them.

    :param value: The value.

    :raises TypeError: For a value a payload may not hold.

    :return: The canonical text.
    """
    if value is None or isinstance(value, bool):
        return json.dumps(value)

    if isinstance(value, int):
        return str(value)

    if isinstance(value, str):
        return json.dumps(value, ensure_ascii=False)

    if isinstance(value, dict):
        # JCS orders keys by their UTF-16 code units
        keys = sorted(value, key=lambda k: k.encode('utf-16-be'))
        return '{' + ','.join(f'{canonicalize(k)}:{canonicalize(value[k])}'
                              for k in keys) + '}'

    if isinstance(value, list):
        return '[' + ','.join(canonicalize(v) for v in value) + ']'

    raise TypeError(f'cannot canonicalize {type(value).__name__}')


def did_of(key: Ed25519PrivateKey) -> str:
    """
    The did:key a key signs as.

    :param key: The private key.

    :return: The did:key.
    """
    raw = key.public_key().public_bytes(serialization.Encoding.Raw,
                                        serialization.PublicFormat.Raw)
    data = ED25519_PREFIX + raw

    number = int.from_bytes(data, 'big')
    text = ''
    while number:
        number, digit = divmod(number, 58)
        text = BASE58[digit] + text

    zeros = len(data) - len(data.lstrip(b'\x00'))

    return 'did:key:z' + BASE58[0] * zeros + text


def sign_event(key: Ed25519PrivateKey,
               network: str,
               event_type: str,
               created_at: str,
               payload: dict[str, Any]) -> bytes:
    """
    Signs an event and writes its envelope.

    :param key: The key that signs it.
    :param network: The protocol field, such as ecoa-dev.
    :param event_type: Such as verification.claim_created.
    :param created_at: RFC 3339, UTC, whole seconds.
    :param payload: The payload.

    :return: The envelope's bytes.
    """
    unsigned = {
        'protocol': network,
        'version': 1,
        'author': did_of(key),
        'deviceKey': None,
        'eventType': event_type,
        'createdAt': created_at,
        'payload': payload,
    }

    signing_input = PREFIX + canonicalize(unsigned).encode()
    digest = hashlib.sha256(signing_input).digest()
    event_id = 'evt_' + b32encode(digest).decode().lower().rstrip('=')
    signature = urlsafe_b64encode(key.sign(signing_input)).decode().rstrip('=')

    envelope = {**unsigned, 'eventId': event_id, 'signature': signature}

    return json.dumps(envelope, ensure_ascii=False,
                      separators=(',', ':')).encode()
