"""
Review endpoints: passed on from the indexer.
"""

# --- IMPORTS ---
from fastapi import APIRouter
from fastapi import Depends
from fastapi import Query
from fastapi import Response
from src.clients.indexer import IndexerClient
from src.dependencies.indexer import get_indexer
from src.schemas.reads import MODERATORS_HELP
from src.schemas.reads import ItemPage
from src.schemas.reads import ReviewDetail
from src.schemas.reads import read_responses
from src.utils.relay import relay


# --- GLOBALS ---
router = APIRouter()


# --- CODE ---
@router.get('/{event_id}',
            summary='Get a review',
            response_model=ReviewDetail,
            responses=read_responses(400, 404, 410, 451))
async def get_review(
    event_id: str,
    moderators: str | None = Query(None, description=MODERATORS_HELP),
    indexer: IndexerClient = Depends(get_indexer),
) -> Response:
    """
    One version of a review, any of them, with every version of its chain.
    """
    return relay(await indexer.get_review(event_id=event_id,
                       moderators=moderators))


@router.get('/{event_id}/responses',
            summary="List a review's responses",
            response_model=ItemPage,
            responses=read_responses(400, 404, 410, 451))
async def list_responses(
    event_id: str,
    limit: str | None = Query(None, description='1 to 100; 20 by default.'),
    after: str | None = Query(None, description='The next of a page.'),
    labeled: str | None = Query(None, description='show, to see what '
                                                  'moderation hides, marked.'),
    moderators: str | None = Query(None, description=MODERATORS_HELP),
    indexer: IndexerClient = Depends(get_indexer),
) -> Response:
    """
    The conversation on a review, in order.
    """
    return relay(await indexer.list_review_responses(
        event_id=event_id, limit=limit, after=after, labeled=labeled,
        moderators=moderators,
    ))
