"""
System endpoints.
"""

# --- IMPORTS ---
from fastapi import APIRouter
from fastapi import Depends
from fastapi import Response
from src.clients.indexer import IndexerClient
from src.dependencies.indexer import get_indexer
from src.dependencies.system import get_health
from src.dependencies.system import get_info
from src.system import Health
from src.system import Info
from src.utils.relay import relay


# --- GLOBALS ---
router = APIRouter()


# --- CODE ---
@router.get('/health',
            summary='Liveness probe',
            response_model=Health)
def health_endpoint(health: Health = Depends(get_health)) -> Health:
    """
    Reports whether the service finished starting up.

    Always answers `200`; read the `status` field.
    """
    return health


@router.get('/info',
            summary='Build metadata',
            response_model=Info)
def info_endpoint(info: Info = Depends(get_info)) -> Info:
    """
    Returns the running build's name, description and version.
    """
    return info


@router.get('/nodes',
            summary='Observed network nodes')
async def nodes_endpoint(
    indexer: IndexerClient = Depends(get_indexer),
) -> Response:
    """
    Returns the nodes this indexer follows and their observed log positions.
    """
    return relay(await indexer.get_nodes())
