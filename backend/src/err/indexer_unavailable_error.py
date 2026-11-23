"""
Indexer unavailable error.
"""

# --- IMPORTS ---
from src.err.ecoa_error import EcoaError

import logging


# --- ERROR ---
class IndexerUnavailableError(EcoaError):
    """
    Raised when the indexer does not answer a read, or fails it.
    """

    MESSAGE = 'Service temporarily unavailable!'
    STATUS_CODE = 503
    LOG_LEVEL = logging.ERROR
