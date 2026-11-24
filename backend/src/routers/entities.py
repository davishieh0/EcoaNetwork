"""
Entity endpoints: passed on from the indexer.
"""

# --- IMPORTS ---
from fastapi import APIRouter
from fastapi import Depends
from fastapi import Query
from fastapi import Response
from src.clients.indexer import IndexerClient
from src.dependencies.indexer import get_indexer
from src.schemas.reads import MODERATORS_HELP
from src.schemas.reads import Entity
from src.schemas.reads import EntityPage
from src.schemas.reads import ItemPage
from src.schemas.reads import read_responses
from src.utils.relay import relay


# --- GLOBALS ---
router = APIRouter()


# --- CODE ---
@router.get('',
            summary='Search entities',
            response_model=EntityPage,
            responses=read_responses(400))
async def search(
    q: str | None = Query(None, description='Part of a name, any case, '
                                            'with or without accents.'),
    limit: str | None = Query(None, description='1 to 100; 20 by default.'),
    after: str | None = Query(None, description='The next of a page.'),
    category: str | None = Query(None, description='One of the protocol\u2019s '
                                                   'categories, such as '
                                                   'food-and-drink.'),
    moderators: str | None = Query(None, description=MODERATORS_HELP),
    indexer: IndexerClient = Depends(get_indexer),
) -> Response:
    """
    Entities with a name that holds q, newest first; every entity without q.
    """
    return relay(await indexer.search_entities(q=q, limit=limit, after=after,
                                               category=category,
                       moderators=moderators))


@router.get('/{entity}',
            summary='Get an entity',
            response_model=Entity,
            responses=read_responses(400, 404))
async def get_entity(
    entity: str,
    moderators: str | None = Query(None, description=MODERATORS_HELP),
    indexer: IndexerClient = Depends(get_indexer),
) -> Response:
    """
    The entity and the value of each of its fields.
    """
    return relay(await indexer.get_entity(entity=entity,
                       moderators=moderators))


@router.get('/{entity}/reviews',
            summary="List an entity's reviews",
            response_model=ItemPage,
            responses=read_responses(400))
async def list_reviews(
    entity: str,
    limit: str | None = Query(None, description='1 to 100; 20 by default.'),
    after: str | None = Query(None, description='The next of a page.'),
    labeled: str | None = Query(None, description='show, to see what '
                                                  'moderation hides, marked.'),
    moderators: str | None = Query(None, description=MODERATORS_HELP),
    indexer: IndexerClient = Depends(get_indexer),
) -> Response:
    """
    The current version of each review, newest first.
    """
    return relay(await indexer.list_entity_reviews(
        entity=entity, limit=limit, after=after, labeled=labeled,
        moderators=moderators,
    ))
