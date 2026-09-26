use std::io::Write;

use blog_backend::setup::{
    FirstRun, MEMBER_ROLE, SUPERUSER_ROLE, first_run, load_verification_seed, registration_role,
    validate_seed,
};

#[test]
fn empty_database_with_an_author_seed_applies_that_profile() {
    let seed = load_verification_seed(std::path::Path::new("../seed/verification.json"))
        .expect("seed file")
        .expect("seed file exists");
    validate_seed(&seed).unwrap();
    assert_eq!(first_run(0, seed.author.is_some()), FirstRun::ApplySeed);
    assert_eq!(seed.author.unwrap().name, "Ada Lovelace");
}

#[test]
fn empty_database_without_a_seed_author_promotes_the_first_user() {
    assert_eq!(first_run(0, false), FirstRun::FirstUserBecomesAdmin);
    assert_eq!(registration_role(0), SUPERUSER_ROLE);
}

#[test]
fn later_registrations_stay_members_and_seed_does_not_rerun() {
    assert_eq!(registration_role(1), MEMBER_ROLE);
    assert_eq!(first_run(2, true), FirstRun::AlreadyInitialized);
}

#[test]
fn missing_seed_file_is_not_an_error() {
    let missing = std::env::temp_dir().join("blog-missing-seed.json");
    let _ = std::fs::remove_file(&missing);
    assert!(load_verification_seed(&missing).unwrap().is_none());
}

#[test]
fn seed_author_requires_a_password() {
    let dir = std::env::temp_dir();
    let path = dir.join("blog-invalid-seed.json");
    let mut file = std::fs::File::create(&path).unwrap();
    write!(
        file,
        r#"{{"author":{{"name":"A","email":"a@example.test","password":""}}}}"#
    )
    .unwrap();
    let seed = load_verification_seed(&path).unwrap().unwrap();
    assert!(validate_seed(&seed).is_err());
    let _ = std::fs::remove_file(path);
}
