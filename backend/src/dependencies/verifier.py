"""
Verifier dependencies.
"""

# --- IMPORTS ---
from fastapi import Request
from src.services.verifier import HumanVerifier


# --- CODE ---
def get_verifier(request: Request) -> HumanVerifier:
    """
    Provides the human verifier.

    :param request: The current request.

    :return: The verifier opened at startup.
    """
    verifier: HumanVerifier = request.app.state.verifier
    return verifier
