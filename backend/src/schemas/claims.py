"""
Claim schemas.
"""

# --- IMPORTS ---
from pydantic import BaseModel
from pydantic import Field
from src.schemas.events import Outcome


# --- CODE ---
class HumanClaimRequest(BaseModel):
    """
    Asks for a human claim about an identity, with the solved check.
    """
    subject: str = Field(
        pattern=r'^did:key:z6Mk[1-9A-HJ-NP-Za-km-z]{44}$',
        description='The did:key the claim is about: the one asking.',
        examples=['did:key:z6MkuoUeru2RSnwM5Bpm1B2THsFp6CgYjFrfV4uw9TAubctb'],
    )
    altcha: str = Field(
        min_length=1,
        max_length=4000,
        description='The solved proof of work, as the widget hands it.',
    )


class HumanClaimResult(BaseModel):
    """
    The claim issued, and what the node did with it.
    """
    event_id: str = Field(
        serialization_alias='eventId',
        description='The eventId of the claim.',
    )
    verifier: str = Field(description='The did:key that signed it.')
    outcome: Outcome = Field(description='What became of the claim.')
    reason: str | None = Field(
        default=None,
        description='Why the node rejected it; null otherwise.',
    )
