"""
Users storage.
"""

# --- IMPORTS ---
from src.err.data_integrity_error import DataIntegrityError
from src.err.ecoa_error import EcoaError
from src.err.handle_taken_error import HandleTakenError
from src.err.invalid_request_error import InvalidRequestError
from src.models.user import User
from src.storage.base import Storage


# --- GLOBALS ---
BROKEN_RULES: dict[str, type[EcoaError]] = {
    'uq_users_handle': HandleTakenError,
    'ck_users_handle_format': InvalidRequestError,
}


# --- CODE ---
class UserStorage(Storage[User]):
    """
    Storage responsible for the users table.
    """

    model = User

    async def find_by_sub(self, sub: str) -> User | None:
        """
        Finds the user behind an identity provider account.

        :param sub: The account's permanent id in the identity provider.

        :return: The user, or None when the account has no profile yet.
        """
        return await self.find_one(sub=sub)


    async def find_by_handle(self, handle: str) -> User | None:
        """
        Finds the user holding a handle.

        :param handle: The handle.

        :return: The user, or None when the handle is free.
        """
        return await self.find_one(handle=handle)


    async def insert(self, instance: User) -> User:
        """
        Adds a user.

        :param instance: The user to add.

        :raises HandleTakenError: If another account already uses the handle.
        :raises InvalidRequestError: If the handle is not in the right format.

        :return: The user, with what the database filled in.
        """
        try:
            return await super().insert(instance)

        # the database names the rule that was broken: say which one it was
        except DataIntegrityError as error:
            details = str(error.args[1])

            for constraint, translated in BROKEN_RULES.items():
                if constraint in details:
                    raise translated(details) from error

            raise
