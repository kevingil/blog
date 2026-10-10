use std::{
    error::Error,
    io,
    sync::{Arc, Mutex, MutexGuard},
};

use async_trait::async_trait;
use axum::{
    Router,
    body::Body,
    http::{Method, Request, StatusCode, header::CONTENT_TYPE},
};
use blog_backend::{
    api::auth::{AuthState, routes},
    core::auth::{
        Account, AccountId, AccountRepository, AccountUpdate, AuthService, LoginInput,
        PasswordUpdate, RegistrationInput,
    },
    error::AppError,
};
use chrono::Utc;
use http_body_util::BodyExt;
use jsonwebtoken::{Algorithm, EncodingKey, Header, encode};
use serde_json::{Value, json};
use tower::ServiceExt;
use uuid::Uuid;

const TEST_SECRET: &str = "test-secret-key-for-unit-tests-only";
type TestResult<T = ()> = Result<T, Box<dyn Error + Send + Sync>>;

#[derive(Default)]
struct RepositoryState {
    accounts: Vec<Account>,
    created: Vec<Account>,
    identity_updates: Vec<(AccountId, String, String)>,
    password_updates: Vec<AccountId>,
    deleted: Vec<AccountId>,
    public_authors: Vec<AccountId>,
    fail_find_by_email: bool,
    conditional_password_update_fails: bool,
    conditional_delete_fails: bool,
}

#[derive(Default)]
struct MockAccountRepository {
    state: Mutex<RepositoryState>,
}

impl MockAccountRepository {
    fn with_accounts(accounts: Vec<Account>) -> Self {
        Self {
            state: Mutex::new(RepositoryState {
                accounts,
                ..RepositoryState::default()
            }),
        }
    }

    fn state(&self) -> MutexGuard<'_, RepositoryState> {
        match self.state.lock() {
            Ok(state) => state,
            Err(poisoned) => poisoned.into_inner(),
        }
    }
}

#[async_trait]
impl AccountRepository for MockAccountRepository {
    async fn find_by_id(&self, id: AccountId) -> Result<Option<Account>, AppError> {
        Ok(self
            .state()
            .accounts
            .iter()
            .find(|account| account.id == id)
            .cloned())
    }

    async fn find_by_email(&self, email: &str) -> Result<Option<Account>, AppError> {
        let state = self.state();
        if state.fail_find_by_email {
            return Err(AppError::database("no underlying error was recorded"));
        }
        Ok(state
            .accounts
            .iter()
            .find(|account| account.email == email)
            .cloned())
    }

    async fn create(&self, account: &Account) -> Result<(), AppError> {
        let mut state = self.state();
        state.created.push(account.clone());
        state.accounts.push(account.clone());
        Ok(())
    }

    async fn update_identity(
        &self,
        id: AccountId,
        name: &str,
        email: &str,
    ) -> Result<bool, AppError> {
        let mut state = self.state();
        state
            .identity_updates
            .push((id, name.to_owned(), email.to_owned()));
        let Some(account) = state.accounts.iter_mut().find(|account| account.id == id) else {
            return Ok(false);
        };
        account.name = name.to_owned();
        account.email = email.to_owned();
        Ok(true)
    }

    async fn update_password_if_current(
        &self,
        id: AccountId,
        expected_password_hash: &str,
        new_password_hash: &str,
    ) -> Result<bool, AppError> {
        let mut state = self.state();
        if state.conditional_password_update_fails {
            return Ok(false);
        }
        let Some(account) = state
            .accounts
            .iter_mut()
            .find(|account| account.id == id && account.password_hash == expected_password_hash)
        else {
            return Ok(false);
        };
        account.password_hash = new_password_hash.to_owned();
        state.password_updates.push(id);
        Ok(true)
    }

    async fn delete_if_password_hash(
        &self,
        id: AccountId,
        expected_password_hash: &str,
    ) -> Result<bool, AppError> {
        let mut state = self.state();
        if state.conditional_delete_fails {
            return Ok(false);
        }
        let before = state.accounts.len();
        state
            .accounts
            .retain(|account| account.id != id || account.password_hash != expected_password_hash);
        let deleted = state.accounts.len() != before;
        if deleted {
            state.deleted.push(id);
        }
        Ok(deleted)
    }

    async fn count(&self) -> Result<i64, AppError> {
        Ok(i64::try_from(self.state().accounts.len()).unwrap_or(i64::MAX))
    }

    async fn assign_public_author(&self, id: AccountId) -> Result<(), AppError> {
        self.state().public_authors.push(id);
        Ok(())
    }
}

fn service(repository: Arc<MockAccountRepository>) -> Result<AuthService, AppError> {
    AuthService::new(repository, TEST_SECRET)
}

async fn test_account(email: &str, password: &str) -> TestResult<Account> {
    let password_hash = service(Arc::new(MockAccountRepository::default()))?
        .hash_password(password)
        .await?;
    Ok(Account {
        id: AccountId(Uuid::new_v4()),
        name: "Test User".to_owned(),
        email: email.to_owned(),
        password_hash,
        role: "user".to_owned(),
        created_at: None,
        updated_at: None,
        bio: Some("bio stays intact".to_owned()),
        profile_image: Some("profile.png".to_owned()),
        email_public: Some("public@example.com".to_owned()),
        social_links: Some([("website".to_owned(), json!("https://example.com"))].into()),
        meta_description: Some("metadata stays intact".to_owned()),
        organization_id: Some(Uuid::new_v4()),
    })
}

fn auth_router(repository: Arc<MockAccountRepository>) -> TestResult<Router> {
    let state = AuthState::new(Arc::new(service(repository)?));
    Ok(routes::router::<AuthState>().with_state(state).into())
}

async fn call(
    router: Router,
    method: Method,
    path: &str,
    content_type: &str,
    body: String,
    authorization: Option<&str>,
) -> TestResult<(StatusCode, Value)> {
    let mut builder = Request::builder()
        .method(method)
        .uri(path)
        .header(CONTENT_TYPE, content_type);
    if let Some(authorization) = authorization {
        builder = builder.header("authorization", authorization);
    }
    let request = builder.body(Body::from(body))?;
    let response = router.oneshot(request).await?;
    let status = response.status();
    let bytes = response.into_body().collect().await?.to_bytes();
    Ok((status, serde_json::from_slice(&bytes)?))
}

async fn call_json(
    router: Router,
    method: Method,
    path: &str,
    body: Value,
    authorization: Option<&str>,
) -> TestResult<(StatusCode, Value)> {
    call(
        router,
        method,
        path,
        "application/json",
        body.to_string(),
        authorization,
    )
    .await
}

fn multipart(fields: &[(&str, &str)]) -> (String, String) {
    let boundary = "auth-test-boundary";
    let mut body = String::new();
    for (name, value) in fields {
        body.push_str("--");
        body.push_str(boundary);
        body.push_str("\r\nContent-Disposition: form-data; name=\"");
        body.push_str(name);
        body.push_str("\"\r\n\r\n");
        body.push_str(value);
        body.push_str("\r\n");
    }
    body.push_str("--");
    body.push_str(boundary);
    body.push_str("--\r\n");
    (format!("multipart/form-data; boundary={boundary}"), body)
}

fn required_string<'a>(value: &'a Value, pointer: &str) -> TestResult<&'a str> {
    value
        .pointer(pointer)
        .and_then(Value::as_str)
        .ok_or_else(|| io::Error::other(format!("missing string at {pointer}")).into())
}

#[tokio::test]
async fn bcrypt_uses_cost_ten_and_preserves_go_byte_boundaries() -> TestResult {
    let auth = service(Arc::new(MockAccountRepository::default()))?;
    let first = auth.hash_password("mySecurePassword123").await?;
    let second = auth.hash_password("mySecurePassword123").await?;
    assert!(first.starts_with("$2b$10$"));
    assert_ne!(first, second);

    let ascii_72 = "a".repeat(72);
    let ascii_hash = auth.hash_password(&ascii_72).await?;
    assert!(ascii_hash.starts_with("$2b$10$"));

    let multibyte_72 = "é".repeat(36);
    let multibyte_hash = auth.hash_password(&multibyte_72).await?;
    assert!(multibyte_hash.starts_with("$2b$10$"));

    assert!(matches!(
        auth.hash_password(&format!("{ascii_72}a")).await,
        Err(AppError::InvalidInput(_))
    ));
    assert!(matches!(
        auth.hash_password(&format!("{multibyte_72}a")).await,
        Err(AppError::InvalidInput(_))
    ));
    Ok(())
}

#[test]
fn constructor_rejects_empty_jwt_secret() {
    assert!(matches!(
        AuthService::new(Arc::new(MockAccountRepository::default()), ""),
        Err(AppError::InvalidInput(_))
    ));
}

#[tokio::test]
async fn bcrypt_verification_handles_success_mismatch_empty_and_invalid_hashes() -> TestResult {
    let account = test_account("test@example.com", "correctPassword123").await?;
    let repository = Arc::new(MockAccountRepository::with_accounts(vec![account]));
    let auth = service(repository)?;

    assert!(
        auth.login(LoginInput {
            email: "test@example.com".to_owned(),
            password: "correctPassword123".to_owned(),
        })
        .await
        .is_ok()
    );
    for password in ["wrongPassword", "", &"x".repeat(73)] {
        assert!(matches!(
            auth.login(LoginInput {
                email: "test@example.com".to_owned(),
                password: password.to_owned(),
            })
            .await,
            Err(AppError::Unauthorized)
        ));
    }

    let mut invalid_hash_account = test_account("invalid@example.com", "password123").await?;
    invalid_hash_account.password_hash = "invalidhash".to_owned();
    let invalid = service(Arc::new(MockAccountRepository::with_accounts(vec![
        invalid_hash_account,
    ])))?;
    assert!(matches!(
        invalid
            .login(LoginInput {
                email: "invalid@example.com".to_owned(),
                password: "password123".to_owned(),
            })
            .await,
        Err(AppError::Unauthorized)
    ));
    Ok(())
}

#[tokio::test]
async fn existing_go_bcrypt_hashes_remain_verifiable() -> TestResult {
    let mut normal = test_account("go-normal@example.com", "temporary").await?;
    normal.password_hash =
        "$2a$10$4SatGeT0ymV1ycP7gHgXJ.n/3iHU6yvetwWKz6SdZnOFYGcoms8/G".to_owned();
    let mut boundary = test_account("go-boundary@example.com", "temporary").await?;
    boundary.password_hash =
        "$2a$10$o1uQTVQylM4z8Oa7w68vsuf6mLIc.RK/aPfyf7ww9FOL8yI4l/ckm".to_owned();
    let auth = service(Arc::new(MockAccountRepository::with_accounts(vec![
        normal, boundary,
    ])))?;

    assert!(
        auth.login(LoginInput {
            email: "go-normal@example.com".to_owned(),
            password: "go-compatible-password".to_owned(),
        })
        .await
        .is_ok()
    );
    assert!(
        auth.login(LoginInput {
            email: "go-boundary@example.com".to_owned(),
            password: "x".repeat(72),
        })
        .await
        .is_ok()
    );
    Ok(())
}

#[tokio::test]
async fn authentication_needs_a_current_hs256_token_for_an_existing_account() -> TestResult {
    let account = test_account("test@example.com", "password123").await?;
    let account_id = account.id;
    let auth = service(Arc::new(MockAccountRepository::with_accounts(vec![
        account.clone(),
    ])))?;
    let valid = auth.issue_token(&account)?;
    assert_eq!(auth.authenticate(&valid).await?.id, account_id);
    let sv = auth.validate_token(&valid)?.claims.sv;

    let now = Utc::now().timestamp();
    let sub = account_id.0.to_string();
    let cases = [
        (
            "no expiry",
            Header::new(Algorithm::HS256),
            json!({"sub": sub, "iat": now, "sv": sv}),
        ),
        (
            "expired",
            Header::new(Algorithm::HS256),
            json!({"sub": sub, "exp": now - 1, "iat": now - 60, "sv": sv}),
        ),
        (
            "wrong algorithm",
            Header::new(Algorithm::HS384),
            json!({"sub": sub, "exp": now + 3600, "iat": now, "sv": sv}),
        ),
        (
            "malformed subject",
            Header::new(Algorithm::HS256),
            json!({"sub": "not-a-valid-uuid", "exp": now + 3600, "iat": now, "sv": sv}),
        ),
        (
            "unknown account",
            Header::new(Algorithm::HS256),
            json!({"sub": Uuid::new_v4().to_string(), "exp": now + 3600, "iat": now, "sv": sv}),
        ),
        (
            "stale session version",
            Header::new(Algorithm::HS256),
            json!({"sub": sub, "exp": now + 3600, "iat": now, "sv": "stale"}),
        ),
        (
            "issued before session versions",
            Header::new(Algorithm::HS256),
            json!({"sub": sub, "exp": now + 3600}),
        ),
    ];
    for (case, header, claims) in cases {
        let token = encode(
            &header,
            &claims,
            &EncodingKey::from_secret(TEST_SECRET.as_bytes()),
        )?;
        assert!(
            matches!(auth.authenticate(&token).await, Err(AppError::Unauthorized)),
            "{case} token was accepted"
        );
    }
    Ok(())
}

#[tokio::test]
async fn issued_tokens_last_thirty_days() -> TestResult {
    let account = test_account("test@example.com", "password123").await?;
    let auth = service(Arc::new(MockAccountRepository::default()))?;
    let before = Utc::now().timestamp();
    let token = auth.issue_token(&account)?;
    let after = Utc::now().timestamp();
    let decoded = auth.validate_token(&token)?;
    let expiry = i64::try_from(decoded.claims.exp)?;
    let issued_at = i64::try_from(decoded.claims.iat)?;

    assert_eq!(decoded.header.alg, Algorithm::HS256);
    assert!((before..=after).contains(&issued_at));
    assert_eq!(expiry - issued_at, 30 * 86_400);
    Ok(())
}

#[tokio::test]
async fn tokens_stop_working_after_a_password_change_or_account_deletion() -> TestResult {
    let account = test_account("test@example.com", "oldpassword").await?;
    let account_id = account.id;
    let repository = Arc::new(MockAccountRepository::with_accounts(vec![account.clone()]));
    let auth = service(repository)?;
    let before_change = auth.issue_token(&account)?;

    let after_change = auth
        .update_password(account_id, PasswordUpdate {
            current_password: "oldpassword".to_owned(),
            new_password: "newpassword".to_owned(),
        })
        .await?;
    assert!(matches!(
        auth.authenticate(&before_change).await,
        Err(AppError::Unauthorized)
    ));
    assert_eq!(auth.authenticate(&after_change).await?.id, account_id);

    auth.delete_account(account_id, "newpassword").await?;
    assert!(matches!(
        auth.authenticate(&after_change).await,
        Err(AppError::Unauthorized)
    ));
    Ok(())
}

#[tokio::test]
async fn login_returns_user_data_and_hides_lookup_failures() -> TestResult {
    let account = test_account("test@example.com", "correctpassword").await?;
    let account_id = account.id;
    let repository = Arc::new(MockAccountRepository::with_accounts(vec![account]));
    let result = service(repository)?
        .login(LoginInput {
            email: "test@example.com".to_owned(),
            password: "correctpassword".to_owned(),
        })
        .await?;
    assert!(!result.token.is_empty());
    assert_eq!(result.user.id, account_id.0.to_string());
    assert_eq!(result.user.name, "Test User");
    assert_eq!(result.user.email, "test@example.com");
    assert_eq!(result.user.role, "user");

    let repository = Arc::new(MockAccountRepository::default());
    repository.state().fail_find_by_email = true;
    assert!(matches!(
        service(repository)?
            .login(LoginInput {
                email: "test@example.com".to_owned(),
                password: "correctpassword".to_owned(),
            })
            .await,
        Err(AppError::Unauthorized)
    ));
    Ok(())
}

#[tokio::test]
async fn registration_creates_a_user_and_rejects_conflicts() -> TestResult {
    let repository = Arc::new(MockAccountRepository::default());
    let auth = service(repository.clone())?;
    let input = RegistrationInput {
        name: "New User".to_owned(),
        email: "newuser@example.com".to_owned(),
        password: "securepassword".to_owned(),
    };
    auth.register(input.clone(), None).await?;
    let created = repository
        .state()
        .created
        .first()
        .cloned()
        .ok_or_else(|| io::Error::other("registration did not create an account"))?;
    assert_eq!(created.name, "New User");
    assert_eq!(created.email, "newuser@example.com");
    assert_eq!(created.role, "admin");
    assert_eq!(repository.state().public_authors, vec![created.id]);
    assert!(created.password_hash.starts_with("$2b$10$"));
    assert!(matches!(
        auth.register(input, Some(&created)).await,
        Err(AppError::Conflict(_))
    ));
    Ok(())
}

#[tokio::test]
async fn registration_after_the_first_account_needs_an_admin_and_creates_members() -> TestResult {
    let member = test_account("member@example.com", "password123").await?;
    let mut owner = test_account("owner@example.com", "password123").await?;
    owner.role = "admin".to_owned();
    let repository = Arc::new(MockAccountRepository::with_accounts(vec![
        owner.clone(),
        member.clone(),
    ]));
    let auth = service(repository.clone())?;
    let input = RegistrationInput {
        name: "Second User".to_owned(),
        email: "second@example.com".to_owned(),
        password: "securepassword".to_owned(),
    };

    for requested_by in [None, Some(&member)] {
        assert!(matches!(
            auth.register(input.clone(), requested_by).await,
            Err(AppError::Forbidden)
        ));
    }
    assert!(matches!(
        auth.register(
            RegistrationInput {
                email: "member@example.com".to_owned(),
                ..input.clone()
            },
            None
        )
        .await,
        Err(AppError::Forbidden)
    ));
    assert!(repository.state().created.is_empty());

    auth.register(input, Some(&owner)).await?;
    let created = repository
        .state()
        .created
        .first()
        .cloned()
        .ok_or_else(|| io::Error::other("registration did not create an account"))?;
    assert_eq!(created.role, "user");
    assert!(repository.state().public_authors.is_empty());
    Ok(())
}

#[tokio::test]
async fn get_account_returns_the_account_or_not_found() -> TestResult {
    let account = test_account("test@example.com", "password123").await?;
    let account_id = account.id;
    let auth = service(Arc::new(MockAccountRepository::with_accounts(vec![
        account,
    ])))?;
    assert_eq!(auth.get_account(account_id).await?.id, account_id);
    assert!(matches!(
        auth.get_account(AccountId(Uuid::new_v4())).await,
        Err(AppError::NotFound)
    ));
    Ok(())
}

#[tokio::test]
async fn identity_update_is_scoped_and_handles_email_conflicts() -> TestResult {
    let account = test_account("old@example.com", "password123").await?;
    let original = account.clone();
    let account_id = account.id;
    let repository = Arc::new(MockAccountRepository::with_accounts(vec![account]));
    let auth = service(repository.clone())?;

    auth.update_account(account_id, AccountUpdate {
        name: "New Name".to_owned(),
        email: "new@example.com".to_owned(),
    })
    .await?;
    let updated = repository
        .state()
        .accounts
        .first()
        .cloned()
        .ok_or_else(|| io::Error::other("updated account missing"))?;
    assert_eq!(updated.name, "New Name");
    assert_eq!(updated.email, "new@example.com");
    assert_eq!(updated.password_hash, original.password_hash);
    assert_eq!(updated.bio, original.bio);
    assert_eq!(updated.profile_image, original.profile_image);
    assert_eq!(updated.social_links, original.social_links);
    assert_eq!(updated.organization_id, original.organization_id);

    let taken_account = test_account("taken@example.com", "password123").await?;
    repository.state().accounts.push(taken_account);
    assert!(matches!(
        auth.update_account(account_id, AccountUpdate {
            name: "New Name".to_owned(),
            email: "taken@example.com".to_owned(),
        })
        .await,
        Err(AppError::Conflict(_))
    ));
    Ok(())
}

#[tokio::test]
async fn conditional_password_update_and_delete_fail_closed_on_races() -> TestResult {
    let account = test_account("test@example.com", "oldpassword").await?;
    let account_id = account.id;
    let repository = Arc::new(MockAccountRepository::with_accounts(vec![account]));
    let auth = service(repository.clone())?;

    repository.state().conditional_password_update_fails = true;
    assert!(matches!(
        auth.update_password(account_id, PasswordUpdate {
            current_password: "oldpassword".to_owned(),
            new_password: "newpassword".to_owned(),
        })
        .await,
        Err(AppError::Unauthorized)
    ));
    repository.state().conditional_password_update_fails = false;
    auth.update_password(account_id, PasswordUpdate {
        current_password: "oldpassword".to_owned(),
        new_password: "newpassword".to_owned(),
    })
    .await?;

    repository.state().conditional_delete_fails = true;
    assert!(matches!(
        auth.delete_account(account_id, "newpassword").await,
        Err(AppError::Unauthorized)
    ));
    repository.state().conditional_delete_fails = false;
    auth.delete_account(account_id, "newpassword").await?;
    assert_eq!(repository.state().deleted, vec![account_id]);
    Ok(())
}

#[tokio::test]
async fn auth_http_supports_json_and_frontend_multipart_fields() -> TestResult {
    let account = test_account("test@example.com", "correctpassword").await?;
    let account_id = account.id;
    let repository = Arc::new(MockAccountRepository::with_accounts(vec![account]));
    let router = auth_router(repository)?;

    let (status, body) = call_json(
        router.clone(),
        Method::POST,
        "/auth/logout",
        json!({}),
        None,
    )
    .await?;
    assert_eq!(status, StatusCode::OK);
    assert_eq!(
        body,
        json!({"data": {"message": "Logged out successfully"}})
    );

    let (status, body) = call_json(
        router.clone(),
        Method::POST,
        "/auth/login",
        json!({"email": "test@example.com", "password": "correctpassword"}),
        None,
    )
    .await?;
    assert_eq!(status, StatusCode::OK);
    assert_eq!(
        required_string(&body, "/data/user/id")?,
        account_id.0.to_string()
    );
    let token = required_string(&body, "/data/token")?.to_owned();
    let bearer = format!("Bearer {token}");

    let (content_type, body) =
        multipart(&[("name", "Updated User"), ("email", "updated@example.com")]);
    let (status, response) = call(
        router.clone(),
        Method::PUT,
        "/auth/account",
        &content_type,
        body,
        Some(&bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::OK);
    assert_eq!(
        response,
        json!({"data": {"message": "Account updated successfully"}})
    );

    let (content_type, body) = multipart(&[
        ("currentPassword", "correctpassword"),
        ("newPassword", "newpassword"),
        ("confirmPassword", "newpassword"),
    ]);
    let (status, response) = call(
        router.clone(),
        Method::PUT,
        "/auth/password",
        &content_type,
        body,
        Some(&bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::OK);
    assert_eq!(response["data"]["message"], "Password updated successfully");
    let replacement = format!("Bearer {}", required_string(&response, "/data/token")?);

    let (status, response) = call_json(
        router.clone(),
        Method::POST,
        "/auth/refresh",
        json!({}),
        Some(&bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::UNAUTHORIZED);
    assert_eq!(response["error"], "Invalid or expired token");

    let (content_type, body) = multipart(&[("password", "newpassword")]);
    let (status, response) = call(
        router,
        Method::DELETE,
        "/auth/account",
        &content_type,
        body,
        Some(&replacement),
    )
    .await?;
    assert_eq!(status, StatusCode::OK);
    assert_eq!(
        response,
        json!({"data": {"message": "Account deleted successfully"}})
    );
    Ok(())
}

#[tokio::test]
async fn refresh_renews_a_live_session_with_current_account_details() -> TestResult {
    let account = test_account("test@example.com", "correctpassword").await?;
    let account_id = account.id;
    let repository = Arc::new(MockAccountRepository::with_accounts(vec![account]));
    let router = auth_router(repository.clone())?;
    let (_, login) = call_json(
        router.clone(),
        Method::POST,
        "/auth/login",
        json!({"email": "test@example.com", "password": "correctpassword"}),
        None,
    )
    .await?;
    let bearer = format!("Bearer {}", required_string(&login, "/data/token")?);
    if let Some(stored) = repository.state().accounts.first_mut() {
        stored.name = "Renamed User".to_owned();
    }

    let (status, refreshed) = call_json(
        router.clone(),
        Method::POST,
        "/auth/refresh",
        json!({}),
        Some(&bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::OK);
    assert_eq!(
        required_string(&refreshed, "/data/user/id")?,
        account_id.0.to_string()
    );
    assert_eq!(refreshed["data"]["user"]["name"], "Renamed User");
    let renewed = format!("Bearer {}", required_string(&refreshed, "/data/token")?);
    let (status, _) = call_json(
        router.clone(),
        Method::POST,
        "/auth/refresh",
        json!({}),
        Some(&renewed),
    )
    .await?;
    assert_eq!(status, StatusCode::OK);

    for (authorization, message) in [
        (None, "Not authenticated"),
        (Some("Token abc"), "Invalid token format"),
        (Some("Bearer not-a-jwt"), "Invalid or expired token"),
    ] {
        let (status, body) = call_json(
            router.clone(),
            Method::POST,
            "/auth/refresh",
            json!({}),
            authorization,
        )
        .await?;
        assert_eq!(status, StatusCode::UNAUTHORIZED);
        assert_eq!(body["error"], message);
    }
    Ok(())
}

#[tokio::test]
async fn repeated_failed_logins_are_throttled_per_email() -> TestResult {
    let account = test_account("test@example.com", "correctpassword").await?;
    let router = auth_router(Arc::new(MockAccountRepository::with_accounts(vec![
        account,
    ])))?;
    let login = |email: &str, password: &str| {
        Request::builder()
            .method(Method::POST)
            .uri("/auth/login")
            .header(CONTENT_TYPE, "application/json")
            .body(Body::from(
                json!({"email": email, "password": password}).to_string(),
            ))
    };

    for _ in 0..10 {
        let response = router
            .clone()
            .oneshot(login("test@example.com", "wrongpassword")?)
            .await?;
        assert_eq!(response.status(), StatusCode::UNAUTHORIZED);
    }
    let response = router
        .clone()
        .oneshot(login("TEST@example.com", "correctpassword")?)
        .await?;
    assert_eq!(response.status(), StatusCode::TOO_MANY_REQUESTS);
    let retry_after = response
        .headers()
        .get("retry-after")
        .and_then(|value| value.to_str().ok())
        .and_then(|value| value.parse::<u64>().ok())
        .ok_or_else(|| io::Error::other("429 response has no Retry-After seconds"))?;
    assert!((1..=15 * 60 + 1).contains(&retry_after));
    let body: Value = serde_json::from_slice(&response.into_body().collect().await?.to_bytes())?;
    assert_eq!(body["code"], "TOO_MANY_REQUESTS");

    let response = router
        .oneshot(login("other@example.com", "wrongpassword")?)
        .await?;
    assert_eq!(response.status(), StatusCode::UNAUTHORIZED);
    Ok(())
}

#[tokio::test]
async fn auth_http_rejects_unexpected_or_malformed_multipart() -> TestResult {
    let account = test_account("test@example.com", "correctpassword").await?;
    let repository = Arc::new(MockAccountRepository::with_accounts(vec![account.clone()]));
    let router = auth_router(repository.clone())?;

    let (content_type, body) = multipart(&[
        ("email", "test@example.com"),
        ("password", "correctpassword"),
    ]);
    let (status, _) = call(
        router.clone(),
        Method::POST,
        "/auth/login",
        &content_type,
        body,
        None,
    )
    .await?;
    assert_eq!(status, StatusCode::BAD_REQUEST);

    let token = service(repository)?.issue_token(&account)?;
    let bearer = format!("Bearer {token}");

    let oversized = "x".repeat(1_025);
    for fields in [
        vec![("name", "First"), ("name", "Second")],
        vec![("name", "Updated"), ("unexpected", "value")],
        vec![("name", "x"), ("email", oversized.as_str())],
    ] {
        let (content_type, body) = multipart(&fields);
        let (status, _) = call(
            router.clone(),
            Method::PUT,
            "/auth/account",
            &content_type,
            body,
            Some(&bearer),
        )
        .await?;
        assert_eq!(status, StatusCode::BAD_REQUEST);
    }

    let (status, _) = call(
        router,
        Method::PUT,
        "/auth/account",
        "multipart/form-data; boundary=broken",
        "--broken\r\nnot-a-valid-multipart-body\r\n--broken--\r\n".to_owned(),
        Some(&bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::BAD_REQUEST);
    Ok(())
}

#[tokio::test]
async fn auth_http_validation_uses_stronger_registration_rules() -> TestResult {
    let router = auth_router(Arc::new(MockAccountRepository::default()))?;
    let (status, body) = call_json(
        router.clone(),
        Method::POST,
        "/auth/login",
        json!({"email": "not-an-email", "password": "short"}),
        None,
    )
    .await?;
    assert_eq!(status, StatusCode::BAD_REQUEST);
    assert_eq!(
        body,
        json!({
            "error": "Email: Email must be a valid email address",
            "code": "VALIDATION_ERROR",
            "details": {
                "Email": "Email must be a valid email address",
                "Password": "Password must be at least 6 characters"
            }
        })
    );

    let (status, body) = call_json(
        router,
        Method::POST,
        "/auth/register",
        json!({"name": "N", "email": "new@example.com", "password": "short"}),
        None,
    )
    .await?;
    assert_eq!(status, StatusCode::BAD_REQUEST);
    assert_eq!(
        body["details"]["Name"],
        "Name must be at least 2 characters"
    );
    assert_eq!(
        body["details"]["Password"],
        "Password must be at least 8 characters"
    );
    Ok(())
}

#[test]
fn auth_openapi_enumerates_operations_content_types_security_and_errors() -> TestResult {
    let (_, document) = routes::router::<AuthState>().split_for_parts();
    let document = serde_json::to_value(document)?;

    assert_eq!(
        document["paths"]["/auth/login"]["post"]["operationId"],
        "authLogin"
    );
    assert_eq!(
        document["paths"]["/auth/logout"]["post"]["operationId"],
        "authLogout"
    );
    assert!(document["paths"]["/auth/logout"]["post"]["security"].is_null());
    assert_eq!(
        document["paths"]["/auth/password"]["put"]["security"][0]["bearerAuth"],
        json!([])
    );
    assert_eq!(
        document["paths"]["/auth/refresh"]["post"]["operationId"],
        "authRefresh"
    );
    assert_eq!(
        document["paths"]["/auth/refresh"]["post"]["security"][0]["bearerAuth"],
        json!([])
    );
    assert_eq!(
        document["paths"]["/auth/register"]["post"]["security"],
        json!([{}, {"bearerAuth": []}])
    );
    assert!(
        document["paths"]["/auth/account"]["put"]["requestBody"]["content"]["multipart/form-data"]
            .is_object()
    );

    let expected = [
        (
            "/auth/login",
            "post",
            &["200", "400", "401", "429", "500"][..],
        ),
        ("/auth/refresh", "post", &["200", "401", "500"][..]),
        (
            "/auth/register",
            "post",
            &["201", "400", "403", "409", "500"][..],
        ),
        ("/auth/logout", "post", &["200"][..]),
        (
            "/auth/account",
            "put",
            &["200", "400", "401", "404", "409", "500"][..],
        ),
        (
            "/auth/password",
            "put",
            &["200", "400", "401", "404", "500"][..],
        ),
        (
            "/auth/account",
            "delete",
            &["200", "400", "401", "404", "500"][..],
        ),
    ];
    for (path, method, statuses) in expected {
        let responses = &document["paths"][path][method]["responses"];
        for status in statuses {
            assert!(
                responses[*status].is_object(),
                "{method} {path} missing {status}"
            );
        }
    }
    Ok(())
}
