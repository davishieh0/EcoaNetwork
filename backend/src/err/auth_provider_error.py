"""
Auth provider error.
"""

# --- IMPORTS ---
from src.err.ecoa_error import EcoaError

import logging


# --- ERROR ---
class AuthProviderError(EcoaError):
    """
    Raised when the identity provider fails or answers unexpectedly.
    """

    MESSAGE = 'Authentication service unavailable!'
    STATUS_CODE = 502
    LOG_LEVEL = logging.ERROR
