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


def test_get_finds_by_primary_key(run: Run) -> None:
    async def body(database: Database) -> None:
        notes = NoteStorage(database)
        note = await notes.insert(Note(title='first'))

        found = await notes.get(note.id)

        assert found is not None
        assert found.title == 'first'
        assert await notes.get(note.id + 1) is None

    run(body)


def test_find_one_matches_columns_by_equality(run: Run) -> None:
    async def body(database: Database) -> None:
        notes = NoteStorage(database)
        await notes.insert(Note(title='first', body='a'))
        second = await notes.insert(Note(title='second', body='b'))

        found = await notes.find_one(title='second')

        assert found is not None
        assert found.id == second.id
        assert await notes.find_one(title='second', body='a') is None
        assert await notes.find_one(title='missing') is None

    run(body)
