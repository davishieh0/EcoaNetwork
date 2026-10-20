"""
Authentication schemas.
"""

# --- IMPORTS ---
from datetime import datetime
from pydantic import BaseModel
from pydantic import EmailStr
from pydantic import Field
from src.models.user import HANDLE_PATTERN
from typing import Annotated


# --- GLOBALS ---
Password = Annotated[str, Field(min_length=8, max_length=256)]
Code = Annotated[str, Field(pattern=r'^[0-9]{6}$')]
Handle = Annotated[str, Field(pattern=HANDLE_PATTERN)]


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


class RegisterRequest(BaseModel):
    """
    Creates an account.
    """
    handle: Handle
    email: EmailStr
    password: Password


class EmailRequest(BaseModel):
    """
    Names an account by its email.
    """
    email: EmailStr


class VerifyRequest(BaseModel):
    """
    Confirms an account's email with the code it was sent.
    """
    email: EmailStr
    code: Code


class LoginRequest(BaseModel):
    """
    Signs an account in.
    """
    email: EmailStr

    # no length rule here: a wrong password is answered as wrong, not invalid
    password: Annotated[str, Field(min_length=1, max_length=256)]


class ResetPasswordRequest(BaseModel):
    """
    Sets a new password with the code the account was sent.
    """
    email: EmailStr
    code: Code
    password: Password


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
