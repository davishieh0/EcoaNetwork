"""
Passes the indexer's answer on to the client.
"""

# --- IMPORTS ---
from fastapi import Response
from src.schemas.reads import IndexerAnswer


# --- CODE ---
def relay(answer: IndexerAnswer) -> Response:
    """
    Answers with exactly what the indexer answered.

    The body goes on as bytes, never parsed: every envelope in it is the one
    that was signed, and the client checks the signature over it.

    :param answer: What the indexer answered.

    :return: The indexer's status and body.
    """
    return Response(
        content=answer.body,
        status_code=answer.status,
        media_type='application/json',
    )
