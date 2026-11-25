"""
Profile endpoints: passed on from the indexer.
"""

# --- IMPORTS ---
from fastapi import APIRouter
from fastapi import Depends
from fastapi import Query
from fastapi import Response
from src.clients.indexer import IndexerClient
from src.dependencies.indexer import get_indexer
from src.schemas.reads import MODERATORS_HELP
from src.schemas.reads import Item
from src.schemas.reads import Profiles
from src.schemas.reads import read_responses
from src.utils.relay import relay


# --- GLOBALS ---
router = APIRouter()


# --- CODE ---
@router.get('',
            summary='Get many profiles',
            response_model=Profiles,
            responses=read_responses(400))
async def list_profiles(
    ids: str | None = Query(None, description='1 to 100 did:key, '
                                              'comma-separated.'),
    moderators: str | None = Query(None, description=MODERATORS_HELP),
    indexer: IndexerClient = Depends(get_indexer),
) -> Response:
    """
    The current profiles of some identities, in the order asked, for the
    authors of a page in one call.
    """
    return relay(await indexer.list_profiles(ids=ids,
                       moderators=moderators))


@router.get('/{did}',
            summary='Get a profile',
            response_model=Item,
            responses=read_responses(400, 404, 451))
async def get_profile(
    did: str,
    moderators: str | None = Query(None, description=MODERATORS_HELP),
    indexer: IndexerClient = Depends(get_indexer),
) -> Response:
    """
    An identity's current profile, through any of its keys. A display name is
    verified by nothing.
    """
    return relay(await indexer.get_profile(did=did,
                       moderators=moderators))
