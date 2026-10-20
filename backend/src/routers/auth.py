"""
Authentication endpoints.
"""

# --- IMPORTS ---
from fastapi import APIRouter
from fastapi import Depends
from fastapi import Request
from fastapi import Response
from fastapi.responses import JSONResponse
from src.dependencies.auth import get_auth
from src.dependencies.auth import get_current_account
from src.err.access_token_expired_error import AccessTokenExpiredError
from src.err.auth_provider_error import AuthProviderError
from src.err.database_unavailable_error import DatabaseUnavailableError
from src.err.email_already_registered_error import EmailAlreadyRegisteredError
from src.err.email_not_verified_error import EmailNotVerifiedError
from src.err.expired_code_error import ExpiredCodeError
from src.err.handle_taken_error import HandleTakenError
from src.err.invalid_code_error import InvalidCodeError
from src.err.invalid_credentials_error import InvalidCredentialsError
from src.err.invalid_password_error import InvalidPasswordError
from src.err.invalid_session_error import InvalidSessionError
from src.err.not_authenticated_error import NotAuthenticatedError
from src.err.too_many_requests_error import TooManyRequestsError
from src.schemas.auth import AccessClaims
from src.schemas.auth import Account
from src.schemas.auth import EmailRequest
from src.schemas.auth import LoginRequest
from src.schemas.auth import RegisterRequest
from src.schemas.auth import ResetPasswordRequest
from src.schemas.auth import VerifyRequest
from src.schemas.errors import VALIDATION_RESPONSE
from src.schemas.errors import error_responses
from src.services.auth import AuthService
from src.utils.cookies import ACCESS_TOKEN_COOKIE
from src.utils.cookies import REFRESH_TOKEN_COOKIE
from src.utils.cookies import clear_session_cookies
from src.utils.cookies import set_session_cookies


# --- GLOBALS ---
router = APIRouter()

# every route may be throttled, or find the provider down
COMMON = (TooManyRequestsError, AuthProviderError)


# --- ACCOUNT ---
@router.post('/register',
             summary='Create an account',
             status_code=201,
             response_model=Account,
             responses={
                 **VALIDATION_RESPONSE,
                 **error_responses(
                     HandleTakenError,
                     EmailAlreadyRegisteredError,
                     InvalidPasswordError,
                     DatabaseUnavailableError,
                     *COMMON,
                 ),
             })
async def register(
    body: RegisterRequest,
    auth: AuthService = Depends(get_auth),
) -> JSONResponse:
    """
    Creates the account and emails its confirmation code.
    """
    account = await auth.register(body.handle, body.email, body.password)

    return JSONResponse(account.model_dump(mode='json'), status_code=201)


@router.post('/verify',
             summary='Confirm an email',
             status_code=204,
             responses={
                 **VALIDATION_RESPONSE,
                 **error_responses(InvalidCodeError, ExpiredCodeError,
                                   *COMMON),
             })
async def verify(
    body: VerifyRequest,
    auth: AuthService = Depends(get_auth),
) -> Response:
    """
    Confirms the account's email with the code it was sent.
    """
    await auth.verify(body.email, body.code)

    return Response(status_code=204)


@router.post('/verify/resend',
             summary='Send the confirmation code again',
             status_code=204,
             responses={**VALIDATION_RESPONSE, **error_responses(*COMMON)})
async def resend_verification(
    body: EmailRequest,
    auth: AuthService = Depends(get_auth),
) -> Response:
    """
    Emails a new confirmation code. The previous one stops working.
    """
    await auth.resend_verification(body.email)

    return Response(status_code=204)


@router.post('/login',
             summary='Sign in',
             response_model=Account,
             responses={
                 **VALIDATION_RESPONSE,
                 **error_responses(InvalidCredentialsError,
                                   EmailNotVerifiedError,
                                   DatabaseUnavailableError, *COMMON),
             })
async def login(
    body: LoginRequest,
    auth: AuthService = Depends(get_auth),
) -> JSONResponse:
    """
    Signs the account in, stores the session in cookies and returns the
    account.
    """
    session = await auth.login(body.email, body.password)

    response = JSONResponse(session.account.model_dump(mode='json'))
    set_session_cookies(response, session.tokens)

    return response


@router.post('/refresh',
             summary='Renew the session',
             status_code=204,
             responses=error_responses(InvalidSessionError,
                                       NotAuthenticatedError, *COMMON))
async def refresh(
    request: Request,
    auth: AuthService = Depends(get_auth),
) -> Response:
    """
    Trades the refresh token for a new access token.
    """
    tokens = await auth.refresh(
        request.cookies.get(ACCESS_TOKEN_COOKIE),
        request.cookies.get(REFRESH_TOKEN_COOKIE),
    )

    response = Response(status_code=204)
    set_session_cookies(response, tokens)

    return response


@router.post('/logout',
             summary='Sign out',
             status_code=204,
             responses=error_responses(*COMMON))
async def logout(
    request: Request,
    auth: AuthService = Depends(get_auth),
) -> Response:
    """
    Ends the session: revokes it with the provider and removes the cookies.
    """
    await auth.logout(request.cookies.get(REFRESH_TOKEN_COOKIE))

    response = Response(status_code=204)
    clear_session_cookies(response)

    return response


# --- PASSWORD ---
@router.post('/forgot-password',
             summary='Ask for a password reset code',
             status_code=204,
             responses={**VALIDATION_RESPONSE, **error_responses(*COMMON)})
async def forgot_password(
    body: EmailRequest,
    auth: AuthService = Depends(get_auth),
) -> Response:
    """
    Emails a code that lets the account choose a new password.
    """
    await auth.forgot_password(body.email)

    return Response(status_code=204)


@router.post('/reset-password',
             summary='Set a new password',
             status_code=204,
             responses={
                 **VALIDATION_RESPONSE,
                 **error_responses(InvalidCodeError, ExpiredCodeError,
                                   InvalidPasswordError, *COMMON),
             })
async def reset_password(
    body: ResetPasswordRequest,
    auth: AuthService = Depends(get_auth),
) -> Response:
    """
    Sets the new password with the code the account was sent.
    """
    await auth.reset_password(body.email, body.code, body.password)

    return Response(status_code=204)


@router.get('/me',
            summary='The signed-in account',
            response_model=Account,
            responses=error_responses(NotAuthenticatedError,
                                      AccessTokenExpiredError,
                                      DatabaseUnavailableError, *COMMON))
async def me(
    request: Request,
    account: AccessClaims = Depends(get_current_account),
    auth: AuthService = Depends(get_auth),
) -> JSONResponse:
    """
    Returns the signed-in account. The email comes from the identity
    provider, which also refuses a session that was signed out.
    """
    found = await auth.me(account.sub, request.cookies[ACCESS_TOKEN_COOKIE])

    return JSONResponse(found.model_dump(mode='json'))
