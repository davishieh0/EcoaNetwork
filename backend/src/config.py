"""
Application configuration.
"""

# --- IMPORTS ---
from pydantic_settings import BaseSettings
from pydantic_settings import SettingsConfigDict


# --- CONFIG ---
class Config(BaseSettings):
    """
    Reads configuration from environment variables and .env file.
    """

    # general
    ENVIRONMENT: str = 'development'
    LOG_LEVEL: str = 'INFO'

    # routing
    API_PREFIX: str = '/api'

    # cors
    CORS_ORIGINS: str = 'http://localhost:*'

    # aws (the endpoint is only set when pointing at LocalStack)
    AWS_REGION: str
    AWS_ENDPOINT_URL: str = ''

    @property
    def cors_origins(self) -> list[str]:
        """
        Parses CORS_ORIGINS into a list.

        :return: List of allowed CORS origins.
        """
        return [o.strip() for o in self.CORS_ORIGINS.split(',') if o.strip()]


    @property
    def is_production(self) -> bool:
        """
        Whether this instance runs in production.

        :return: True when ENVIRONMENT is 'production'.
        """
        return self.ENVIRONMENT.strip().lower() == 'production'

    # pydantic settings
    model_config = SettingsConfigDict(env_file='.env', extra='ignore')


# shared singleton: import `config` instead of instantiating Config again
config = Config()
