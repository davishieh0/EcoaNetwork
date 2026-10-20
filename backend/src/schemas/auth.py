"""
Authentication schemas.
"""

# --- IMPORTS ---
from pydantic import BaseModel


# --- CODE ---
class Tokens(BaseModel):
    """
    What a successful sign-in or refresh hands back.
    """
    access_token: str
    expires_in: int
    refresh_token: str | None = None
