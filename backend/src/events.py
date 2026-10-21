"""
Startup and shutdown event handlers.
"""

# --- IMPORTS ---
from fastapi import FastAPI
from src.clients.aws import Aws
from src.clients.cognito import CognitoClient
from src.clients.database import Database
from src.clients.node import NodeClient
from src.clients.tokens import TokenVerifier
from src.config import config
from src.services.auth import AuthService
from src.services.events import EventService
from src.storage.users import UserStorage
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

    # set up the database connection (it opens on the first query)
    app.state.database = Database(config.DATABASE_URL.get_secret_value())

    # initialize the storages
    app.state.users = UserStorage(app.state.database)

    # open the aws connection
    app.state.aws = Aws(config.AWS_REGION, config.AWS_ENDPOINT_URL)

    # initialize the aws clients
    app.state.cognito = CognitoClient(
        await app.state.aws.client('cognito-idp', signed=False),
        config.COGNITO_CLIENT_ID,
        config.COGNITO_CLIENT_SECRET.get_secret_value(),
    )

    # set up the access token checks
    app.state.tokens = TokenVerifier(
        config.cognito_issuer,
        config.cognito_jwks_uri,
        config.COGNITO_CLIENT_ID,
    )

    # initialize the services
    app.state.auth = AuthService(
        app.state.cognito,
        app.state.users,
        app.state.tokens
    )

    # open the node connections
    app.state.node = NodeClient(config.node_addrs)

    # initialize the event service
    app.state.events = EventService(app.state.node)

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

    # close the node connections
    await app.state.node.close()

    # close the aws connection
    await app.state.aws.close()

    # close the database connections
    await app.state.database.close()
