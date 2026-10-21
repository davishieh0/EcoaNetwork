"""
Dependency providers for events.
"""

# --- IMPORTS ---
from fastapi import Request
from pydantic import Field
from pydantic import Json
from pydantic import TypeAdapter
from pydantic import ValidationError
from src.err.invalid_request_error import InvalidRequestError
from src.services.events import EventService
from typing import Annotated
from typing import Any


# --- GLOBALS ---
MAX_ENVELOPE = 16 * 1024
SIZE_VALIDATOR: TypeAdapter[bytes] = TypeAdapter(
    Annotated[bytes, Field(max_length=MAX_ENVELOPE)]
)
SYNTAX_VALIDATOR: TypeAdapter[Any] = TypeAdapter(Json)


# --- CODE ---
def get_events(request: Request) -> EventService:
    """
    Provides the event service set up at startup.

    :param request: FastAPI request object.

    :return: The application's event service.
    """
    events: EventService = request.app.state.events

    return events


async def read_envelope(request: Request) -> bytes:
    """
    Reads the signed envelope from the request body, without re-encoding it.

    :param request: FastAPI request object.

    :raises InvalidRequestError: If the body is over the protocol ceiling or
        is not JSON.

    :return: The body, byte for byte as the client sent it.
    """
    envelope = await request.body()

    try:
        SIZE_VALIDATOR.validate_python(envelope)
        SYNTAX_VALIDATOR.validate_python(envelope)

    # error messages only, without the envelope itself
    except ValidationError as error:
        reasons = [e['msg'] for e in error.errors(include_input=False)]
        raise InvalidRequestError(reasons) from error

    return envelope
