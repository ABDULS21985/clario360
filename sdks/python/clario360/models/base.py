from __future__ import annotations

from typing import TYPE_CHECKING, Any, Dict, Type, TypeVar

from pydantic import BaseModel as PydanticBaseModel
from pydantic import ConfigDict

from clario360.utils.dataframe import models_to_dataframe

if TYPE_CHECKING:
    from pandas import DataFrame

TModel = TypeVar("TModel", bound="BaseModel")


class BaseModel(PydanticBaseModel):
    """Base model for all Clario 360 SDK models."""

    model_config = ConfigDict(extra="allow", populate_by_name=True)

    @classmethod
    def from_dict(cls: Type[TModel], data: Dict[str, Any]) -> TModel:
        return cls.model_validate(data)

    def to_dict(self) -> Dict[str, Any]:
        return self.model_dump(exclude_none=True)

    def to_dataframe(self) -> "DataFrame":
        return models_to_dataframe([self])
