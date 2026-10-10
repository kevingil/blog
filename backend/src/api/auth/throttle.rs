use std::{
    collections::HashMap,
    sync::{Mutex, MutexGuard},
    time::{Duration, Instant},
};

const WINDOW: Duration = Duration::from_secs(15 * 60);
const MAX_FAILURES: u32 = 10;
const MAX_TRACKED_EMAILS: usize = 10_000;

/// Counts failed sign-ins per email so a password cannot be guessed faster
/// than `MAX_FAILURES` per `WINDOW`. Process-local: the API runs as a single
/// instance, and a restart only resets the counters.
pub struct LoginThrottle {
    window: Duration,
    max_failures: u32,
    failures: Mutex<HashMap<String, Failures>>,
}

struct Failures {
    count: u32,
    window_started: Instant,
}

impl Default for LoginThrottle {
    fn default() -> Self {
        Self::new(WINDOW, MAX_FAILURES)
    }
}

impl LoginThrottle {
    pub fn new(window: Duration, max_failures: u32) -> Self {
        Self {
            window,
            max_failures,
            failures: Mutex::new(HashMap::new()),
        }
    }

    /// How long until `email` may try again, or `None` when it may try now.
    pub fn retry_after(&self, email: &str) -> Option<Duration> {
        let failures = self.failures();
        let entry = failures.get(&key(email))?;
        if entry.count < self.max_failures {
            return None;
        }
        self.window
            .checked_sub(entry.window_started.elapsed())
            .filter(|wait| !wait.is_zero())
    }

    pub fn record_failure(&self, email: &str) {
        let mut failures = self.failures();
        if failures.len() >= MAX_TRACKED_EMAILS {
            let window = self.window;
            failures.retain(|_, entry| entry.window_started.elapsed() < window);
        }
        let key = key(email);
        if !failures.contains_key(&key) && failures.len() >= MAX_TRACKED_EMAILS {
            return;
        }
        let entry = failures.entry(key).or_insert(Failures {
            count: 0,
            window_started: Instant::now(),
        });
        if entry.window_started.elapsed() >= self.window {
            entry.count = 0;
            entry.window_started = Instant::now();
        }
        entry.count = entry.count.saturating_add(1);
    }

    pub fn record_success(&self, email: &str) {
        self.failures().remove(&key(email));
    }

    fn failures(&self) -> MutexGuard<'_, HashMap<String, Failures>> {
        self.failures
            .lock()
            .unwrap_or_else(std::sync::PoisonError::into_inner)
    }
}

fn key(email: &str) -> String {
    email.trim().to_lowercase()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn locks_an_email_after_too_many_failures_and_success_clears_it() {
        let throttle = LoginThrottle::new(Duration::from_secs(60), 3);
        for _ in 0..2 {
            throttle.record_failure("Owner@Example.test");
            assert_eq!(throttle.retry_after("owner@example.test"), None);
        }
        throttle.record_failure("owner@example.test ");
        let wait = throttle.retry_after("OWNER@example.test");
        assert!(wait.is_some_and(|wait| wait <= Duration::from_secs(60)));
        assert_eq!(throttle.retry_after("someone-else@example.test"), None);

        throttle.record_success("owner@example.test");
        assert_eq!(throttle.retry_after("owner@example.test"), None);
    }

    #[test]
    fn failures_older_than_the_window_stop_counting() {
        let throttle = LoginThrottle::new(Duration::from_millis(20), 1);
        throttle.record_failure("owner@example.test");
        assert!(throttle.retry_after("owner@example.test").is_some());
        std::thread::sleep(Duration::from_millis(30));
        assert_eq!(throttle.retry_after("owner@example.test"), None);
        throttle.record_failure("owner@example.test");
        assert!(throttle.retry_after("owner@example.test").is_some());
    }
}
