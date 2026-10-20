"""
Authentication schemas.
"""

# --- IMPORTS ---
from datetime import datetime
from pydantic import BaseModel
from pydantic import Field


# --- CODE ---
class Tokens(BaseModel):
    """
    What a successful sign-in or refresh hands back.
    """
    access_token: str
    expires_in: int
    refresh_token: str | None = None


class AccessClaims(BaseModel):
    """
    What a verified access token says about whoever sent it.
    """
    sub: str
    expires_at: datetime


class AccountAttributes(BaseModel):
    """
    What the identity provider holds about an account.
    """
    email: str
    email_verified: bool


class Account(BaseModel):
    """
    An account as the platform sees it.
    """
    sub: str = Field(description="The account's permanent id.")
    email: str
    handle: str | None = Field(
        description='None while the account has no profile yet: the client '
                    'should ask for a handle before anything else.',
    )


class Session(BaseModel):
    """
    What signing in hands back: the tokens, which travel in cookies, and the
    account, which the client shows.
    """
    tokens: Tokens
    account: Account
