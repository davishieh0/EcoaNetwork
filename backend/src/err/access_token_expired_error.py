"""
Access token expired error.
"""

# --- IMPORTS ---
from src.err.ecoa_error import EcoaError

import logging


# --- ERROR ---
class AccessTokenExpiredError(EcoaError):
    """
    Raised when the access token is genuine but expired: refreshing the session
    gets a new one.
    """

    MESSAGE = 'Access token expired!'
    STATUS_CODE = 401
    LOG_LEVEL = logging.INFO
