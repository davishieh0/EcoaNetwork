"""
Authentication service.
"""

# --- IMPORTS ---
from src.clients.cognito import CognitoClient
from src.clients.tokens import TokenVerifier
from src.err.handle_taken_error import HandleTakenError
from src.err.invalid_session_error import InvalidSessionError
from src.models.user import User
from src.schemas.auth import Account
from src.schemas.auth import Session
from src.schemas.auth import Tokens
from src.storage.users import UserStorage

import asyncio


# --- CODE ---
class AuthService:
    """
    Accounts and sessions: the rules, whatever carries the request.
    """

    def __init__(
        self,
        cognito: CognitoClient,
        users: UserStorage,
        tokens: TokenVerifier,
    ) -> None:
        """
        Initialize the service.

        :param cognito: The identity provider.
        :param users: The users storage.
        :param tokens: The access token verifier.

        :return: None.
        """
        self._cognito = cognito
        self._users = users
        self._tokens = tokens


    # --- ACCOUNT ---
    async def register(
        self,
        handle: str,
        email: str,
        password: str,
    ) -> Account:
        """
        Creates an account, reserves its handle and emails the confirmation
        code.

        :param handle: The public name.
        :param email: The address the account signs in with.
        :param password: The account's password.

        :raises HandleTakenError: If another account already uses the handle.
        :raises EmailAlreadyRegisteredError: If the email has an account.

        :return: The account created.
        """
        if await self._users.find_by_handle(handle):
            raise HandleTakenError(handle)

        sub = await self._cognito.sign_up(email, password)

        await self._users.insert(User(sub=sub, handle=handle))

        return Account(sub=sub, email=email, handle=handle)


    async def verify(self, email: str, code: str) -> None:
        """
        Confirms an account's email with the code it was sent.

        :param email: The account's address.
        :param code: The confirmation code.

        :return: None.
        """
        await self._cognito.confirm_sign_up(email, code)


    async def resend_verification(self, email: str) -> None:
        """
        Emails a new confirmation code. The previous one stops working.

        :param email: The account's address.

        :return: None.
        """
        await self._cognito.resend_confirmation_code(email)


    async def login(self, email: str, password: str) -> Session:
        """
        Signs an account in.

        :param email: The account's address.
        :param password: The account's password.

        :raises InvalidCredentialsError: If they do not match an account.
        :raises EmailNotVerifiedError: If the email is not confirmed yet.

        :return: The new session's tokens, and the account signed in.
        """
        tokens = await self._cognito.sign_in(email, password)

        # the provider hands back tokens only: read whose they are
        claims = await self._tokens.verify(tokens.access_token)

        return Session(
            tokens=tokens,
            account=await self.me(claims.sub, tokens.access_token),
        )


    async def refresh(
        self,
        access_token: str | None,
        refresh_token: str | None,
    ) -> Tokens:
        """
        Renews a session.

        :param access_token: The session's access token, usually expired.
        :param refresh_token: The session's refresh token.

        :raises InvalidSessionError: If there is no session to renew.
        :raises NotAuthenticatedError: If the access token is not genuine.

        :return: A new access token, without a refresh token.
        """
        # nothing to renew
        if not access_token or not refresh_token:
            raise InvalidSessionError('no session to renew')

        # the access token is expired, but we still need to know whose it is
        claims = await self._tokens.verify(access_token, allow_expired=True)

        return await self._cognito.refresh(refresh_token, claims.sub)


    async def forgot_password(self, email: str) -> None:
        """
        Emails a code that lets the account choose a new password.

        :param email: The account's address.

        :return: None.
        """
        await self._cognito.forgot_password(email)


    async def reset_password(
        self,
        email: str,
        code: str,
        password: str,
    ) -> None:
        """
        Sets a new password with the code the account was sent.

        :param email: The account's address.
        :param code: The reset code.
        :param password: The new password.

        :return: None.
        """
        await self._cognito.reset_password(email, code, password)


    async def me(self, sub: str, access_token: str) -> Account:
        """
        Reads the signed-in account.

        :param sub: The account's id, from its verified access token.
        :param access_token: That same access token.

        :raises NotAuthenticatedError: If the session was signed out.

        :return: The account; its handle is None while it has no profile.
        """
        # the provider and the database answer independently: ask both at once
        attributes, user = await asyncio.gather(
            self._cognito.get_user(access_token),
            self._users.find_by_sub(sub),
        )

        return Account(
            sub=sub,
            email=attributes.email,
            handle=user.handle if user else None,
        )
