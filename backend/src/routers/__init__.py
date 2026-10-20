"""
HTTP routers.
"""

# --- IMPORTS ---
from fastapi import APIRouter
from fastapi import FastAPI
from src.config import config
from src.routers import auth
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

    app.include_router(api)
