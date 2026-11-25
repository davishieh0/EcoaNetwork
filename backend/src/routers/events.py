"""
Event endpoints.
"""

# --- IMPORTS ---
from fastapi import APIRouter
from fastapi import Depends
from fastapi import Query
from fastapi import Response
from fastapi.responses import JSONResponse
from src.clients.indexer import IndexerClient
from src.dependencies.events import get_events
from src.dependencies.events import read_envelope
from src.dependencies.indexer import get_indexer
from src.err.invalid_request_error import InvalidRequestError
from src.err.node_unavailable_error import NodeUnavailableError
from src.schemas.errors import error_responses
from src.schemas.events import PublishResult
from src.schemas.reads import MODERATORS_HELP
from src.schemas.reads import Item
from src.schemas.reads import read_responses
from src.services.events import EventService
from src.utils.relay import relay


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


@router.get('/{event_id}',
            summary='Get a signed event',
            response_model=Item,
            responses=read_responses(400, 404, 451))
async def get_event(
    event_id: str,
    moderators: str | None = Query(None, description=MODERATORS_HELP),
    indexer: IndexerClient = Depends(get_indexer),
) -> Response:
    """
    The envelope alone, for inspection, passed on from the indexer.
    """
    return relay(await indexer.get_event(event_id=event_id,
                       moderators=moderators))
