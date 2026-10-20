"""
Authentication schemas.
"""

# --- IMPORTS ---
from datetime import datetime
from pydantic import BaseModel


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
