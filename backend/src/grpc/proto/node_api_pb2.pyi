from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Outcome(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    OUTCOME_UNSPECIFIED: _ClassVar[Outcome]
    ACCEPTED: _ClassVar[Outcome]
    ALREADY_EXISTS: _ClassVar[Outcome]
    ORPHAN: _ClassVar[Outcome]
    REJECTED: _ClassVar[Outcome]
OUTCOME_UNSPECIFIED: Outcome
ACCEPTED: Outcome
ALREADY_EXISTS: Outcome
ORPHAN: Outcome
REJECTED: Outcome

class Event(_message.Message):
    __slots__ = ("envelope",)
    ENVELOPE_FIELD_NUMBER: _ClassVar[int]
    envelope: bytes
    def __init__(self, envelope: _Optional[bytes] = ...) -> None: ...

class PublishResponse(_message.Message):
    __slots__ = ("outcome", "reason")
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    outcome: Outcome
    reason: str
    def __init__(self, outcome: _Optional[_Union[Outcome, str]] = ..., reason: _Optional[str] = ...) -> None: ...

class Cursor(_message.Message):
    __slots__ = ("after", "log_id")
    AFTER_FIELD_NUMBER: _ClassVar[int]
    LOG_ID_FIELD_NUMBER: _ClassVar[int]
    after: int
    log_id: str
    def __init__(self, after: _Optional[int] = ..., log_id: _Optional[str] = ...) -> None: ...

class StoredEvent(_message.Message):
    __slots__ = ("envelope", "sequence", "received_at", "log_id", "event_id", "event_type", "head")
    ENVELOPE_FIELD_NUMBER: _ClassVar[int]
    SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    RECEIVED_AT_FIELD_NUMBER: _ClassVar[int]
    LOG_ID_FIELD_NUMBER: _ClassVar[int]
    EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    EVENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    HEAD_FIELD_NUMBER: _ClassVar[int]
    envelope: bytes
    sequence: int
    received_at: int
    log_id: str
    event_id: str
    event_type: str
    head: int
    def __init__(self, envelope: _Optional[bytes] = ..., sequence: _Optional[int] = ..., received_at: _Optional[int] = ..., log_id: _Optional[str] = ..., event_id: _Optional[str] = ..., event_type: _Optional[str] = ..., head: _Optional[int] = ...) -> None: ...
