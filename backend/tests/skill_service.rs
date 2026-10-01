use std::sync::Arc;

use blog_backend::{
    core::skill::{
        CreateAgentSkill, InMemorySkillRepository, SkillContextPort, SkillService,
        format_active_skills,
    },
    error::AppError,
};
use tokio_util::sync::CancellationToken;

#[tokio::test]
async fn skill_service_creates_toggles_and_excludes_disabled_from_prompt() {
    let service = SkillService::new(
        Arc::new(InMemorySkillRepository::default()),
        CancellationToken::new(),
    );
    let created = service
        .create(CreateAgentSkill {
            name: "Editorial voice".to_owned(),
            description: "Keep a calm informational tone".to_owned(),
            instructions: "Avoid hype. Prefer concrete claims.".to_owned(),
            enabled: true,
        })
        .await
        .expect("create skill");
    assert_eq!(created.name, "Editorial voice");
    assert!(created.enabled);

    let duplicate = service
        .create(CreateAgentSkill {
            name: "editorial voice".to_owned(),
            instructions: "Different body".to_owned(),
            enabled: true,
            ..CreateAgentSkill::default()
        })
        .await;
    assert!(matches!(duplicate, Err(AppError::InvalidInput(_))));

    let prompt = service.active_prompt().await.expect("active prompt");
    assert!(prompt.contains("### Editorial voice"));
    assert!(prompt.contains("Avoid hype. Prefer concrete claims."));

    let disabled = service
        .update(
            created.id,
            blog_backend::core::skill::UpdateAgentSkill {
                enabled: Some(false),
                ..blog_backend::core::skill::UpdateAgentSkill::default()
            },
        )
        .await
        .expect("disable skill");
    assert!(!disabled.enabled);
    assert!(service.active_prompt().await.expect("empty prompt").is_empty());
    assert_eq!(service.list().await.expect("list").len(), 1);
}

#[test]
fn format_active_skills_omits_empty_catalog() {
    assert!(format_active_skills(&[]).is_empty());
}
