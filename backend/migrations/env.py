"""
Alembic environment: how migrations reach the database.
"""

# --- IMPORTS ---
from alembic import context
from logging.config import fileConfig
from sqlalchemy.engine import Connection
from sqlalchemy.ext.asyncio import create_async_engine
from sqlalchemy.pool import NullPool
from src.config import config
from src.models.base import Base

import asyncio

# registers every model on Base.metadata, so autogenerate can see them
import src.models  # noqa: F401


# --- GLOBALS ---
DATABASE_URL = config.DATABASE_URL.get_secret_value()

# what autogenerate compares the database against
target_metadata = Base.metadata

# set up the loggers declared in alembic.ini
if context.config.config_file_name is not None:
    fileConfig(context.config.config_file_name)


# --- CODE ---
def run_offline() -> None:
    """
    Writes the migrations as SQL instead of running them (--sql).

    :return: None.
    """
    context.configure(
        url=DATABASE_URL,
        target_metadata=target_metadata,
        literal_binds=True,
        dialect_opts={'paramstyle': 'named'},
    )

    with context.begin_transaction():
        context.run_migrations()


def _run(connection: Connection) -> None:
    """
    Runs the migrations on an open connection.

    :param connection: The connection to migrate through.

    :return: None.
    """
    context.configure(connection=connection, target_metadata=target_metadata)

    with context.begin_transaction():
        context.run_migrations()


async def run_online() -> None:
    """
    Connects to the database and runs the migrations.

    :return: None.
    """
    # one short-lived connection: a pool would only outlive the command
    engine = create_async_engine(DATABASE_URL, poolclass=NullPool)

    async with engine.connect() as connection:
        await connection.run_sync(_run)

    await engine.dispose()


if context.is_offline_mode():
    run_offline()
else:
    asyncio.run(run_online())
