"""
Event endpoints.
"""

# --- IMPORTS ---
from fastapi import APIRouter
from fastapi import Depends
from fastapi.responses import JSONResponse
from src.dependencies.events import get_events
from src.dependencies.events import read_envelope
from src.err.invalid_request_error import InvalidRequestError
from src.err.node_unavailable_error import NodeUnavailableError
from src.schemas.errors import error_responses
from src.schemas.events import PublishResult
from src.services.events import EventService


# --- GLOBALS ---
router = APIRouter()
STATUS_BY_OUTCOME = {
    'accepted': 200,
    'already_exists': 200,
    'orphan': 202,
    'rejected': 400,
}


# --- CODE ---
@router.post('',
             summary='Submit a signed event',
             response_model=PublishResult,
             responses=error_responses(InvalidRequestError,
                                       NodeUnavailableError))
async def publish(
    envelope: bytes = Depends(read_envelope),
    events: EventService = Depends(get_events),
) -> JSONResponse:
    """
    Hands the signed envelope to a node and reports what it did.
    """
    result = await events.publish(envelope)

    return JSONResponse(result.model_dump(mode='json'),
                        status_code=STATUS_BY_OUTCOME[result.outcome])
