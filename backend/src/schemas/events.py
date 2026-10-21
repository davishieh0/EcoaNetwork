"""
Event schemas.
"""

# --- IMPORTS ---
from pydantic import BaseModel
from pydantic import Field
from typing import Literal


# --- GLOBALS ---
Outcome = Literal['accepted', 'already_exists', 'orphan', 'rejected']


# --- CODE ---
class PublishResult(BaseModel):
    """
    What the node did with a submitted event.
    """
    outcome: Outcome = Field(
        description='What became of the event. Branch on this.',
        examples=['accepted'],
    )
    reason: str | None = Field(
        default=None,
        description='Why the node rejected the event; null otherwise.',
    )
