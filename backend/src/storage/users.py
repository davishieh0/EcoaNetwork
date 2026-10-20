"""
Users storage.
"""

# --- IMPORTS ---
from src.models.user import User
from src.storage.base import Storage


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
