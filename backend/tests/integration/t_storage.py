"""
Integration tests for the base storage and the database sessions.
"""

# --- IMPORTS ---
from src.clients.database import Database
from src.err.data_integrity_error import DataIntegrityError
from tests.integration.conftest import Note
from tests.integration.conftest import NoteStorage
from tests.integration.conftest import Run

import pytest


# --- STORAGE ---
def test_insert_returns_what_the_database_filled_in(run: Run) -> None:
    async def body(database: Database) -> None:
        note = await NoteStorage(database).insert(Note(title='first'))

        assert note.id is not None
        assert note.body == ''
        assert note.created_at.tzinfo is not None

    run(body)


def test_insert_many_stores_the_rows_in_order(run: Run) -> None:
    async def body(database: Database) -> None:
        notes = NoteStorage(database)

        added = await notes.insert_many(
            [Note(title='a'), Note(title='b', body='x'), Note(title='c')]
        )

        assert [n.title for n in added] == ['a', 'b', 'c']
        assert [n.body for n in added] == ['', 'x', '']
        assert all(n.id is not None for n in added)
        assert all(n.created_at.tzinfo is not None for n in added)
        assert [n.id for n in await notes.find_many()] == [n.id for n in added]
        assert await notes.insert_many([]) == []

    run(body)


def test_insert_many_stores_nothing_when_one_row_is_refused(run: Run) -> None:
    async def body(database: Database) -> None:
        notes = NoteStorage(database)
        await notes.insert(Note(title='taken'))

        with pytest.raises(DataIntegrityError):
            await notes.insert_many(
                [Note(title='new'), Note(title='taken'), Note(title='other')]
            )

        assert [n.title for n in await notes.find_many()] == ['taken']

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


def test_find_many_filters_and_pages_in_a_stable_order(run: Run) -> None:
    async def body(database: Database) -> None:
        notes = NoteStorage(database)

        for title, content in [('a', 'x'), ('b', 'y'), ('c', 'x'), ('d', 'x')]:
            await notes.insert(Note(title=title, body=content))

        everything = await notes.find_many()
        matching = await notes.find_many(body='x')
        page = await notes.find_many(body='x', limit=2, offset=1)

        assert [n.title for n in everything] == ['a', 'b', 'c', 'd']
        assert [n.title for n in matching] == ['a', 'c', 'd']
        assert [n.title for n in page] == ['c', 'd']

    run(body)


def test_update_changes_the_row_and_reports_a_missing_one(run: Run) -> None:
    async def body(database: Database) -> None:
        notes = NoteStorage(database)
        note = await notes.insert(Note(title='first'))

        changed = await notes.update(note.id, body='changed')

        stored = await notes.get(note.id)

        assert changed is not None
        assert changed.body == 'changed'
        assert stored is not None
        assert stored.body == 'changed'
        assert await notes.update(note.id + 1, body='nobody') is None

    run(body)
