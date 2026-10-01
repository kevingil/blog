use std::path::PathBuf;

use chrono::Utc;
use diesel::{Connection, ExpressionMethods, PgConnection, QueryDsl, RunQueryDsl};
use secrecy::ExposeSecret;
use serde_json::Value;
use uuid::Uuid;

use blog_backend::{
    config::database_url_from_env,
    schema::{account, article, site_settings},
    setup::{FirstRun, first_run, load_verification_seed, validate_seed},
};

fn main() -> anyhow::Result<()> {
    dotenvy::dotenv().ok();
    let path = std::env::var("SEED_FILE")
        .map(PathBuf::from)
        .unwrap_or_else(|_| PathBuf::from("seed/verification.json"));
    let seed = load_verification_seed(&path)?;
    let Some(seed) = seed else {
        println!(
            "No seed file at {}. An empty database promotes the first registered user to admin.",
            path.display()
        );
        return Ok(());
    };
    validate_seed(&seed)?;

    let database_url = database_url_from_env()?;
    let mut connection = PgConnection::establish(database_url.expose_secret())?;
    let existing_accounts: i64 = account::table.count().get_result(&mut connection)?;
    match first_run(existing_accounts, seed.author.is_some()) {
        FirstRun::AlreadyInitialized => {
            println!("Database already has accounts; leaving seed data untouched.");
        }
        FirstRun::FirstUserBecomesAdmin => {
            println!(
                "Seed file {} has no author. The first registered user becomes admin and the public author.",
                path.display()
            );
        }
        FirstRun::ApplySeed => {
            let author = seed
                .author
                .as_ref()
                .ok_or_else(|| anyhow::anyhow!("seed author missing"))?;
            apply_seed(&mut connection, author, seed.article.as_ref())?;
            println!(
                "Seeded superuser {} <{}> as the public author.",
                author.name, author.email
            );
        }
    }
    Ok(())
}

fn apply_seed(
    connection: &mut PgConnection,
    author: &blog_backend::setup::SeedAuthor,
    article_seed: Option<&blog_backend::setup::SeedArticle>,
) -> anyhow::Result<()> {
    let author_id = Uuid::new_v4();
    let password_hash = bcrypt::non_truncating_hash(&author.password, 10)
        .map_err(|error| anyhow::anyhow!("failed to hash seed password: {error}"))?;
    let social_links = author
        .social_links
        .clone()
        .map(|links| Value::Object(links));
    let now = Utc::now();

    connection.transaction::<_, anyhow::Error, _>(|connection| {
        diesel::insert_into(account::table)
            .values((
                account::id.eq(author_id),
                account::name.eq(&author.name),
                account::email.eq(&author.email),
                account::password_hash.eq(&password_hash),
                account::role.eq(blog_backend::setup::SUPERUSER_ROLE),
                account::bio.eq(&author.bio),
                account::profile_image.eq(&author.profile_image),
                account::email_public.eq(
                    author
                        .email_public
                        .clone()
                        .or_else(|| Some(author.email.clone())),
                ),
                account::social_links.eq(social_links),
                account::meta_description.eq(&author.meta_description),
                account::created_at.eq(now),
                account::updated_at.eq(now),
            ))
            .execute(connection)?;

        diesel::update(site_settings::table.find(1))
            .set((
                site_settings::public_profile_type.eq("user"),
                site_settings::public_user_id.eq(author_id),
                site_settings::updated_at.eq(now),
            ))
            .execute(connection)?;

        if let Some(article_seed) = article_seed {
            diesel::insert_into(article::table)
                .values((
                    article::id.eq(Uuid::new_v4()),
                    article::slug.eq(&article_seed.slug),
                    article::author_id.eq(author_id),
                    article::draft_title.eq(&article_seed.title),
                    article::draft_content.eq(&article_seed.content),
                    article::published_title.eq(&article_seed.title),
                    article::published_content.eq(&article_seed.content),
                    article::published_at.eq(now),
                    article::created_at.eq(now),
                    article::updated_at.eq(now),
                ))
                .execute(connection)?;
        }
        Ok(())
    })?;
    Ok(())
}
