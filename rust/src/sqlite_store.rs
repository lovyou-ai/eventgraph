//! SQLite-backed event store using rusqlite.
//!
//! Optional — only used when the `sqlite` feature is enabled.
//! Satisfies the Store trait with persistent storage.

use std::collections::{HashMap, HashSet};
use std::sync::Mutex;

use rusqlite::{params, Connection};

use crate::errors::{EventGraphError, Result};
use crate::event::Event;
use crate::store::{ChainVerification, Store};
use crate::types::*;

const SCHEMA: &str = "
CREATE TABLE IF NOT EXISTS events (
    position        INTEGER PRIMARY KEY AUTOINCREMENT,
    event_id        TEXT NOT NULL UNIQUE,
    event_type      TEXT NOT NULL,
    version         INTEGER NOT NULL,
    timestamp_nanos INTEGER NOT NULL,
    source          TEXT NOT NULL,
    content         TEXT NOT NULL,
    causes          TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    hash            TEXT NOT NULL,
    prev_hash       TEXT NOT NULL,
    signature       BLOB NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_events_type ON events(event_type);
CREATE INDEX IF NOT EXISTS idx_events_source ON events(source);
CREATE INDEX IF NOT EXISTS idx_events_conversation ON events(conversation_id);
";

fn row_to_event(row: &rusqlite::Row<'_>) -> rusqlite::Result<Event> {
    let event_id: String = row.get(1)?;
    let event_type: String = row.get(2)?;
    let version: i32 = row.get(3)?;
    let timestamp_nanos: i64 = row.get(4)?;
    let source: String = row.get(5)?;
    let content_json: String = row.get(6)?;
    let causes_json: String = row.get(7)?;
    let conversation_id: String = row.get(8)?;
    let hash: String = row.get(9)?;
    let prev_hash: String = row.get(10)?;
    let signature: Vec<u8> = row.get(11)?;

    let causes: Vec<String> = serde_json::from_str(&causes_json).unwrap_or_default();
    let cause_ids: Vec<EventId> = causes.into_iter().map(|c| EventId::new(&c).unwrap()).collect();
    let content: std::collections::BTreeMap<String, serde_json::Value> =
        serde_json::from_str(&content_json).unwrap_or_default();

    Ok(Event {
        version: version as u32,
        id: EventId::new(&event_id).unwrap(),
        event_type: EventType::new(&event_type).unwrap(),
        timestamp_nanos: timestamp_nanos as u64,
        source: ActorId::new(&source).unwrap(),
        content,
        causes: crate::types::NonEmpty::of(cause_ids).unwrap(),
        conversation_id: ConversationId::new(&conversation_id).unwrap(),
        hash: Hash::new(&hash).unwrap(),
        prev_hash: Hash::new(&prev_hash).unwrap(),
        signature: Signature::new(signature).unwrap(),
    })
}

/// SQLite-backed event store. Thread-safe via Mutex.
pub struct SqliteStore {
    conn: Mutex<Connection>,
}

impl SqliteStore {
    /// Opens a SQLite store at the given path. Use ":memory:" for in-memory.
    pub fn open(path: &str) -> Result<Self> {
        let conn = Connection::open(path).map_err(|e| EventGraphError::StoreUnavailable {
            detail: e.to_string(),
        })?;
        conn.execute_batch("PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;")
            .map_err(|e| EventGraphError::StoreUnavailable {
                detail: e.to_string(),
            })?;
        conn.execute_batch(SCHEMA)
            .map_err(|e| EventGraphError::StoreUnavailable {
                detail: e.to_string(),
            })?;
        Ok(Self {
            conn: Mutex::new(conn),
        })
    }

    /// Opens an in-memory SQLite store.
    pub fn in_memory() -> Result<Self> {
        Self::open(":memory:")
    }

    fn query_events(conn: &Connection, sql: &str, params: &[&dyn rusqlite::ToSql]) -> Vec<Event> {
        let mut stmt = conn.prepare(sql).unwrap();
        let events = stmt
            .query_map(params, row_to_event)
            .unwrap()
            .filter_map(|r| r.ok())
            .collect();
        events
    }

    fn query_single(
        conn: &Connection,
        sql: &str,
        params: &[&dyn rusqlite::ToSql],
    ) -> Option<Event> {
        let mut stmt = conn.prepare(sql).unwrap();
        stmt.query_row(params, row_to_event).ok()
    }

    pub fn recent(&self, limit: usize) -> Vec<Event> {
        let conn = self.conn.lock().unwrap();
        Self::query_events(
            &conn,
            "SELECT * FROM events ORDER BY position DESC LIMIT ?1",
            &[&(limit as i64)],
        )
    }

    pub fn by_type(&self, event_type: &EventType, limit: usize) -> Vec<Event> {
        let conn = self.conn.lock().unwrap();
        Self::query_events(
            &conn,
            "SELECT * FROM events WHERE event_type = ?1 ORDER BY position DESC LIMIT ?2",
            &[&event_type.value(), &(limit as i64)],
        )
    }

    pub fn by_source(&self, source: &ActorId, limit: usize) -> Vec<Event> {
        let conn = self.conn.lock().unwrap();
        Self::query_events(
            &conn,
            "SELECT * FROM events WHERE source = ?1 ORDER BY position DESC LIMIT ?2",
            &[&source.value(), &(limit as i64)],
        )
    }

    pub fn by_conversation(&self, conversation_id: &ConversationId, limit: usize) -> Vec<Event> {
        let conn = self.conn.lock().unwrap();
        Self::query_events(
            &conn,
            "SELECT * FROM events WHERE conversation_id = ?1 ORDER BY position DESC LIMIT ?2",
            &[&conversation_id.value(), &(limit as i64)],
        )
    }

    pub fn ancestors(&self, event_id: &EventId, max_depth: usize) -> Result<Vec<Event>> {
        let conn = self.conn.lock().unwrap();
        let start = Self::query_single(
            &conn,
            "SELECT * FROM events WHERE event_id = ?1",
            &[&event_id.value()],
        )
        .ok_or_else(|| EventGraphError::EventNotFound {
            event_id: event_id.value().to_string(),
        })?;

        let mut result = Vec::new();
        let mut visited = HashSet::new();
        visited.insert(event_id.value().to_string());
        let mut frontier: Vec<String> = start
            .causes
            .iter()
            .filter(|c| c.value() != event_id.value())
            .map(|c| c.value().to_string())
            .collect();

        for _ in 0..max_depth {
            if frontier.is_empty() {
                break;
            }
            let mut next = Vec::new();
            for eid in &frontier {
                if !visited.insert(eid.clone()) {
                    continue;
                }
                if let Some(ev) = Self::query_single(
                    &conn,
                    "SELECT * FROM events WHERE event_id = ?1",
                    &[eid as &dyn rusqlite::ToSql],
                ) {
                    for c in ev.causes.iter() {
                        if !visited.contains(c.value()) {
                            next.push(c.value().to_string());
                        }
                    }
                    result.push(ev);
                }
            }
            frontier = next;
        }
        Ok(result)
    }

    pub fn descendants(&self, event_id: &EventId, max_depth: usize) -> Result<Vec<Event>> {
        let conn = self.conn.lock().unwrap();
        Self::query_single(
            &conn,
            "SELECT * FROM events WHERE event_id = ?1",
            &[&event_id.value()],
        )
        .ok_or_else(|| EventGraphError::EventNotFound {
            event_id: event_id.value().to_string(),
        })?;

        // Build reverse index
        let all_events = Self::query_events(&conn, "SELECT * FROM events ORDER BY position ASC", &[]);
        let mut children: HashMap<String, Vec<String>> = HashMap::new();
        for ev in &all_events {
            for c in ev.causes.iter() {
                if c.value() != ev.id.value() {
                    children
                        .entry(c.value().to_string())
                        .or_default()
                        .push(ev.id.value().to_string());
                }
            }
        }

        let mut result = Vec::new();
        let mut visited = HashSet::new();
        visited.insert(event_id.value().to_string());
        let mut frontier = children
            .get(event_id.value())
            .cloned()
            .unwrap_or_default();

        for _ in 0..max_depth {
            if frontier.is_empty() {
                break;
            }
            let mut next = Vec::new();
            for eid in &frontier {
                if !visited.insert(eid.clone()) {
                    continue;
                }
                if let Some(ev) = Self::query_single(
                    &conn,
                    "SELECT * FROM events WHERE event_id = ?1",
                    &[eid as &dyn rusqlite::ToSql],
                ) {
                    result.push(ev);
                    if let Some(ch) = children.get(eid) {
                        for child in ch {
                            if !visited.contains(child.as_str()) {
                                next.push(child.clone());
                            }
                        }
                    }
                }
            }
            frontier = next;
        }
        Ok(result)
    }
}

impl Store for SqliteStore {
    fn append(&mut self, event: Event) -> Result<Event> {
        let conn = self.conn.lock().unwrap();

        // Idempotency
        if let Some(existing) = Self::query_single(
            &conn,
            "SELECT * FROM events WHERE event_id = ?1",
            &[&event.id.value()],
        ) {
            if existing.hash.value() != event.hash.value() {
                return Err(EventGraphError::ChainIntegrity {
                    position: 0,
                    detail: format!("hash mismatch for existing event {}", event.id.value()),
                });
            }
            return Ok(existing);
        }

        // Chain continuity
        if let Some(head) = Self::query_single(
            &conn,
            "SELECT * FROM events ORDER BY position DESC LIMIT 1",
            &[],
        ) {
            if event.prev_hash.value() != head.hash.value() {
                return Err(EventGraphError::ChainIntegrity {
                    position: 0,
                    detail: format!(
                        "prev_hash {} != head hash {}",
                        event.prev_hash.value(),
                        head.hash.value()
                    ),
                });
            }
        }

        let causes_json =
            serde_json::to_string(&event.causes.iter().map(|c| c.value()).collect::<Vec<_>>())
                .unwrap();
        let content_json = serde_json::to_string(&event.content()).unwrap();

        conn.execute(
            "INSERT INTO events (event_id, event_type, version, timestamp_nanos, source, content, causes, conversation_id, hash, prev_hash, signature) VALUES (?1,?2,?3,?4,?5,?6,?7,?8,?9,?10,?11)",
            params![
                event.id.value(),
                event.event_type.value(),
                event.version as i32,
                event.timestamp_nanos as i64,
                event.source.value(),
                content_json,
                causes_json,
                event.conversation_id.value(),
                event.hash.value(),
                event.prev_hash.value(),
                event.signature.bytes().to_vec(),
            ],
        ).map_err(|e| EventGraphError::StoreUnavailable { detail: e.to_string() })?;

        Ok(event)
    }

    fn get(&self, event_id: &EventId) -> Result<&Event> {
        // SQLiteStore can't return references — this is a trait design issue.
        // For now, we can't implement the trait directly as it requires &Event returns.
        // Users should use the inherent methods or we'd need to change the trait.
        unimplemented!("SqliteStore cannot return references; use query methods directly")
    }

    fn head(&self) -> Option<&Event> {
        unimplemented!("SqliteStore cannot return references; use head_owned() instead")
    }

    fn count(&self) -> usize {
        let conn = self.conn.lock().unwrap();
        let count: i64 = conn
            .query_row("SELECT COUNT(*) FROM events", [], |row| row.get(0))
            .unwrap_or(0);
        count as usize
    }

    fn verify_chain(&self) -> ChainVerification {
        let conn = self.conn.lock().unwrap();
        let events = Self::query_events(&conn, "SELECT * FROM events ORDER BY position ASC", &[]);
        for i in 1..events.len() {
            if events[i - 1].hash.value() != events[i].prev_hash.value() {
                return ChainVerification {
                    valid: false,
                    length: i,
                };
            }
        }
        ChainVerification {
            valid: true,
            length: events.len(),
        }
    }

    fn close(&mut self) {
        // Connection dropped when SqliteStore is dropped
    }
}

impl SqliteStore {
    /// Get an event by ID, returning an owned Event.
    pub fn get_owned(&self, event_id: &EventId) -> Result<Event> {
        let conn = self.conn.lock().unwrap();
        Self::query_single(
            &conn,
            "SELECT * FROM events WHERE event_id = ?1",
            &[&event_id.value()],
        )
        .ok_or_else(|| EventGraphError::EventNotFound {
            event_id: event_id.value().to_string(),
        })
    }

    /// Get the most recent event as an owned value.
    pub fn head_owned(&self) -> Option<Event> {
        let conn = self.conn.lock().unwrap();
        Self::query_single(
            &conn,
            "SELECT * FROM events ORDER BY position DESC LIMIT 1",
            &[],
        )
    }
}
