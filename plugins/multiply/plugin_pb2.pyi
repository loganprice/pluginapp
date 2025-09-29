from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Iterable as _Iterable, Mapping as _Mapping, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class InfoRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class PluginInfo(_message.Message):
    __slots__ = ("name", "version", "description", "parameter_specs", "auth")
    class ParameterSpecsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: ParamSpec
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[ParamSpec, _Mapping]] = ...) -> None: ...
    NAME_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    PARAMETER_SPECS_FIELD_NUMBER: _ClassVar[int]
    AUTH_FIELD_NUMBER: _ClassVar[int]
    name: str
    version: str
    description: str
    parameter_specs: _containers.MessageMap[str, ParamSpec]
    auth: Authorization
    def __init__(self, name: _Optional[str] = ..., version: _Optional[str] = ..., description: _Optional[str] = ..., parameter_specs: _Optional[_Mapping[str, ParamSpec]] = ..., auth: _Optional[_Union[Authorization, _Mapping]] = ...) -> None: ...

class ParamSpec(_message.Message):
    __slots__ = ("name", "description", "required", "default_value", "type", "allowed_values")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_VALUE_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    ALLOWED_VALUES_FIELD_NUMBER: _ClassVar[int]
    name: str
    description: str
    required: bool
    default_value: str
    type: str
    allowed_values: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, name: _Optional[str] = ..., description: _Optional[str] = ..., required: bool = ..., default_value: _Optional[str] = ..., type: _Optional[str] = ..., allowed_values: _Optional[_Iterable[str]] = ...) -> None: ...

class ExecuteRequest(_message.Message):
    __slots__ = ("params",)
    class ParamsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    PARAMS_FIELD_NUMBER: _ClassVar[int]
    params: _containers.ScalarMap[str, str]
    def __init__(self, params: _Optional[_Mapping[str, str]] = ...) -> None: ...

class ExecuteOutput(_message.Message):
    __slots__ = ("output", "error", "progress")
    OUTPUT_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_FIELD_NUMBER: _ClassVar[int]
    output: str
    error: Error
    progress: Progress
    def __init__(self, output: _Optional[str] = ..., error: _Optional[_Union[Error, _Mapping]] = ..., progress: _Optional[_Union[Progress, _Mapping]] = ...) -> None: ...

class Error(_message.Message):
    __slots__ = ("message", "code", "details")
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    CODE_FIELD_NUMBER: _ClassVar[int]
    DETAILS_FIELD_NUMBER: _ClassVar[int]
    message: str
    code: str
    details: str
    def __init__(self, message: _Optional[str] = ..., code: _Optional[str] = ..., details: _Optional[str] = ...) -> None: ...

class Progress(_message.Message):
    __slots__ = ("percent_complete", "stage", "current_step", "total_steps")
    PERCENT_COMPLETE_FIELD_NUMBER: _ClassVar[int]
    STAGE_FIELD_NUMBER: _ClassVar[int]
    CURRENT_STEP_FIELD_NUMBER: _ClassVar[int]
    TOTAL_STEPS_FIELD_NUMBER: _ClassVar[int]
    percent_complete: float
    stage: str
    current_step: int
    total_steps: int
    def __init__(self, percent_complete: _Optional[float] = ..., stage: _Optional[str] = ..., current_step: _Optional[int] = ..., total_steps: _Optional[int] = ...) -> None: ...

class SummaryRequest(_message.Message):
    __slots__ = ("plugin_name", "start_time", "end_time", "success", "error", "metadata", "metrics")
    class MetadataEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    class MetricsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: float
        def __init__(self, key: _Optional[str] = ..., value: _Optional[float] = ...) -> None: ...
    PLUGIN_NAME_FIELD_NUMBER: _ClassVar[int]
    START_TIME_FIELD_NUMBER: _ClassVar[int]
    END_TIME_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    METRICS_FIELD_NUMBER: _ClassVar[int]
    plugin_name: str
    start_time: int
    end_time: int
    success: bool
    error: str
    metadata: _containers.ScalarMap[str, str]
    metrics: _containers.ScalarMap[str, float]
    def __init__(self, plugin_name: _Optional[str] = ..., start_time: _Optional[int] = ..., end_time: _Optional[int] = ..., success: bool = ..., error: _Optional[str] = ..., metadata: _Optional[_Mapping[str, str]] = ..., metrics: _Optional[_Mapping[str, float]] = ...) -> None: ...

class SummaryResponse(_message.Message):
    __slots__ = ("plugin_name", "start_time", "end_time", "duration", "success", "error", "metadata", "metrics")
    class MetadataEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    class MetricsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: float
        def __init__(self, key: _Optional[str] = ..., value: _Optional[float] = ...) -> None: ...
    PLUGIN_NAME_FIELD_NUMBER: _ClassVar[int]
    START_TIME_FIELD_NUMBER: _ClassVar[int]
    END_TIME_FIELD_NUMBER: _ClassVar[int]
    DURATION_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    METRICS_FIELD_NUMBER: _ClassVar[int]
    plugin_name: str
    start_time: int
    end_time: int
    duration: float
    success: bool
    error: str
    metadata: _containers.ScalarMap[str, str]
    metrics: _containers.ScalarMap[str, float]
    def __init__(self, plugin_name: _Optional[str] = ..., start_time: _Optional[int] = ..., end_time: _Optional[int] = ..., duration: _Optional[float] = ..., success: bool = ..., error: _Optional[str] = ..., metadata: _Optional[_Mapping[str, str]] = ..., metrics: _Optional[_Mapping[str, float]] = ...) -> None: ...

class Authorization(_message.Message):
    __slots__ = ("source", "values")
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    VALUES_FIELD_NUMBER: _ClassVar[int]
    source: str
    values: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, source: _Optional[str] = ..., values: _Optional[_Iterable[str]] = ...) -> None: ...
