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
