"""
Read schemas.

IndexerAnswer is what the indexer connection hands back. The rest describe
what the indexer answers, for the documentation only: the read routes pass
its body on as bytes and never parse it, so no envelope in it is rewritten on
the way.
"""

# --- IMPORTS ---
from pydantic import BaseModel
from pydantic import Field
from src.err.indexer_unavailable_error import IndexerUnavailableError
from src.schemas.errors import ErrorResponse
from src.schemas.errors import error_responses
from typing import Any


# --- GLOBALS ---
# what each failure the indexer explains means, by status
PASSED_ON = {
    400: 'A malformed id or parameter: `invalid_event_id_error`, '
         '`invalid_entity_id_error`, `invalid_identity_error`, '
         '`invalid_ids_error`, `invalid_limit_error`, `invalid_cursor_error`, '
         '`invalid_labeled_error`, `invalid_query_error`.',
    404: 'Nothing with that id: `event_not_found_error`, '
         '`review_not_found_error`, `entity_not_found_error`, '
         '`profile_not_found_error`.',
    410: '`review_withdrawn_error` — its author withdrew it.',
    451: '`removed_by_moderation_error` — a moderator the indexer follows '
         'removed it.',
}


# --- CODE ---
class IndexerAnswer(BaseModel):
    """
    What the indexer answered a read, untouched.
    """

    status: int = Field(
        description='The status the indexer answered: 200, or one of the '
                    'failures it explains itself (400, 404, 410, 451).',
    )
    body: bytes = Field(
        description='The body, byte for byte: never parsed, so no envelope '
                    'in it is rewritten.',
    )


class Item(BaseModel):
    """
    One signed event, with what the indexer worked out about it.
    """

    envelope: str = Field(
        description='The event as signed, as a string: parse it to read it, '
                    'and check its signature over these exact characters.',
    )
    derived: dict[str, Any] = Field(
        description='What any honest indexer must agree on, from the events '
                    'alone.',
    )
    observed: dict[str, Any] = Field(
        description='What depends on when and where this indexer looked: '
                    'receivedAt, the labels that count.',
    )


class ItemPage(BaseModel):
    """
    One page of a list of signed events.
    """

    items: list[Item]
    next: str | None = Field(
        description='The after of the next page; null on the last one.',
    )


class ReviewDetail(BaseModel):
    """
    One version of a review, with every version of its chain.
    """

    review: Item
    versions: list[Item] = Field(description='Oldest first, forks included.')


class Entity(BaseModel):
    """
    One entity, and the value of each of its fields.
    """

    entity: str
    derived: dict[str, Any]
    observed: dict[str, Any] = Field(
        description='name, category, description: each the value that '
                    'holds and the proposal that set it, or null.',
    )


class EntityPage(BaseModel):
    """
    One page of entities.
    """

    items: list[Entity]
    next: str | None


class Profiles(BaseModel):
    """
    The current profiles of the identities asked for.
    """

    items: list[Item]


class Identity(BaseModel):
    """
    The identity a key belongs to.
    """

    identity: str = Field(description='The first key of its chain.')
    keys: list[str] = Field(description='In the order of its chain.')
    current: str = Field(description='The key it signs with now.')
    contested: bool = Field(
        description='Two successors at one of its keys: its new events '
                    'deserve suspicion.',
    )
    revoked: list[dict[str, Any]]


def read_responses(*statuses: int) -> dict[int | str, dict[str, Any]]:
    """
    Builds the OpenAPI entries of a read route's failures: those the indexer
    explains, passed on as it answered them, and the indexer not answering.

    :param statuses: The statuses the indexer may answer this route with.

    :return: Mapping of status code to its OpenAPI response entry.
    """
    passed_on: dict[int | str, dict[str, Any]] = {
        status: {'model': ErrorResponse, 'description': PASSED_ON[status]}
        for status in statuses
    }

    return {**passed_on, **error_responses(IndexerUnavailableError)}
