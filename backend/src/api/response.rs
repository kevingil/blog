use serde::Serialize;
use utoipa::ToSchema;

#[derive(Debug, Serialize, ToSchema)]
pub struct SuccessResponse<T> {
    pub data: T,
}

impl<T> SuccessResponse<T> {
    pub const fn new(data: T) -> Self {
        Self { data }
    }
}
