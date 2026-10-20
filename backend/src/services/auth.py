"""
Authentication service.
"""

# --- IMPORTS ---
from src.clients.cognito import CognitoClient
from src.clients.tokens import TokenVerifier
from src.err.handle_taken_error import HandleTakenError
from src.models.user import User
from src.schemas.auth import Account
from src.storage.users import UserStorage


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
