"""
Base storage.
"""

# --- IMPORTS ---
from src.clients.database import Database
from src.models.base import Base


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


    async def insert(self, instance: Model) -> Model:
        """
        Adds a row.

        :param instance: The row to add.

        :return: The same row, with what the database filled in.
        """
        async with self._database.session() as session:
            session.add(instance)

        return instance
