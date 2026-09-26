use std::{fs, path::Path};

use serde::Deserialize;
use serde_json::Value;

use crate::error::AppError;

/// Role stored on `account.role` for the first account in an empty database.
pub const SUPERUSER_ROLE: &str = "admin";

/// Role stored on every account created after the first.
pub const MEMBER_ROLE: &str = "user";

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum FirstRun {
    /// Database already has accounts. Leave site settings and roles alone.
    AlreadyInitialized,
    /// No accounts yet, and the seed file names an author. Insert that author
    /// as the superuser and point site settings at their profile.
    ApplySeed,
    /// No accounts and no seed author. The next registration becomes the
    /// superuser and the public author.
    FirstUserBecomesAdmin,
}

#[derive(Debug, Clone, PartialEq, Deserialize)]
pub struct VerificationSeed {
    pub author: Option<SeedAuthor>,
    pub article: Option<SeedArticle>,
}

#[derive(Debug, Clone, PartialEq, Deserialize)]
pub struct SeedAuthor {
    pub name: String,
    pub email: String,
    pub password: String,
    pub bio: Option<String>,
    pub email_public: Option<String>,
    pub meta_description: Option<String>,
    pub profile_image: Option<String>,
    pub social_links: Option<serde_json::Map<String, Value>>,
}

#[derive(Debug, Clone, PartialEq, Deserialize)]
pub struct SeedArticle {
    pub slug: String,
    pub title: String,
    pub content: String,
}

pub fn registration_role(existing_accounts: i64) -> &'static str {
    if existing_accounts == 0 {
        SUPERUSER_ROLE
    } else {
        MEMBER_ROLE
    }
}

pub fn first_run(existing_accounts: i64, seed_has_author: bool) -> FirstRun {
    if existing_accounts > 0 {
        FirstRun::AlreadyInitialized
    } else if seed_has_author {
        FirstRun::ApplySeed
    } else {
        FirstRun::FirstUserBecomesAdmin
    }
}

pub fn load_verification_seed(path: &Path) -> Result<Option<VerificationSeed>, AppError> {
    if !path.exists() {
        return Ok(None);
    }
    let bytes = fs::read(path).map_err(|_| AppError::Internal)?;
    serde_json::from_slice(&bytes).map_err(|_| AppError::InvalidInput("seed file is not valid JSON".to_owned()))
}

pub fn validate_seed(seed: &VerificationSeed) -> Result<(), AppError> {
    let Some(author) = &seed.author else {
        return Ok(());
    };
    if author.name.trim().is_empty() || author.email.trim().is_empty() || author.password.is_empty()
    {
        return Err(AppError::InvalidInput(
            "seed author needs name, email, and password".to_owned(),
        ));
    }
    if let Some(article) = &seed.article
        && (article.slug.trim().is_empty()
            || article.title.trim().is_empty()
            || article.content.trim().is_empty())
    {
        return Err(AppError::InvalidInput(
            "seed article needs slug, title, and content".to_owned(),
        ));
    }
    Ok(())
}
