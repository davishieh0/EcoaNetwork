"""
Integration tests for the base storage and the database sessions.
"""

# --- IMPORTS ---
from src.clients.database import Database
from tests.integration.conftest import Note
from tests.integration.conftest import NoteStorage
from tests.integration.conftest import Run


# --- STORAGE ---
def test_insert_returns_what_the_database_filled_in(run: Run) -> None:
    async def body(database: Database) -> None:
        note = await NoteStorage(database).insert(Note(title='first'))

        assert note.id is not None
        assert note.body == ''
        assert note.created_at.tzinfo is not None

    run(body)
