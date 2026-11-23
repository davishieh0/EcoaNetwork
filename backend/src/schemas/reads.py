"""
Read schemas.

IndexerAnswer is what the indexer connection hands back. The rest describe
what the indexer answers, for the documentation only: the read routes pass
its body on as bytes and never parse it, so no envelope in it is rewritten on
the way.
"""

# --- IMPORTS ---
from pydantic import BaseModel
from pydantic import Field


# --- CODE ---
class IndexerAnswer(BaseModel):
    """
    What the indexer answered a read, untouched.
    """

    status: int = Field(
        description='The status the indexer answered: 200, or one of the '
                    'failures it explains itself (400, 404, 410, 451).',
    )
    body: bytes = Field(
        description='The body, byte for byte: never parsed, so no envelope '
                    'in it is rewritten.',
    )

