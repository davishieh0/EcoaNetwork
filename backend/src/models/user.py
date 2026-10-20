"""
User model.
"""

# --- IMPORTS ---
from datetime import datetime
from sqlalchemy import CheckConstraint
from sqlalchemy import DateTime
from sqlalchemy import Text
from sqlalchemy import func
from sqlalchemy import text
from sqlalchemy.orm import Mapped
from sqlalchemy.orm import mapped_column
from src.models.base import Base
from uuid import UUID


# --- GLOBALS ---
# 3 to 30 characters: lowercase letters, digits and '_', starting with a letter
HANDLE_PATTERN = '^[a-z][a-z0-9_]{2,29}$'


# --- CODE ---
class User(Base):
    """
    An account on the platform.
    """

    __tablename__ = 'users'
    __table_args__ = (
        CheckConstraint(f"handle ~ '{HANDLE_PATTERN}'", name='handle_format'),
    )

    id: Mapped[UUID] = mapped_column(
        primary_key=True, server_default=text('uuidv7()'), init=False
    )
    sub: Mapped[str] = mapped_column(Text, unique=True)
    handle: Mapped[str] = mapped_column(Text, unique=True)
    created_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), server_default=func.now(), init=False
    )
