"""
Base storage.
"""

# --- IMPORTS ---
from sqlalchemy import inspect
from sqlalchemy import select
from src.clients.database import Database
from src.models.base import Base
from typing import Any


# --- GLOBALS ---
DEFAULT_LIMIT = 50


# --- CODE ---
class Storage[Model: Base]:
    """
    The operations every table has.
    """

    model: type[Model]

    def __init__(self, database: Database) -> None:
        """
        Initialize the storage.

        :param database: The database every operation opens a session on.

        :return: None.
        """
        self._database = database


    async def get(self, key: Any) -> Model | None:
        """
        Finds a row by its primary key.

        :param key: The primary key.

        :return: The row, or None when there is none.
        """
        async with self._database.session() as session:
            return await session.get(self.model, key)


    async def find_one(self, **filters: Any) -> Model | None:
        """
        Finds the first row whose columns equal the given values.

        :param filters: Column names and the values they must equal.

        :return: The row, or None when nothing matches.
        """
        statement = select(self.model).filter_by(**filters).limit(1)

        async with self._database.session() as session:
            return (await session.scalars(statement)).first()


    async def find_many(
        self,
        *,
        limit: int = DEFAULT_LIMIT,
        offset: int = 0,
        **filters: Any,
    ) -> list[Model]:
        """
        Finds the rows whose columns equal the given values.

        :param limit: The most rows to return.
        :param offset: How many matching rows to skip first.
        :param filters: Column names and the values they must equal.

        :return: The rows, ordered by primary key so pages do not overlap.
        """
        statement = (
            select(self.model)
            .filter_by(**filters)
            .order_by(*inspect(self.model).primary_key)
            .limit(limit)
            .offset(offset)
        )

        async with self._database.session() as session:
            return list(await session.scalars(statement))


    async def insert(self, instance: Model) -> Model:
        """
        Adds a row.

        :param instance: The row to add.

        :return: The same row, with what the database filled in.
        """
        async with self._database.session() as session:
            session.add(instance)

        return instance
