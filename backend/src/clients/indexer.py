"""
Indexer connection.
"""

# --- IMPORTS ---
from src.err.indexer_unavailable_error import IndexerUnavailableError
from src.schemas.reads import IndexerAnswer
from urllib.parse import quote

import httpx


# --- GLOBALS ---
CALL_TIMEOUT = 5


# --- CODE ---
class IndexerClient:
    """
    Reads from the indexer's HTTP API, one method per route, and hands back
    what it answered. The only part of the backend that knows the indexer's
    routes.
    """

    def __init__(
        self,
        url: str,
        transport: httpx.AsyncBaseTransport | None = None,
    ) -> None:
        """
        Initialize the indexer connection.

        Nothing connects here: the pool opens on the first call.

        :param url: The base URL of the indexer's read API.
        :param transport: Stands in for the network, in tests.

        :return: None.
        """
        self._client = httpx.AsyncClient(
            base_url=url, timeout=CALL_TIMEOUT, transport=transport
        )


    @staticmethod
    def _escape(segment: str) -> str:
        """
        Escapes one path segment, keeping the colons of entity ids and
        did:key.

        :param segment: The segment, as the client sent it.

        :return: The segment, safe to put in a path.
        """
        if segment == '.':
            return '%2E'
        if segment == '..':
            return '%2E%2E'
        return quote(segment, safe=':@')


    async def _get(
        self,
        *segments: str,
        params: dict[str, str | None] | None = None,
    ) -> IndexerAnswer:
        """
        Reads one route of the indexer, the one place that builds its paths.

        :param segments: The route's segments, such as 'entities', an entity
            id and 'reviews'.
        :param params: The query parameters, None for those not given.

        :raises IndexerUnavailableError: If the indexer does not answer in
            time, or answers with a 5xx.

        :return: The indexer's status and body.
        """
        # build the path and the query parameters
        path = '/' + '/'.join(self._escape(segment) for segment in segments)
        given = {k: v for k, v in (params or {}).items() if v is not None}

        try:
            response = await self._client.get(path, params=given)

        # down, unreachable or too slow
        except httpx.HTTPError as error:
            raise IndexerUnavailableError(repr(error)) from error

        # its database down, or a fault of its own: details stay in the log
        if response.status_code >= 500:
            raise IndexerUnavailableError(response.status_code, response.text)

        return IndexerAnswer(status=response.status_code, body=response.content)


    async def get_entity(
        self,
        entity: str,
        moderators: str | None = None,
    ) -> IndexerAnswer:
        """
        An entity and the value of each of its fields.

        :param entity: The entity id.
        :param moderators: The moderators the client follows, comma-separated
            did:key; None for the indexer's own.

        :raises IndexerUnavailableError: If the indexer does not answer.

        :return: The indexer's status and body.
        """
        return await self._get('entities', entity,
                               params={'moderators': moderators})


    async def get_event(
        self,
        event_id: str,
        moderators: str | None = None,
    ) -> IndexerAnswer:
        """
        A signed event alone, for inspection.

        :param event_id: The eventId.
        :param moderators: The moderators the client follows, comma-separated
            did:key; None for the indexer's own.

        :raises IndexerUnavailableError: If the indexer does not answer.

        :return: The indexer's status and body.
        """
        return await self._get('events', event_id,
                               params={'moderators': moderators})


    async def get_identity(self, did: str) -> IndexerAnswer:
        """
        The identity a key belongs to.

        :param did: The did:key.

        :raises IndexerUnavailableError: If the indexer does not answer.

        :return: The indexer's status and body.
        """
        return await self._get('identities', did)


    async def get_profile(
        self,
        did: str,
        moderators: str | None = None,
    ) -> IndexerAnswer:
        """
        An identity's current profile, through any of its keys.

        :param did: A did:key of the identity.
        :param moderators: The moderators the client follows, comma-separated
            did:key; None for the indexer's own.

        :raises IndexerUnavailableError: If the indexer does not answer.

        :return: The indexer's status and body.
        """
        return await self._get('profiles', did,
                               params={'moderators': moderators})


    async def get_review(
        self,
        event_id: str,
        moderators: str | None = None,
    ) -> IndexerAnswer:
        """
        One version of a review, with every version of its chain.

        :param event_id: The eventId of any version.
        :param moderators: The moderators the client follows, comma-separated
            did:key; None for the indexer's own.

        :raises IndexerUnavailableError: If the indexer does not answer.

        :return: The indexer's status and body.
        """
        return await self._get('reviews', event_id,
                               params={'moderators': moderators})


    async def list_entity_reviews(
        self,
        entity: str,
        limit: str | None,
        after: str | None,
        labeled: str | None,
        moderators: str | None = None,
    ) -> IndexerAnswer:
        """
        The current version of each of an entity's reviews, newest first.

        :param entity: The entity id.
        :param limit: The page size, as the client sent it.
        :param after: The next of a previous page.
        :param labeled: 'show', to see what moderation hides, marked.
        :param moderators: The moderators the client follows, comma-separated
            did:key; None for the indexer's own.

        :raises IndexerUnavailableError: If the indexer does not answer.

        :return: The indexer's status and body.
        """
        return await self._get('entities', entity, 'reviews',
                               params={'limit': limit, 'after': after,
                                       'labeled': labeled,
                                       'moderators': moderators})


    async def list_profiles(
        self,
        ids: str | None,
        moderators: str | None = None,
    ) -> IndexerAnswer:
        """
        The current profiles of some identities, in the order asked.

        :param ids: 1 to 100 did:key, comma-separated.
        :param moderators: The moderators the client follows, comma-separated
            did:key; None for the indexer's own.

        :raises IndexerUnavailableError: If the indexer does not answer.

        :return: The indexer's status and body.
        """
        return await self._get('profiles', params={'ids': ids,
                                                   'moderators': moderators})


    async def list_review_responses(
        self,
        event_id: str,
        limit: str | None,
        after: str | None,
        labeled: str | None,
        moderators: str | None = None,
    ) -> IndexerAnswer:
        """
        The conversation on a review, in order.

        :param event_id: The eventId of any version of the review.
        :param limit: The page size, as the client sent it.
        :param after: The next of a previous page.
        :param labeled: 'show', to see what moderation hides, marked.
        :param moderators: The moderators the client follows, comma-separated
            did:key; None for the indexer's own.

        :raises IndexerUnavailableError: If the indexer does not answer.

        :return: The indexer's status and body.
        """
        return await self._get('reviews', event_id, 'responses',
                               params={'limit': limit, 'after': after,
                                       'labeled': labeled,
                                       'moderators': moderators})


    async def search_entities(
        self,
        q: str | None,
        limit: str | None,
        after: str | None,
        category: str | None = None,
        moderators: str | None = None,
    ) -> IndexerAnswer:
        """
        Entities with a name that holds q, newest first; every entity
        without q.

        :param q: Part of a name; None for every entity.
        :param limit: The page size, as the client sent it.
        :param after: The next of a previous page.
        :param category: One of the protocol's categories, to keep only
            the entities it holds for.
        :param moderators: The moderators the client follows, comma-separated
            did:key; None for the indexer's own.

        :raises IndexerUnavailableError: If the indexer does not answer.

        :return: The indexer's status and body.
        """
        return await self._get('entities',
                               params={'q': q, 'limit': limit, 'after': after,
                                       'category': category,
                                       'moderators': moderators})


    async def get_nodes(self) -> IndexerAnswer:
        """
        The nodes this indexer follows and their observed log positions.

        :raises IndexerUnavailableError: If the indexer does not answer.

        :return: The indexer's status and body.
        """
        return await self._get('nodes')


    async def close(self) -> None:
        """
        Closes the connection pool.

        :return: None.
        """
        await self._client.aclose()
