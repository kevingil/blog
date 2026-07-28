// @generated automatically by Diesel CLI.

diesel::table! {
    use diesel::sql_types::*;
    use pgvector::sql_types::*;

    account (id) {
        id -> Uuid,
        #[max_length = 255]
        name -> Varchar,
        #[max_length = 255]
        email -> Varchar,
        #[max_length = 255]
        password_hash -> Varchar,
        #[max_length = 50]
        role -> Varchar,
        created_at -> Nullable<Timestamptz>,
        updated_at -> Nullable<Timestamptz>,
        bio -> Nullable<Text>,
        profile_image -> Nullable<Text>,
        #[max_length = 255]
        email_public -> Nullable<Varchar>,
        social_links -> Nullable<Jsonb>,
        meta_description -> Nullable<Text>,
        organization_id -> Nullable<Uuid>,
    }
}

diesel::table! {
    use diesel::sql_types::*;
    use pgvector::sql_types::*;

    article (id) {
        id -> Uuid,
        #[max_length = 255]
        slug -> Varchar,
        author_id -> Uuid,
        tag_ids -> Nullable<Array<Nullable<Int4>>>,
        created_at -> Nullable<Timestamptz>,
        updated_at -> Nullable<Timestamptz>,
        published_at -> Nullable<Timestamptz>,
        imagen_request_id -> Nullable<Uuid>,
        session_memory -> Nullable<Jsonb>,
        #[max_length = 500]
        draft_title -> Nullable<Varchar>,
        draft_content -> Nullable<Text>,
        draft_image_url -> Nullable<Text>,
        draft_embedding -> Nullable<Vector>,
        #[max_length = 500]
        published_title -> Nullable<Varchar>,
        published_content -> Nullable<Text>,
        published_image_url -> Nullable<Text>,
        published_embedding -> Nullable<Vector>,
        current_draft_version_id -> Nullable<Uuid>,
        current_published_version_id -> Nullable<Uuid>,
    }
}

diesel::table! {
    use diesel::sql_types::*;
    use pgvector::sql_types::*;

    article_source (id) {
        id -> Uuid,
        article_id -> Uuid,
        #[max_length = 500]
        title -> Nullable<Varchar>,
        content -> Text,
        url -> Nullable<Text>,
        #[max_length = 50]
        source_type -> Nullable<Varchar>,
        embedding -> Nullable<Vector>,
        meta_data -> Nullable<Jsonb>,
        created_at -> Nullable<Timestamptz>,
    }
}

diesel::table! {
    use diesel::sql_types::*;
    use pgvector::sql_types::*;

    article_version (id) {
        id -> Uuid,
        article_id -> Uuid,
        version_number -> Int4,
        #[max_length = 20]
        status -> Varchar,
        #[max_length = 500]
        title -> Varchar,
        content -> Nullable<Text>,
        image_url -> Nullable<Text>,
        embedding -> Nullable<Vector>,
        edited_by -> Nullable<Uuid>,
        created_at -> Nullable<Timestamptz>,
    }
}

diesel::table! {
    use diesel::sql_types::*;
    use pgvector::sql_types::*;

    chat_message (id) {
        id -> Uuid,
        article_id -> Uuid,
        #[max_length = 50]
        role -> Varchar,
        content -> Text,
        meta_data -> Nullable<Jsonb>,
        created_at -> Nullable<Timestamptz>,
    }
}

diesel::table! {
    use diesel::sql_types::*;
    use pgvector::sql_types::*;

    content_topic_match (id) {
        id -> Uuid,
        content_id -> Uuid,
        topic_id -> Uuid,
        similarity_score -> Float8,
        is_primary -> Nullable<Bool>,
        created_at -> Nullable<Timestamptz>,
    }
}

diesel::table! {
    use diesel::sql_types::*;
    use pgvector::sql_types::*;

    crawled_content (id) {
        id -> Uuid,
        data_source_id -> Uuid,
        url -> Text,
        #[max_length = 500]
        title -> Nullable<Varchar>,
        content -> Text,
        summary -> Nullable<Text>,
        #[max_length = 255]
        author -> Nullable<Varchar>,
        published_at -> Nullable<Timestamptz>,
        embedding -> Nullable<Vector>,
        meta_data -> Nullable<Jsonb>,
        created_at -> Nullable<Timestamptz>,
    }
}

diesel::table! {
    use diesel::sql_types::*;
    use pgvector::sql_types::*;

    data_source (id) {
        id -> Uuid,
        organization_id -> Nullable<Uuid>,
        #[max_length = 255]
        name -> Varchar,
        url -> Text,
        feed_url -> Nullable<Text>,
        #[max_length = 50]
        source_type -> Nullable<Varchar>,
        #[max_length = 50]
        crawl_frequency -> Nullable<Varchar>,
        is_enabled -> Nullable<Bool>,
        is_discovered -> Nullable<Bool>,
        discovered_from_id -> Nullable<Uuid>,
        last_crawled_at -> Nullable<Timestamptz>,
        next_crawl_at -> Nullable<Timestamptz>,
        #[max_length = 50]
        crawl_status -> Nullable<Varchar>,
        error_message -> Nullable<Text>,
        content_count -> Nullable<Int4>,
        meta_data -> Nullable<Jsonb>,
        created_at -> Nullable<Timestamptz>,
        updated_at -> Nullable<Timestamptz>,
        user_id -> Nullable<Uuid>,
        subscriber_count -> Nullable<Int4>,
    }
}

diesel::table! {
    use diesel::sql_types::*;
    use pgvector::sql_types::*;

    file_index (id) {
        id -> Uuid,
        s3_key -> Text,
        filename -> Text,
        directory_path -> Nullable<Text>,
        #[max_length = 100]
        file_type -> Nullable<Varchar>,
        file_size -> Nullable<Int8>,
        #[max_length = 100]
        content_type -> Nullable<Varchar>,
        meta_data -> Nullable<Jsonb>,
        created_at -> Nullable<Timestamptz>,
        updated_at -> Nullable<Timestamptz>,
    }
}

diesel::table! {
    use diesel::sql_types::*;
    use pgvector::sql_types::*;

    imagen_request (id) {
        id -> Uuid,
        prompt -> Text,
        #[max_length = 50]
        provider -> Varchar,
        #[max_length = 100]
        model_name -> Varchar,
        #[max_length = 255]
        request_id -> Nullable<Varchar>,
        #[max_length = 20]
        status -> Nullable<Varchar>,
        output_url -> Nullable<Text>,
        file_index_id -> Nullable<Uuid>,
        error_message -> Nullable<Text>,
        meta_data -> Nullable<Jsonb>,
        created_at -> Nullable<Timestamptz>,
        completed_at -> Nullable<Timestamptz>,
    }
}

diesel::table! {
    use diesel::sql_types::*;
    use pgvector::sql_types::*;

    insight (id) {
        id -> Uuid,
        organization_id -> Nullable<Uuid>,
        topic_id -> Nullable<Uuid>,
        #[max_length = 500]
        title -> Varchar,
        summary -> Text,
        content -> Nullable<Text>,
        key_points -> Nullable<Jsonb>,
        source_content_ids -> Nullable<Array<Nullable<Uuid>>>,
        embedding -> Nullable<Vector>,
        generated_at -> Nullable<Timestamptz>,
        period_start -> Nullable<Timestamptz>,
        period_end -> Nullable<Timestamptz>,
        is_read -> Nullable<Bool>,
        is_pinned -> Nullable<Bool>,
        is_used_in_article -> Nullable<Bool>,
        meta_data -> Nullable<Jsonb>,
    }
}

diesel::table! {
    use diesel::sql_types::*;
    use pgvector::sql_types::*;

    insight_topic (id) {
        id -> Uuid,
        organization_id -> Nullable<Uuid>,
        #[max_length = 255]
        name -> Varchar,
        description -> Nullable<Text>,
        keywords -> Nullable<Jsonb>,
        embedding -> Nullable<Vector>,
        is_auto_generated -> Nullable<Bool>,
        content_count -> Nullable<Int4>,
        last_insight_at -> Nullable<Timestamptz>,
        #[max_length = 20]
        color -> Nullable<Varchar>,
        #[max_length = 50]
        icon -> Nullable<Varchar>,
        created_at -> Nullable<Timestamptz>,
        updated_at -> Nullable<Timestamptz>,
    }
}

diesel::table! {
    use diesel::sql_types::*;
    use pgvector::sql_types::*;

    organization (id) {
        id -> Uuid,
        #[max_length = 255]
        name -> Varchar,
        #[max_length = 100]
        slug -> Varchar,
        bio -> Nullable<Text>,
        logo_url -> Nullable<Text>,
        website_url -> Nullable<Text>,
        #[max_length = 255]
        email_public -> Nullable<Varchar>,
        social_links -> Nullable<Jsonb>,
        meta_description -> Nullable<Text>,
        created_at -> Nullable<Timestamptz>,
        updated_at -> Nullable<Timestamptz>,
    }
}

diesel::table! {
    use diesel::sql_types::*;
    use pgvector::sql_types::*;

    page (id) {
        id -> Uuid,
        #[max_length = 50]
        slug -> Varchar,
        #[max_length = 500]
        title -> Varchar,
        content -> Nullable<Text>,
        description -> Nullable<Text>,
        image_url -> Nullable<Text>,
        meta_data -> Nullable<Jsonb>,
        is_published -> Nullable<Bool>,
        created_at -> Nullable<Timestamptz>,
        updated_at -> Nullable<Timestamptz>,
    }
}

diesel::table! {
    use diesel::sql_types::*;
    use pgvector::sql_types::*;

    project (id) {
        id -> Uuid,
        #[max_length = 500]
        title -> Varchar,
        description -> Text,
        image_url -> Nullable<Text>,
        url -> Nullable<Text>,
        created_at -> Nullable<Timestamptz>,
        updated_at -> Nullable<Timestamptz>,
        content -> Nullable<Text>,
        tag_ids -> Nullable<Array<Nullable<Int4>>>,
    }
}

diesel::table! {
    use diesel::sql_types::*;
    use pgvector::sql_types::*;

    site_settings (id) {
        id -> Int4,
        #[max_length = 20]
        public_profile_type -> Nullable<Varchar>,
        public_user_id -> Nullable<Uuid>,
        public_organization_id -> Nullable<Uuid>,
        created_at -> Nullable<Timestamptz>,
        updated_at -> Nullable<Timestamptz>,
    }
}

diesel::table! {
    use diesel::sql_types::*;
    use pgvector::sql_types::*;

    tag (id) {
        id -> Int4,
        #[max_length = 255]
        name -> Varchar,
        created_at -> Nullable<Timestamptz>,
    }
}

diesel::table! {
    use diesel::sql_types::*;
    use pgvector::sql_types::*;

    task_run (id) {
        id -> Uuid,
        #[max_length = 50]
        kind -> Varchar,
        #[max_length = 120]
        task_name -> Varchar,
        #[max_length = 50]
        status -> Varchar,
        organization_id -> Nullable<Uuid>,
        user_id -> Nullable<Uuid>,
        triggered_by_user_id -> Nullable<Uuid>,
        #[max_length = 50]
        trigger_source -> Varchar,
        parent_run_id -> Nullable<Uuid>,
        summary -> Nullable<Text>,
        error_summary -> Nullable<Text>,
        input_payload -> Nullable<Jsonb>,
        output_summary -> Nullable<Jsonb>,
        metrics -> Nullable<Jsonb>,
        started_at -> Nullable<Timestamptz>,
        completed_at -> Nullable<Timestamptz>,
        created_at -> Nullable<Timestamptz>,
        updated_at -> Nullable<Timestamptz>,
    }
}

diesel::table! {
    use diesel::sql_types::*;
    use pgvector::sql_types::*;

    task_run_event (id) {
        id -> Uuid,
        task_run_id -> Uuid,
        task_run_step_id -> Nullable<Uuid>,
        #[max_length = 120]
        event_type -> Varchar,
        #[max_length = 20]
        level -> Varchar,
        message -> Text,
        meta_data -> Nullable<Jsonb>,
        created_at -> Nullable<Timestamptz>,
    }
}

diesel::table! {
    use diesel::sql_types::*;
    use pgvector::sql_types::*;

    task_run_step (id) {
        id -> Uuid,
        task_run_id -> Uuid,
        #[max_length = 120]
        step_key -> Varchar,
        #[max_length = 255]
        step_name -> Varchar,
        #[max_length = 50]
        status -> Varchar,
        summary -> Nullable<Text>,
        error_summary -> Nullable<Text>,
        metrics -> Nullable<Jsonb>,
        started_at -> Nullable<Timestamptz>,
        completed_at -> Nullable<Timestamptz>,
        created_at -> Nullable<Timestamptz>,
        updated_at -> Nullable<Timestamptz>,
    }
}

diesel::table! {
    use diesel::sql_types::*;
    use pgvector::sql_types::*;

    user_insight_status (id) {
        id -> Uuid,
        user_id -> Uuid,
        insight_id -> Uuid,
        is_read -> Nullable<Bool>,
        is_pinned -> Nullable<Bool>,
        is_used_in_article -> Nullable<Bool>,
        read_at -> Nullable<Timestamptz>,
        created_at -> Nullable<Timestamptz>,
    }
}

diesel::joinable!(account -> organization (organization_id));
diesel::joinable!(article -> account (author_id));
diesel::joinable!(article_source -> article (article_id));
diesel::joinable!(article_version -> account (edited_by));
diesel::joinable!(chat_message -> article (article_id));
diesel::joinable!(content_topic_match -> crawled_content (content_id));
diesel::joinable!(content_topic_match -> insight_topic (topic_id));
diesel::joinable!(crawled_content -> data_source (data_source_id));
diesel::joinable!(data_source -> account (user_id));
diesel::joinable!(data_source -> organization (organization_id));
diesel::joinable!(imagen_request -> file_index (file_index_id));
diesel::joinable!(insight -> insight_topic (topic_id));
diesel::joinable!(insight -> organization (organization_id));
diesel::joinable!(insight_topic -> organization (organization_id));
diesel::joinable!(site_settings -> account (public_user_id));
diesel::joinable!(site_settings -> organization (public_organization_id));
diesel::joinable!(task_run -> organization (organization_id));
diesel::joinable!(task_run_event -> task_run (task_run_id));
diesel::joinable!(task_run_event -> task_run_step (task_run_step_id));
diesel::joinable!(task_run_step -> task_run (task_run_id));
diesel::joinable!(user_insight_status -> account (user_id));
diesel::joinable!(user_insight_status -> insight (insight_id));

diesel::allow_tables_to_appear_in_same_query!(
    account,
    article,
    article_source,
    article_version,
    chat_message,
    content_topic_match,
    crawled_content,
    data_source,
    file_index,
    imagen_request,
    insight,
    insight_topic,
    organization,
    page,
    project,
    site_settings,
    tag,
    task_run,
    task_run_event,
    task_run_step,
    user_insight_status,
);
