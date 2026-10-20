"""
Integration tests for the users storage.
"""

# --- IMPORTS ---
from src.clients.database import Database
from src.err.data_integrity_error import DataIntegrityError
from src.err.handle_taken_error import HandleTakenError
from src.err.invalid_request_error import InvalidRequestError
from src.models.user import User
from src.storage.users import UserStorage
from tests.integration.conftest import Run

import pytest


# --- GLOBALS ---
SUB = '53fc7aea-8051-7059-c165-f3d7f728d9a5'
OTHER_SUB = 'f9e1242b-5012-4b45-a21c-61bef1365cd0'


# --- TESTS ---
def test_insert_fills_in_the_id_and_the_creation_time(run: Run) -> None:
    async def body(database: Database) -> None:
        users = UserStorage(database)
        user = await users.insert(User(sub=SUB, handle='eduardo'))

        # uuid version 7: ordered by time
        assert user.id.version == 7
        assert user.created_at.tzinfo is not None
        assert user.sub == SUB
        assert user.handle == 'eduardo'

    run(body)


def test_ids_sort_in_the_order_the_users_were_created(run: Run) -> None:
    async def body(database: Database) -> None:
        users = UserStorage(database)
        first = await users.insert(User(sub=SUB, handle='first'))
        second = await users.insert(User(sub=OTHER_SUB, handle='second'))

        assert first.id < second.id
        assert [u.handle for u in await users.find_many()] == [
            'first',
            'second',
        ]

    run(body)


def test_find_by_sub_and_by_handle(run: Run) -> None:
    async def body(database: Database) -> None:
        users = UserStorage(database)
        user = await users.insert(User(sub=SUB, handle='eduardo'))

        by_sub = await users.find_by_sub(SUB)
        by_handle = await users.find_by_handle('eduardo')

        assert by_sub is not None and by_sub.id == user.id
        assert by_handle is not None and by_handle.id == user.id

        # an account with no profile yet, and a handle nobody holds
        assert await users.find_by_sub(OTHER_SUB) is None
        assert await users.find_by_handle('nobody') is None

    run(body)


def test_handle_another_account_holds_is_taken(run: Run) -> None:
    async def body(database: Database) -> None:
        users = UserStorage(database)
        await users.insert(User(sub=SUB, handle='eduardo'))

        with pytest.raises(HandleTakenError):
            await users.insert(User(sub=OTHER_SUB, handle='eduardo'))

        # the refused account was not stored
        assert await users.find_by_sub(OTHER_SUB) is None

    run(body)


def test_account_cannot_have_two_profiles(run: Run) -> None:
    async def body(database: Database) -> None:
        users = UserStorage(database)
        await users.insert(User(sub=SUB, handle='eduardo'))

        with pytest.raises(DataIntegrityError) as raised:
            await users.insert(User(sub=SUB, handle='another'))

        assert 'uq_users_sub' in str(raised.value)
        assert await users.find_by_handle('another') is None

    run(body)


@pytest.mark.parametrize(
    'handle',
    [
        'ab',  # too short
        'a' * 31,  # too long
        'Eduardo',  # uppercase
        '1eduardo',  # starts with a digit
        '_eduardo',  # starts with a separator
        'edu.k',  # separators other than '_'
        'edu-k',
        'edu k',
        'eduardo\n',  # nothing may follow the last character
        '',
    ],
)
def test_database_refuses_a_handle_in_the_wrong_format(
    run: Run, handle: str
) -> None:
    async def body(database: Database) -> None:
        users = UserStorage(database)

        with pytest.raises(InvalidRequestError):
            await users.insert(User(sub=SUB, handle=handle))

        assert await users.find_by_sub(SUB) is None

    run(body)


@pytest.mark.parametrize(
    'handle', ['abc', 'a' * 30, 'eduardo', 'edu_k', 'e2e_test_1', 'a__']
)
def test_database_accepts_a_handle_in_the_right_format(
    run: Run, handle: str
) -> None:
    async def body(database: Database) -> None:
        users = UserStorage(database)
        user = await users.insert(User(sub=SUB, handle=handle))

        assert user.handle == handle

    run(body)
