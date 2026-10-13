"""
Data integrity error.
"""

# --- IMPORTS ---
from src.err.ecoa_error import EcoaError

import logging


# --- ERROR ---
class DataIntegrityError(EcoaError):
    """
    Raised when a write breaks a rule of the database, e.g. a unique value
    that is already taken.
    """

    MESSAGE = 'Conflicting data!'
    STATUS_CODE = 409
    LOG_LEVEL = logging.WARNING
