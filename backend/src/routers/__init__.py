"""
HTTP routers.
"""

# --- IMPORTS ---
from fastapi import APIRouter
from fastapi import FastAPI
from src.config import config
from src.routers import auth
from src.routers import claims
from src.routers import entities
from src.routers import events
from src.routers import identities
from src.routers import profiles
from src.routers import reviews
from src.routers import system


# --- CODE ---
def mount(app: FastAPI) -> None:
    """
    Mounts all routers on the application, under the API prefix.

    :param app: The FastAPI application instance.

    :return: None.
    """
    api = APIRouter(prefix=config.API_PREFIX)

    api.include_router(system.router, tags=['system'], prefix='/system')
    api.include_router(auth.router, tags=['auth'], prefix='/auth')
    api.include_router(events.router, tags=['events'], prefix='/events')
    api.include_router(claims.router, tags=['claims'], prefix='/claims')

    # reads: passed on from the indexer, no account needed
    api.include_router(entities.router, tags=['reads'], prefix='/entities')
    api.include_router(reviews.router, tags=['reads'], prefix='/reviews')
    api.include_router(profiles.router, tags=['reads'], prefix='/profiles')
    api.include_router(identities.router, tags=['reads'],
                       prefix='/identities')

    app.include_router(api)
