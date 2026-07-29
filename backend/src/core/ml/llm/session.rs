use std::{
    collections::HashMap,
    sync::{Arc, Mutex},
};

use async_trait::async_trait;
use chrono::{DateTime, Utc};
use thiserror::Error;
use uuid::Uuid;

use super::{ContentPart, LlmMessage, MessageRole};

#[derive(Debug, Error, Clone, PartialEq, Eq)]
pub enum StoreError {
    #[error("store lock poisoned")]
    LockPoisoned,
    #[error("session not found")]
    SessionNotFound,
    #[error("message not found")]
    MessageNotFound,
}

#[derive(Debug, Clone, PartialEq)]
pub struct Session {
    pub id: String,
    pub title: String,
    pub cost: f64,
    pub completion_tokens: i64,
    pub prompt_tokens: i64,
    pub summary_message_id: String,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[async_trait]
pub trait SessionStore: Send + Sync {
    async fn create_session(&self, title: &str) -> Result<Session, StoreError>;
    async fn get_session(&self, id: &str) -> Result<Session, StoreError>;
    async fn save_session(&self, session: Session) -> Result<Session, StoreError>;
    async fn create_message(
        &self,
        session_id: &str,
        role: MessageRole,
        parts: Vec<ContentPart>,
        model: &str,
    ) -> Result<LlmMessage, StoreError>;
    async fn update_message(&self, message: LlmMessage) -> Result<(), StoreError>;
    async fn list_messages(&self, session_id: &str) -> Result<Vec<LlmMessage>, StoreError>;
}

#[derive(Default)]
struct State {
    sessions: HashMap<String, Session>,
    messages: HashMap<String, LlmMessage>,
    session_messages: HashMap<String, Vec<String>>,
}

#[derive(Clone, Default)]
pub struct InMemorySessionStore {
    state: Arc<Mutex<State>>,
}

#[async_trait]
impl SessionStore for InMemorySessionStore {
    async fn create_session(&self, title: &str) -> Result<Session, StoreError> {
        let now = Utc::now();
        let session = Session {
            id: format!("session_{}", Uuid::new_v4()),
            title: title.to_owned(),
            cost: 0.0,
            completion_tokens: 0,
            prompt_tokens: 0,
            summary_message_id: String::new(),
            created_at: now,
            updated_at: now,
        };
        self.state
            .lock()
            .map_err(|_| StoreError::LockPoisoned)?
            .sessions
            .insert(session.id.clone(), session.clone());
        Ok(session)
    }

    async fn get_session(&self, id: &str) -> Result<Session, StoreError> {
        self.state
            .lock()
            .map_err(|_| StoreError::LockPoisoned)?
            .sessions
            .get(id)
            .cloned()
            .ok_or(StoreError::SessionNotFound)
    }

    async fn save_session(&self, mut session: Session) -> Result<Session, StoreError> {
        session.updated_at = Utc::now();
        self.state
            .lock()
            .map_err(|_| StoreError::LockPoisoned)?
            .sessions
            .insert(session.id.clone(), session.clone());
        Ok(session)
    }

    async fn create_message(
        &self,
        session_id: &str,
        role: MessageRole,
        parts: Vec<ContentPart>,
        model: &str,
    ) -> Result<LlmMessage, StoreError> {
        let mut state = self.state.lock().map_err(|_| StoreError::LockPoisoned)?;
        if !state.sessions.contains_key(session_id) {
            return Err(StoreError::SessionNotFound);
        }
        let message = LlmMessage::new(session_id, role, parts, model);
        state.messages.insert(message.id.clone(), message.clone());
        state
            .session_messages
            .entry(session_id.to_owned())
            .or_default()
            .push(message.id.clone());
        Ok(message)
    }

    async fn update_message(&self, mut message: LlmMessage) -> Result<(), StoreError> {
        let mut state = self.state.lock().map_err(|_| StoreError::LockPoisoned)?;
        if !state.messages.contains_key(&message.id) {
            return Err(StoreError::MessageNotFound);
        }
        message.updated_at = Utc::now();
        state.messages.insert(message.id.clone(), message);
        Ok(())
    }

    async fn list_messages(&self, session_id: &str) -> Result<Vec<LlmMessage>, StoreError> {
        let state = self.state.lock().map_err(|_| StoreError::LockPoisoned)?;
        let Some(ids) = state.session_messages.get(session_id) else {
            return Ok(Vec::new());
        };
        Ok(ids
            .iter()
            .filter_map(|id| state.messages.get(id).cloned())
            .collect())
    }
}
