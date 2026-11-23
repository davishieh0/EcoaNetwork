"""
Dependency providers for the indexer.
"""

# --- IMPORTS ---
from fastapi import Request
from src.clients.indexer import IndexerClient


# --- CODE ---
def get_indexer(request: Request) -> IndexerClient:
    """
    Provides the indexer connection set up at startup.

    :param request: FastAPI request object.

    :return: The application's indexer connection.
    """
    indexer: IndexerClient = request.app.state.indexer
    return indexer
