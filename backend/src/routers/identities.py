"""
Identity endpoints: passed on from the indexer.
"""

# --- IMPORTS ---
from fastapi import APIRouter
from fastapi import Depends
from fastapi import Response
from src.clients.indexer import IndexerClient
from src.dependencies.indexer import get_indexer
from src.schemas.reads import Identity
from src.schemas.reads import read_responses
from src.utils.relay import relay


# --- GLOBALS ---
router = APIRouter()


# --- CODE ---
@router.get('/{did}',
            summary='Get an identity',
            response_model=Identity,
            responses=read_responses(400))
async def get_identity(
    did: str,
    indexer: IndexerClient = Depends(get_indexer),
) -> Response:
    """
    The identity a key belongs to: its keys, the current one, whether it is
    contested, its revocations.
    """
    return relay(await indexer.get_identity(did=did))
