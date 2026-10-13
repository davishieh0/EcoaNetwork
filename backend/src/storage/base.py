"""
Base storage.
"""

# --- IMPORTS ---
from collections.abc import Mapping
from collections.abc import Sequence
from sqlalchemy import ColumnElement
from sqlalchemy import CursorResult
from sqlalchemy import delete
from sqlalchemy import inspect
from sqlalchemy import select
from sqlalchemy import tuple_
from sqlalchemy import update
from src.clients.database import Database
from src.models.base import Base
from typing import Any
from typing import cast


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


    def _among(self, keys: Sequence[Any]) -> ColumnElement[bool]:
        """
        Builds the condition matching the rows with the given primary keys.

        :param keys: The primary keys; tuples when the key spans columns.

        :return: The condition.
        """
        columns = inspect(self.model).primary_key

        if len(columns) == 1:
            return columns[0].in_(keys)

        return tuple_(*columns).in_(keys)


    async def _update(
        self, key: Any, values: Mapping[str, Any]
    ) -> Model | None:
        """
        Changes a row. Storages whose table has columns that may change
        expose it as update, typed with those columns.

        :param key: The primary key of the row to change.
        :param values: Column names and their new values.

        :return: The changed row, or None when there is none with that key.
        """
        async with self._database.session() as session:
            instance = await session.get(self.model, key)

            # nothing to change
            if instance is None:
                return None

            for column, value in values.items():
                setattr(instance, column, value)

        return instance


    async def _update_many(
        self, keys: Sequence[Any], values: Mapping[str, Any]
    ) -> int:
        """
        Changes several rows to the same values, in one statement: either all
        of them change, or none. Exposed as update_many, like _update.

        :param keys: The primary keys of the rows to change.
        :param values: Column names and their new values.

        :return: How many rows changed. Keys with no row are skipped.
        """
        # nothing to change
        if not keys or not values:
            return 0

        statement = (
            update(self.model).where(self._among(keys)).values(dict(values))
        )

        async with self._database.session() as session:
            result = await session.execute(statement)

        return cast(CursorResult[Any], result).rowcount


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


    async def insert_many(self, instances: list[Model]) -> list[Model]:
        """
        Adds several rows together: either all of them are stored, or none.

        :param instances: The rows to add.

        :return: The same rows, in the given order, with what the database
                 filled in.
        """
        # nothing to add
        if not instances:
            return []

        async with self._database.session() as session:
            session.add_all(instances)

        return instances


    async def delete(self, key: Any) -> bool:
        """
        Removes a row.

        :param key: The primary key of the row to remove.

        :return: True when a row was removed, False when there was none.
        """
        async with self._database.session() as session:
            instance = await session.get(self.model, key)

            # nothing to remove
            if instance is None:
                return False

            await session.delete(instance)

        return True


    async def delete_many(self, keys: Sequence[Any]) -> int:
        """
        Removes several rows in one statement: either all of them go, or none.

        :param keys: The primary keys of the rows to remove.

        :return: How many rows were removed. Keys with no row are skipped.
        """
        # nothing to remove
        if not keys:
            return 0

        statement = delete(self.model).where(self._among(keys))

        async with self._database.session() as session:
            result = await session.execute(statement)

        return cast(CursorResult[Any], result).rowcount
