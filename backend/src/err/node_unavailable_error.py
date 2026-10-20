"""
Node unavailable error.
"""

# --- IMPORTS ---
from src.err.ecoa_error import EcoaError

import logging


# --- ERROR ---
class NodeUnavailableError(EcoaError):
    """
    Raised when no node in the list answers a publish.
    """

    MESSAGE = 'Service temporarily unavailable!'
    STATUS_CODE = 503
    LOG_LEVEL = logging.ERROR
