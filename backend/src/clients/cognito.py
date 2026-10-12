"""
Cognito user pool client.
"""

# --- IMPORTS ---
from botocore.exceptions import BotoCoreError
from botocore.exceptions import ClientError
from src.err.auth_provider_error import AuthProviderError
from src.err.ecoa_error import EcoaError
from src.err.email_already_registered_error import EmailAlreadyRegisteredError
from src.err.email_not_verified_error import EmailNotVerifiedError
from src.err.expired_code_error import ExpiredCodeError
from src.err.invalid_code_error import InvalidCodeError
from src.err.invalid_credentials_error import InvalidCredentialsError
from src.err.invalid_password_error import InvalidPasswordError
from src.err.invalid_request_error import InvalidRequestError
from src.err.invalid_session_error import InvalidSessionError
from src.err.too_many_requests_error import TooManyRequestsError
from src.schemas.auth import Tokens
from typing import Any

import base64
import hashlib
import hmac


# --- GLOBALS ---
# Cognito error code -> the error to raise, or None to treat it as success
ErrorMap = dict[str, type[EcoaError] | None]

# what a code means wherever it shows up; operations override what differs
ERRORS: ErrorMap = {
    'CodeMismatchException': InvalidCodeError,
    'ExpiredCodeException': ExpiredCodeError,
    'InvalidParameterException': InvalidRequestError,
    'InvalidPasswordException': InvalidPasswordError,
    'LimitExceededException': TooManyRequestsError,
    'TooManyFailedAttemptsException': TooManyRequestsError,
    'TooManyRequestsException': TooManyRequestsError,
    'UserNotConfirmedException': EmailNotVerifiedError,
    'UsernameExistsException': EmailAlreadyRegisteredError,
}


# --- CODE ---
class CognitoClient:
    """
    The user pool operations the backend needs, behind plain methods.
    """

    def __init__(
        self,
        client: Any,
        client_id: str,
        client_secret: str,
    ) -> None:
        """
        Initialize the Cognito client.

        :param client: An open 'cognito-idp' service client.
        :param client_id: The app client the backend signs in through.
        :param client_secret: That app client's secret.

        :return: None.
        """
        self._client = client
        self._client_id = client_id
        self._client_secret = client_secret


    def _secret_hash(self, username: str) -> str:
        """
        Proves the call comes from whoever holds the client secret.

        :param username: The username the operation is about.

        :return: The SECRET_HASH Cognito expects for that username.
        """
        digest = hmac.new(
            self._client_secret.encode(),
            (username + self._client_id).encode(),
            hashlib.sha256,
        ).digest()

        return base64.b64encode(digest).decode()


    @staticmethod
    def _translate(
        operation: str,
        error: ClientError,
        errors: ErrorMap,
    ) -> EcoaError | None:
        """
        Turns a Cognito failure into the error the backend answers with.

        :param operation: The operation that failed.
        :param error: What Cognito raised.
        :param errors: The operation's own meaning for some codes.

        :return: The error to raise, or None when the failure counts as
                 success.
        """
        code = error.response['Error']['Code']
        message = error.response['Error'].get('Message', '')
        details = f'{operation} failed with {code}: {message}'

        # a rejected client secret is our misconfiguration: not the caller's
        if 'secret' in message.lower():
            return AuthProviderError(details)

        # Cognito reports a locked-out sign-in as bad credentials
        if 'attempts exceeded' in message.lower():
            return TooManyRequestsError(details)

        mapping = {**ERRORS, **errors}

        # a code nobody planned for is the provider misbehaving
        if code not in mapping:
            return AuthProviderError(details)

        translated = mapping[code]

        return translated(details) if translated else None


    async def _call(
        self,
        operation: str,
        errors: ErrorMap | None = None,
        **params: Any,
    ) -> dict[str, Any]:
        """
        Runs one Cognito operation and translates its failures.

        :param operation: The operation, as the service client names it.
        :param errors: The operation's own meaning for some codes.
        :param params: The operation's parameters.

        :return: Cognito's response, empty when a failure counts as success.
        """
        try:
            response: dict[str, Any] = await getattr(
                self._client, operation
            )(**params)

        # Cognito answered with an error
        except ClientError as error:
            translated = self._translate(operation, error, errors or {})

            if translated is None:
                return {}

            raise translated from error

        # Cognito could not be reached at all
        except BotoCoreError as error:
            raise AuthProviderError(f'{operation} failed: {error}') from error

        return response


    def _tokens(self, operation: str, response: dict[str, Any]) -> Tokens:
        """
        Reads the tokens out of an authentication response.

        :param operation: The operation that produced the response.
        :param response: Cognito's response.

        :return: The tokens.
        """
        result = response.get('AuthenticationResult')

        # no tokens means Cognito wants another step, e.g. an MFA code
        if not result:
            challenge = response.get('ChallengeName')
            raise AuthProviderError(
                f'{operation} asked for an unsupported challenge: {challenge}'
            )

        return Tokens(
            access_token=result['AccessToken'],
            expires_in=result['ExpiresIn'],
            refresh_token=result.get('RefreshToken'),
        )


    async def sign_up(self, email: str, password: str) -> str:
        """
        Creates an account and emails its confirmation code.

        :param email: The address the account signs in with.
        :param password: The account's password.

        :return: The account's permanent id (its 'sub').
        """
        response = await self._call(
            'sign_up',
            ClientId=self._client_id,
            SecretHash=self._secret_hash(email),
            Username=email,
            Password=password,
            UserAttributes=[{'Name': 'email', 'Value': email}],
        )

        return str(response['UserSub'])


    async def confirm_sign_up(self, email: str, code: str) -> None:
        """
        Confirms an account with the code it was emailed.

        :param email: The account's address.
        :param code: The confirmation code.

        :return: None.
        """
        await self._call(
            'confirm_sign_up',
            errors={
                'NotAuthorizedException': InvalidCodeError,
                'UserNotFoundException': InvalidCodeError,
            },
            ClientId=self._client_id,
            SecretHash=self._secret_hash(email),
            Username=email,
            ConfirmationCode=code,
        )


    async def resend_confirmation_code(self, email: str) -> None:
        """
        Emails a new confirmation code, invalidating the previous one.

        Succeeds whether or not there is an account waiting on a code.

        :param email: The account's address.

        :return: None.
        """
        await self._call(
            'resend_confirmation_code',
            errors={
                'InvalidParameterException': None,
                'UserNotFoundException': None,
            },
            ClientId=self._client_id,
            SecretHash=self._secret_hash(email),
            Username=email,
        )


    async def sign_in(self, email: str, password: str) -> Tokens:
        """
        Signs an account in with its email and password.

        :param email: The account's address.
        :param password: The account's password.

        :return: The session's tokens.
        """
        response = await self._call(
            'initiate_auth',
            errors={
                'NotAuthorizedException': InvalidCredentialsError,
                'UserNotFoundException': InvalidCredentialsError,
            },
            ClientId=self._client_id,
            AuthFlow='USER_PASSWORD_AUTH',
            AuthParameters={
                'USERNAME': email,
                'PASSWORD': password,
                'SECRET_HASH': self._secret_hash(email),
            },
        )

        return self._tokens('sign_in', response)


    async def refresh(self, refresh_token: str, sub: str) -> Tokens:
        """
        Trades a refresh token for a new access token.

        :param refresh_token: The session's refresh token.
        :param sub: The account's permanent id. Cognito hashes the secret
                    against it here, not against the email.

        :return: The new tokens, without a refresh token.
        """
        response = await self._call(
            'initiate_auth',
            errors={
                'NotAuthorizedException': InvalidSessionError,
                'UserNotFoundException': InvalidSessionError,
            },
            ClientId=self._client_id,
            AuthFlow='REFRESH_TOKEN_AUTH',
            AuthParameters={
                'REFRESH_TOKEN': refresh_token,
                'SECRET_HASH': self._secret_hash(sub),
            },
        )

        return self._tokens('refresh', response)


    async def sign_out(self, refresh_token: str) -> None:
        """
        Revokes a refresh token, and every access token issued from it.

        Succeeds when the token is already revoked or was never valid.

        :param refresh_token: The session's refresh token.

        :return: None.
        """
        await self._call(
            'revoke_token',
            errors={
                'InvalidParameterException': None,
                'UnauthorizedException': None,
            },
            Token=refresh_token,
            ClientId=self._client_id,
            ClientSecret=self._client_secret,
        )


    async def forgot_password(self, email: str) -> None:
        """
        Emails a code that lets the account choose a new password.

        Succeeds whether or not an account uses the address.

        :param email: The account's address.

        :return: None.
        """
        await self._call(
            'forgot_password',
            errors={
                'InvalidParameterException': None,
                'UserNotFoundException': None,
            },
            ClientId=self._client_id,
            SecretHash=self._secret_hash(email),
            Username=email,
        )
