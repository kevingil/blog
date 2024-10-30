
from typing import Optional
from pydantic import BaseModel, Field, ConfigDict
from apispec_pydantic_plugin import Registry 

class UserSchema(BaseModel):
    id: Optional[int] = Field(default=None, exclude=True)
    password: str = Field(default=None, exclude=True)
    model_config = ConfigDict(
        from_attributes=True,
        exclude={"_password"},
    )

Registry.register(UserSchema)
