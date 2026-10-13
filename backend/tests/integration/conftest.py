"""
Shared fixtures for the tests that run against a real database.
"""

# --- IMPORTS ---
from collections.abc import Awaitable
from collections.abc import Callable
from collections.abc import Iterator
from datetime import datetime
from sqlalchemy import DateTime
from sqlalchemy import String
from sqlalchemy import func
from sqlalchemy import text
from sqlalchemy.ext.asyncio import create_async_engine
from sqlalchemy.orm import Mapped
from sqlalchemy.orm import mapped_column
from src.clients.database import Database
from src.models.base import Base
from src.storage.base import Storage
from typing import TypedDict
from typing import Unpack

import asyncio
import os
import pytest


# --- GLOBALS ---
# the empty database infra/local creates for tests; never the one in .env
DATABASE_URL = os.environ.get(
    'TEST_DATABASE_URL',
    'postgresql+asyncpg://backend:goodpassword@localhost:5432/'
    'ecoa_backend_test',
)

# the tests drop every table: refuse anything that is not a test database
if not DATABASE_URL.rsplit('/', 1)[-1].endswith('_test'):
    raise RuntimeError('integration tests only run on a *_test database')

# where the table that only exists for these tests lives
SCHEMA = 'storage_tests'

Run = Callable[[Callable[[Database], Awaitable[None]]], None]


# --- MODELS ---
class Note(Base):
    """A table that only exists while these tests run."""

    __tablename__ = 'notes'
    __table_args__ = {'schema': SCHEMA}

    id: Mapped[int] = mapped_column(primary_key=True, init=False)
    title: Mapped[str] = mapped_column(String(50), unique=True)
    body: Mapped[str] = mapped_column(String(200), default='')
    created_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), server_default=func.now(), init=False
    )


class NoteChanges(TypedDict, total=False):
    """The columns of a note that may change."""

    title: str
    body: str


class NoteStorage(Storage[Note]):
    """The base storage, bound to the test table."""

    model = Note

    async def update(
        self, key: int, **changes: Unpack[NoteChanges]
    ) -> Note | None:
        """Changes a note."""
        return await self._update(key, changes)


# --- HELPERS ---
async def _reset(create: bool) -> None:
    """Drops every table and, when asked, builds them all again."""
    engine = create_async_engine(DATABASE_URL)

    async with engine.begin() as connection:
        await connection.execute(
            text(f'DROP SCHEMA IF EXISTS {SCHEMA} CASCADE')
        )
        await connection.run_sync(Base.metadata.drop_all)

        if create:
            await connection.execute(text(f'CREATE SCHEMA {SCHEMA}'))
            await connection.run_sync(Base.metadata.create_all)

    await engine.dispose()


# --- FIXTURES ---
@pytest.fixture
def run() -> Iterator[Run]:
    """
    Runs an async test body against freshly created, empty tables.

    The body receives a Database it can open sessions on.
    """

    def runner(body: Callable[[Database], Awaitable[None]]) -> None:
        async def main() -> None:
            await _reset(create=True)
            database = Database(DATABASE_URL)

            try:
                await body(database)
            finally:
                await database.close()
                await _reset(create=False)

        asyncio.run(main())

    yield runner
