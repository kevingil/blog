use std::collections::BTreeMap;

use serde::{Deserialize, Serialize};
use utoipa::ToSchema;

use crate::core::auth::{
    AccountUpdate, LoginInput, LoginResult, PasswordUpdate, RegistrationInput, UserData,
};

#[derive(Debug, Deserialize, ToSchema)]
#[serde(rename_all = "camelCase")]
pub struct LoginRequest {
    #[schema(format = Email)]
    pub email: String,
    #[schema(min_length = 6)]
    pub password: String,
}

#[derive(Debug, Deserialize, ToSchema)]
#[serde(rename_all = "camelCase")]
pub struct RegisterRequest {
    #[schema(min_length = 2, max_length = 100)]
    pub name: String,
    #[schema(format = Email)]
    pub email: String,
    /// Password UTF-8 encoding must not exceed 72 bytes.
    #[schema(min_length = 8, max_length = 72)]
    pub password: String,
}

#[derive(Debug, Serialize, ToSchema)]
#[serde(rename_all = "camelCase")]
pub struct LoginResponse {
    pub token: String,
    pub user: UserResponse,
}

#[derive(Debug, Serialize, ToSchema)]
#[serde(rename_all = "camelCase")]
pub struct UserResponse {
    pub id: String,
    pub name: String,
    pub email: String,
    pub role: String,
}

#[derive(Debug, Deserialize, ToSchema)]
#[serde(rename_all = "camelCase")]
pub struct UpdateAccountRequest {
    #[schema(min_length = 2, max_length = 100)]
    pub name: String,
    #[schema(format = Email)]
    pub email: String,
}

#[derive(Debug, Deserialize, ToSchema)]
#[serde(rename_all = "camelCase")]
pub struct UpdatePasswordRequest {
    /// Password UTF-8 encoding must not exceed 72 bytes.
    #[schema(min_length = 6, max_length = 72)]
    pub current_password: String,
    /// Password UTF-8 encoding must not exceed 72 bytes.
    #[schema(min_length = 6, max_length = 72)]
    pub new_password: String,
}

#[derive(Debug, Deserialize, ToSchema)]
#[serde(rename_all = "camelCase")]
pub struct DeleteAccountRequest {
    /// Password UTF-8 encoding must not exceed 72 bytes.
    #[schema(min_length = 1, max_length = 72)]
    pub password: String,
}

#[derive(Debug, Serialize, ToSchema)]
pub struct MessageResponse {
    pub message: &'static str,
}

#[derive(Debug, Serialize, ToSchema)]
pub struct AuthErrorResponse {
    pub error: String,
    pub code: &'static str,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub details: Option<BTreeMap<String, String>>,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ValidationIssue {
    pub field: &'static str,
    pub message: String,
}

pub trait Validate {
    fn validate(&self) -> Vec<ValidationIssue>;
}

impl Validate for LoginRequest {
    fn validate(&self) -> Vec<ValidationIssue> {
        let mut issues = Vec::new();
        validate_email(&mut issues, "Email", &self.email);
        validate_length(&mut issues, "Password", &self.password, 6, None);
        issues
    }
}

impl Validate for RegisterRequest {
    fn validate(&self) -> Vec<ValidationIssue> {
        let mut issues = Vec::new();
        validate_length(&mut issues, "Name", &self.name, 2, Some(100));
        validate_email(&mut issues, "Email", &self.email);
        validate_length(&mut issues, "Password", &self.password, 8, None);
        validate_password_bytes(&mut issues, "Password", &self.password);
        issues
    }
}

impl Validate for UpdateAccountRequest {
    fn validate(&self) -> Vec<ValidationIssue> {
        let mut issues = Vec::new();
        validate_length(&mut issues, "Name", &self.name, 2, Some(100));
        validate_email(&mut issues, "Email", &self.email);
        issues
    }
}

impl Validate for UpdatePasswordRequest {
    fn validate(&self) -> Vec<ValidationIssue> {
        let mut issues = Vec::new();
        validate_length(
            &mut issues,
            "CurrentPassword",
            &self.current_password,
            6,
            None,
        );
        validate_password_bytes(&mut issues, "CurrentPassword", &self.current_password);
        validate_length(&mut issues, "NewPassword", &self.new_password, 6, None);
        validate_password_bytes(&mut issues, "NewPassword", &self.new_password);
        issues
    }
}

impl Validate for DeleteAccountRequest {
    fn validate(&self) -> Vec<ValidationIssue> {
        let mut issues = Vec::new();
        if self.password.is_empty() {
            issues.push(ValidationIssue {
                field: "Password",
                message: "Password is required".to_owned(),
            });
        } else {
            validate_password_bytes(&mut issues, "Password", &self.password);
        }
        issues
    }
}

fn validate_password_bytes(issues: &mut Vec<ValidationIssue>, field: &'static str, value: &str) {
    if value.len() > 72 {
        issues.push(ValidationIssue {
            field,
            message: format!("{field} must be at most 72 bytes when UTF-8 encoded"),
        });
    }
}

impl From<LoginRequest> for LoginInput {
    fn from(request: LoginRequest) -> Self {
        Self {
            email: request.email,
            password: request.password,
        }
    }
}

impl From<RegisterRequest> for RegistrationInput {
    fn from(request: RegisterRequest) -> Self {
        Self {
            name: request.name,
            email: request.email,
            password: request.password,
        }
    }
}

impl From<UpdateAccountRequest> for AccountUpdate {
    fn from(request: UpdateAccountRequest) -> Self {
        Self {
            name: request.name,
            email: request.email,
        }
    }
}

impl From<UpdatePasswordRequest> for PasswordUpdate {
    fn from(request: UpdatePasswordRequest) -> Self {
        Self {
            current_password: request.current_password,
            new_password: request.new_password,
        }
    }
}

impl From<LoginResult> for LoginResponse {
    fn from(result: LoginResult) -> Self {
        Self {
            token: result.token,
            user: UserResponse::from(result.user),
        }
    }
}

impl From<UserData> for UserResponse {
    fn from(user: UserData) -> Self {
        Self {
            id: user.id,
            name: user.name,
            email: user.email,
            role: user.role,
        }
    }
}

fn validate_email(issues: &mut Vec<ValidationIssue>, field: &'static str, value: &str) {
    if value.is_empty() {
        issues.push(ValidationIssue {
            field,
            message: format!("{field} is required"),
        });
    } else if !email_address::EmailAddress::is_valid(value) {
        issues.push(ValidationIssue {
            field,
            message: format!("{field} must be a valid email address"),
        });
    }
}

fn validate_length(
    issues: &mut Vec<ValidationIssue>,
    field: &'static str,
    value: &str,
    minimum: usize,
    maximum: Option<usize>,
) {
    if value.is_empty() {
        issues.push(ValidationIssue {
            field,
            message: format!("{field} is required"),
        });
    } else if value.chars().count() < minimum {
        issues.push(ValidationIssue {
            field,
            message: format!("{field} must be at least {minimum} characters"),
        });
    } else if let Some(maximum) = maximum
        && value.chars().count() > maximum
    {
        issues.push(ValidationIssue {
            field,
            message: format!("{field} must be at most {maximum} characters"),
        });
    }
}
