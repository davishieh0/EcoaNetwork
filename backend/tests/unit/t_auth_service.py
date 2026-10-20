"""
Unit tests for the authentication service.
"""

# --- IMPORTS ---
from src.err.access_token_expired_error import AccessTokenExpiredError
from src.err.email_already_registered_error import EmailAlreadyRegisteredError
from src.err.handle_taken_error import HandleTakenError
from src.err.invalid_session_error import InvalidSessionError
from src.err.not_authenticated_error import NotAuthenticatedError
from src.err.profile_exists_error import ProfileExistsError
from src.models.user import User
from src.schemas.auth import AccessClaims
from src.schemas.auth import AccountAttributes
from src.schemas.auth import Tokens
from src.services.auth import AuthService
from typing import Any

import asyncio
import datetime
import pytest


# --- GLOBALS ---
SUB = 'sub-1'
EMAIL = 'user@example.com'
PASSWORD = 'Passw0rd!'


# --- FAKES ---
class FakeCognito:
    """Stands in for the Cognito client: records calls, fails on request."""

    def __init__(self) -> None:
        self.calls: list[tuple[str, tuple[Any, ...]]] = []
        self.failures: dict[str, Exception] = {}

    def _record(self, name: str, *args: Any) -> None:
        self.calls.append((name, args))
        if name in self.failures:
            raise self.failures[name]

    def called(self, name: str) -> list[tuple[Any, ...]]:
        return [args for called, args in self.calls if called == name]

    async def sign_up(self, email: str, password: str) -> str:
        self._record('sign_up', email, password)
        return SUB

    async def confirm_sign_up(self, email: str, code: str) -> None:
        self._record('confirm_sign_up', email, code)

    async def resend_confirmation_code(self, email: str) -> None:
        self._record('resend_confirmation_code', email)

    async def sign_in(self, email: str, password: str) -> Tokens:
        self._record('sign_in', email, password)
        return Tokens(access_token=f'access-{SUB}', expires_in=3600,
                      refresh_token='refresh')

    async def refresh(self, refresh_token: str, sub: str) -> Tokens:
        self._record('refresh', refresh_token, sub)
        return Tokens(access_token='renewed', expires_in=3600)

    async def sign_out(self, refresh_token: str) -> None:
        self._record('sign_out', refresh_token)

    async def forgot_password(self, email: str) -> None:
        self._record('forgot_password', email)

    async def reset_password(self, email: str, code: str, pw: str) -> None:
        self._record('reset_password', email, code, pw)

    async def get_user(self, access_token: str) -> AccountAttributes:
        self._record('get_user', access_token)
        return AccountAttributes(email=EMAIL, email_verified=True)


class FakeUsers:
    """Stands in for the users storage, in memory."""

    def __init__(self) -> None:
        self.rows: list[User] = []
        self.taken_on_insert = False

    async def find_by_handle(self, handle: str) -> User | None:
        return next((u for u in self.rows if u.handle == handle), None)

    async def find_by_sub(self, sub: str) -> User | None:
        return next((u for u in self.rows if u.sub == sub), None)

    async def insert(self, instance: User) -> User:
        # someone took the handle between the check and the write
        if self.taken_on_insert:
            raise HandleTakenError(instance.handle)
        self.rows.append(instance)
        return instance


class FakeTokens:
    """Accepts 'access-<sub>', reports 'expired-<sub>' as expired."""

    async def verify(
        self, token: str, *, allow_expired: bool = False
    ) -> AccessClaims:
        kind, _, sub = token.partition('-')
        if kind == 'expired' and not allow_expired:
            raise AccessTokenExpiredError(token)
        if kind not in ('access', 'expired'):
            raise NotAuthenticatedError(token)
        return AccessClaims(
            sub=sub, expires_at=datetime.datetime.now(datetime.UTC)
        )


# --- FIXTURES ---
@pytest.fixture
def cognito() -> FakeCognito:
    return FakeCognito()


@pytest.fixture
def users() -> FakeUsers:
    return FakeUsers()


@pytest.fixture
def auth(cognito: FakeCognito, users: FakeUsers) -> AuthService:
    return AuthService(cognito, users, FakeTokens())  # type: ignore[arg-type]


# --- REGISTER ---
def test_register_creates_the_account_and_reserves_the_handle(
    auth: AuthService, cognito: FakeCognito, users: FakeUsers
) -> None:
    account = asyncio.run(auth.register('eduardo', EMAIL, PASSWORD))

    assert (account.sub, account.email, account.handle) == (
        SUB, EMAIL, 'eduardo'
    )
    assert cognito.called('sign_up') == [(EMAIL, PASSWORD)]
    assert [(u.sub, u.handle) for u in users.rows] == [(SUB, 'eduardo')]


def test_register_with_a_taken_handle_creates_nothing(
    auth: AuthService, cognito: FakeCognito, users: FakeUsers
) -> None:
    users.rows.append(User(sub='someone', handle='eduardo'))

    with pytest.raises(HandleTakenError):
        asyncio.run(auth.register('eduardo', EMAIL, PASSWORD))

    assert cognito.called('sign_up') == []


def test_register_with_a_registered_email_stores_no_profile(
    auth: AuthService, cognito: FakeCognito, users: FakeUsers
) -> None:
    cognito.failures['sign_up'] = EmailAlreadyRegisteredError()

    with pytest.raises(EmailAlreadyRegisteredError):
        asyncio.run(auth.register('eduardo', EMAIL, PASSWORD))

    assert users.rows == []


def test_register_when_the_handle_is_taken_mid_way(
    auth: AuthService, cognito: FakeCognito, users: FakeUsers
) -> None:
    users.taken_on_insert = True

    with pytest.raises(HandleTakenError):
        asyncio.run(auth.register('eduardo', EMAIL, PASSWORD))

    # the account exists without a profile: it is asked for a handle later
    assert cognito.called('sign_up') == [(EMAIL, PASSWORD)]
    assert users.rows == []


# --- OPERATIONS THE PROVIDER CARRIES OUT ---
@pytest.mark.parametrize(
    ('method', 'args', 'operation'),
    [
        ('verify', (EMAIL, '123456'), 'confirm_sign_up'),
        ('resend_verification', (EMAIL,), 'resend_confirmation_code'),
        ('forgot_password', (EMAIL,), 'forgot_password'),
        ('reset_password', (EMAIL, '123456', 'N3w-pass'), 'reset_password'),
    ],
)
def test_operation_is_carried_out_by_the_provider(
    auth: AuthService, cognito: FakeCognito,
    method: str, args: tuple[str, ...], operation: str,
) -> None:
    asyncio.run(getattr(auth, method)(*args))

    assert cognito.called(operation) == [args]


# --- LOGIN ---
def test_login_returns_the_tokens_and_the_account(
    auth: AuthService, cognito: FakeCognito, users: FakeUsers
) -> None:
    users.rows.append(User(sub=SUB, handle='eduardo'))

    session = asyncio.run(auth.login(EMAIL, PASSWORD))

    assert session.tokens.refresh_token == 'refresh'
    assert (session.account.sub, session.account.email,
            session.account.handle) == (SUB, EMAIL, 'eduardo')
    # the account comes from the token just issued
    assert cognito.called('sign_in') == [(EMAIL, PASSWORD)]
    assert cognito.called('get_user') == [(f'access-{SUB}',)]


def test_login_of_an_account_without_a_profile(auth: AuthService) -> None:
    session = asyncio.run(auth.login(EMAIL, PASSWORD))

    assert session.account.handle is None


# --- REFRESH ---
def test_refresh_renews_with_the_sub_of_the_expired_token(
    auth: AuthService, cognito: FakeCognito
) -> None:
    tokens = asyncio.run(auth.refresh(f'expired-{SUB}', 'refresh'))

    assert tokens.access_token == 'renewed'
    assert cognito.called('refresh') == [('refresh', SUB)]


@pytest.mark.parametrize(
    ('access', 'refresh', 'error'),
    [
        (f'expired-{SUB}', None, InvalidSessionError),
        (None, 'refresh', InvalidSessionError),
        (None, None, InvalidSessionError),
        ('forged', 'refresh', NotAuthenticatedError),
    ],
    ids=['no refresh token', 'no access token', 'nothing',
         'forged access token'],
)
def test_refresh_without_a_genuine_session_is_refused(
    auth: AuthService, cognito: FakeCognito,
    access: str | None, refresh: str | None, error: type[Exception],
) -> None:
    with pytest.raises(error):
        asyncio.run(auth.refresh(access, refresh))

    assert cognito.called('refresh') == []


# --- LOGOUT ---
def test_logout_revokes_the_session(
    auth: AuthService, cognito: FakeCognito
) -> None:
    asyncio.run(auth.logout('refresh'))

    assert cognito.called('sign_out') == [('refresh',)]


def test_logout_without_a_session_has_nothing_to_revoke(
    auth: AuthService, cognito: FakeCognito
) -> None:
    asyncio.run(auth.logout(None))

    assert cognito.called('sign_out') == []


# --- ME ---
def test_me_combines_the_provider_and_the_profile(
    auth: AuthService, cognito: FakeCognito, users: FakeUsers
) -> None:
    users.rows.append(User(sub=SUB, handle='eduardo'))

    account = asyncio.run(auth.me(SUB, 'access'))

    assert (account.sub, account.email, account.handle) == (
        SUB, EMAIL, 'eduardo'
    )
    assert cognito.called('get_user') == [('access',)]


def test_me_without_a_profile_has_no_handle(auth: AuthService) -> None:
    assert asyncio.run(auth.me(SUB, 'access')).handle is None


def test_me_after_sign_out_is_refused_by_the_provider(
    auth: AuthService, cognito: FakeCognito
) -> None:
    cognito.failures['get_user'] = NotAuthenticatedError('revoked')

    with pytest.raises(NotAuthenticatedError):
        asyncio.run(auth.me(SUB, 'access'))


# --- PROFILE ---
def test_create_profile_completes_an_account_without_a_handle(
    auth: AuthService, cognito: FakeCognito, users: FakeUsers
) -> None:
    account = asyncio.run(auth.create_profile(SUB, 'access', 'eduardo'))

    assert (account.sub, account.email, account.handle) == (
        SUB, EMAIL, 'eduardo'
    )
    assert [(u.sub, u.handle) for u in users.rows] == [(SUB, 'eduardo')]


def test_create_profile_never_changes_a_handle_already_chosen(
    auth: AuthService, users: FakeUsers
) -> None:
    users.rows.append(User(sub=SUB, handle='eduardo'))

    with pytest.raises(ProfileExistsError):
        asyncio.run(auth.create_profile(SUB, 'access', 'another'))

    assert [u.handle for u in users.rows] == ['eduardo']


def test_create_profile_with_a_taken_handle_stores_nothing(
    auth: AuthService, users: FakeUsers
) -> None:
    users.rows.append(User(sub='someone', handle='eduardo'))

    with pytest.raises(HandleTakenError):
        asyncio.run(auth.create_profile(SUB, 'access', 'eduardo'))

    assert [u.sub for u in users.rows] == ['someone']
