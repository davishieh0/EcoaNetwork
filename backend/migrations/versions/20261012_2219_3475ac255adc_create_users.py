"""create users

Revision ID: 3475ac255adc
Revises:
Create Date: 2026-10-12 22:19:49.816050

"""

from alembic import op
from collections.abc import Sequence

import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision: str = '3475ac255adc'
down_revision: str | Sequence[str] | None = None
branch_labels: str | Sequence[str] | None = None
depends_on: str | Sequence[str] | None = None


def upgrade() -> None:
    """Upgrade schema."""
    op.create_table(
        'users',
        sa.Column(
            'id', sa.Uuid(), server_default=sa.text('uuidv7()'), nullable=False
        ),
        sa.Column('sub', sa.Text(), nullable=False),
        sa.Column('handle', sa.Text(), nullable=False),
        sa.Column(
            'created_at',
            sa.DateTime(timezone=True),
            server_default=sa.text('now()'),
            nullable=False,
        ),
        sa.CheckConstraint(
            "handle ~ '^[a-z][a-z0-9_]{2,29}$'",
            name=op.f('ck_users_handle_format'),
        ),
        sa.PrimaryKeyConstraint('id', name=op.f('pk_users')),
        sa.UniqueConstraint('handle', name=op.f('uq_users_handle')),
        sa.UniqueConstraint('sub', name=op.f('uq_users_sub')),
    )


def downgrade() -> None:
    """Downgrade schema."""
    op.drop_table('users')
