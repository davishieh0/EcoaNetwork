"""
Startup and shutdown event handlers.
"""

# --- IMPORTS ---
from fastapi import FastAPI
from src.clients.aws import Aws
from src.config import config
from src.system import Health
from src.system import Info

import logging


# --- GLOBALS ---
logger = logging.getLogger(__name__)


# --- CODE ---
async def on_startup(app: FastAPI) -> None:
    """
    Initialize the service on startup.

    :param app: The FastAPI application instance.

    :return: None.
    """
    # initialize health and info
    app.state.health = Health()
    app.state.info = Info(
        name=app.title,
        description=app.description,
        version=app.version,
        extra={},
    )

    # open the aws connection
    app.state.aws = Aws(config.AWS_REGION, config.AWS_ENDPOINT_URL)

    # set app health as OK
    app.state.health.status = 'OK'

    # log service start
    logger.info('Service started (status=%s)', app.state.health.status)


async def on_shutdown(app: FastAPI) -> None:
    """
    Run on service shutdown.

    :param app: The FastAPI application instance.

    :return: None.
    """
    logger.info('Service shutting down')

    # close the aws connection
    await app.state.aws.close()
