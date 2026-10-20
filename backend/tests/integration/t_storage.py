"""
Integration tests for the base storage and the database sessions.
"""

# --- IMPORTS ---
from sqlalchemy import text
from src.clients.database import Database
from src.err.data_integrity_error import DataIntegrityError
from src.err.database_unavailable_error import DatabaseUnavailableError
from src.err.invalid_request_error import InvalidRequestError
from tests.integration.conftest import DATABASE_URL
from tests.integration.conftest import SCHEMA
from tests.integration.conftest import Note
from tests.integration.conftest import NoteStorage
from tests.integration.conftest import Run

import asyncio
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


def test_delete_removes_the_row_and_reports_a_missing_one(run: Run) -> None:
    async def body(database: Database) -> None:
        notes = NoteStorage(database)
        note = await notes.insert(Note(title='first'))

        assert await notes.delete(note.id) is True
        assert await notes.get(note.id) is None
        assert await notes.delete(note.id) is False

    run(body)


def test_update_many_changes_only_the_given_rows(run: Run) -> None:
    async def body(database: Database) -> None:
        notes = NoteStorage(database)
        a, b, c = await notes.insert_many(
            [Note(title='a'), Note(title='b'), Note(title='c')]
        )

        changed = await notes.update_many([a.id, c.id, c.id + 100], body='x')

        assert changed == 2
        assert [n.body for n in await notes.find_many()] == ['x', '', 'x']
        assert await notes.update_many([], body='y') == 0
        assert await notes.update_many([b.id]) == 0

    run(body)


def test_update_many_changes_nothing_when_one_row_is_refused(run: Run) -> None:
    async def body(database: Database) -> None:
        notes = NoteStorage(database)
        a, b = await notes.insert_many([Note(title='a'), Note(title='b')])

        # two rows cannot end up with the same unique title
        with pytest.raises(DataIntegrityError):
            await notes.update_many([a.id, b.id], title='same')

        assert [n.title for n in await notes.find_many()] == ['a', 'b']

    run(body)


def test_delete_many_removes_only_the_given_rows(run: Run) -> None:
    async def body(database: Database) -> None:
        notes = NoteStorage(database)
        a, b, c = await notes.insert_many(
            [Note(title='a'), Note(title='b'), Note(title='c')]
        )

        removed = await notes.delete_many([a.id, c.id, c.id + 100])

        assert removed == 2
        assert [n.title for n in await notes.find_many()] == ['b']
        assert await notes.delete_many([]) == 0
        assert len(await notes.find_many()) == 1

    run(body)


# --- SESSIONS ---
def test_session_commits_when_the_block_ends(run: Run) -> None:
    async def body(database: Database) -> None:
        async with database.session() as session:
            await session.execute(
                text(f"INSERT INTO {SCHEMA}.notes (title, body) "
                     "VALUES ('kept', '')")
            )

        assert await NoteStorage(database).find_one(title='kept')

    run(body)


def test_session_rolls_back_when_the_block_raises(run: Run) -> None:
    async def body(database: Database) -> None:
        # an error the backend already knows keeps its own type
        with pytest.raises(InvalidRequestError):
            async with database.session() as session:
                await session.execute(
                    text(f"INSERT INTO {SCHEMA}.notes (title, body) "
                         "VALUES ('lost', '')")
                )
                raise InvalidRequestError('the operation gave up')

        assert await NoteStorage(database).find_one(title='lost') is None

    run(body)


def test_bug_inside_the_block_is_not_blamed_on_the_database(run: Run) -> None:
    async def body(database: Database) -> None:
        with pytest.raises(TypeError):
            async with database.session() as session:
                await session.execute(
                    text(f"INSERT INTO {SCHEMA}.notes (title, body) "
                         "VALUES ('lost', '')")
                )
                raise TypeError('a mistake in the code')

        # the write that came before the bug was still rolled back
        assert await NoteStorage(database).find_one(title='lost') is None

    run(body)


def test_broken_rule_is_a_conflict_with_a_predictable_name(run: Run) -> None:
    async def body(database: Database) -> None:
        notes = NoteStorage(database)
        await notes.insert(Note(title='taken'))

        with pytest.raises(DataIntegrityError) as raised:
            await notes.insert(Note(title='taken'))

        # the name storage reads to tell which rule was broken
        assert 'uq_notes_title' in str(raised.value)

        # the failed write left nothing behind and broke nothing
        assert len(await notes.find_many()) == 1

    run(body)


def test_value_the_column_cannot_hold_is_a_conflict(run: Run) -> None:
    async def body(database: Database) -> None:
        notes = NoteStorage(database)

        # the title column holds 50 characters
        with pytest.raises(DataIntegrityError):
            await notes.insert(Note(title='x' * 51))

        assert await notes.find_many() == []

    run(body)


def test_statement_that_fails_is_reported_as_unavailable(run: Run) -> None:
    async def body(database: Database) -> None:
        with pytest.raises(DatabaseUnavailableError):
            async with database.session() as session:
                await session.execute(text('SELECT * FROM no_such_table'))

    run(body)


# --- UNREACHABLE DATABASE ---
@pytest.mark.parametrize(
    'url',
    [
        DATABASE_URL.replace(':5432', ':5999'),
        DATABASE_URL.replace('@localhost', '@no-such-host.invalid'),
        DATABASE_URL.replace(':goodpassword@', ':wrong-password@'),
        DATABASE_URL.replace('/ecoa_backend', '/no_such_database'),
    ],
    ids=['closed port', 'unknown host', 'wrong password', 'unknown database'],
)
def test_unreachable_database_is_reported_as_unavailable(url: str) -> None:
    async def body() -> None:
        database = Database(url)

        try:
            with pytest.raises(DatabaseUnavailableError) as raised:
                await NoteStorage(database).find_one(title='any')

            # the details are logged: they must never carry the password
            assert 'goodpassword' not in str(raised.value)
            assert 'wrong-password' not in str(raised.value)
        finally:
            await database.close()

    asyncio.run(body())
