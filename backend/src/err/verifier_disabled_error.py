"""
Verifier disabled error.
"""

# --- IMPORTS ---
from src.err.ecoa_error import EcoaError

import logging


# --- ERROR ---
class VerifierDisabledError(EcoaError):
    """
    Raised when this gateway has no verifier key, so it issues no claim.
    """

    MESSAGE = 'This gateway does not issue claims.'
    STATUS_CODE = 503
    LOG_LEVEL = logging.WARNING
