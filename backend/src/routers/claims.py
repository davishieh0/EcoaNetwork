"""
Claim endpoints: the human claim this gateway issues as a verifier.
"""

# --- IMPORTS ---
from fastapi import APIRouter
from fastapi import Depends
from src.dependencies.verifier import get_verifier
from src.err.invalid_captcha_error import InvalidCaptchaError
from src.err.node_unavailable_error import NodeUnavailableError
from src.err.verifier_disabled_error import VerifierDisabledError
from src.schemas.claims import HumanClaimRequest
from src.schemas.claims import HumanClaimResult
from src.schemas.errors import error_responses
from src.services.verifier import HumanVerifier
from typing import Any


# --- GLOBALS ---
router = APIRouter()


# --- CODE ---
@router.get('/human/challenge',
            summary='Get a check to solve for a human claim',
            responses=error_responses(VerifierDisabledError))
async def human_challenge(
    verifier: HumanVerifier = Depends(get_verifier),
) -> dict[str, Any]:
    """
    A proof of work for the browser to solve, signed by this gateway.
    """
    return verifier.challenge()


@router.post('/human',
             summary='Issue a human claim',
             response_model=HumanClaimResult,
             response_model_by_alias=True,
             responses=error_responses(InvalidCaptchaError,
                                       VerifierDisabledError,
                                       NodeUnavailableError))
async def issue_human(
    body: HumanClaimRequest,
    verifier: HumanVerifier = Depends(get_verifier),
) -> HumanClaimResult:
    """
    Checks the solved proof of work, signs a human claim about the subject
    with this gateway's verifier key, and hands it to the network.
    """
    return await verifier.issue(body.subject, body.altcha)
