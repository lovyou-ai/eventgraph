use std::collections::HashMap;

use crate::errors::{EventGraphError, Result};
use crate::event::Event;
use crate::types::EventId;

pub struct ChainVerification {
    pub valid: bool,
    pub length: usize,
}

pub trait Store {
    fn append(&mut self, event: Event) -> Result<Event>;
    fn get(&self, event_id: &EventId) -> Result<&Event>;
    fn head(&self) -> Option<&Event>;
    fn count(&self) -> usize;
    fn verify_chain(&self) -> ChainVerification;
    fn close(&mut self);
}

pub struct InMemoryStore {
    events: Vec<Event>,
    index: HashMap<String, usize>,
}

impl InMemoryStore {
    pub fn new() -> Self {
        Self { events: Vec::new(), index: HashMap::new() }
    }

    pub fn recent(&self, limit: usize) -> Vec<&Event> {
        self.events.iter().rev().take(limit).collect()
    }
}

impl Default for InMemoryStore {
    fn default() -> Self { Self::new() }
}

impl Store for InMemoryStore {
    fn append(&mut self, event: Event) -> Result<Event> {
        if !self.events.is_empty() {
            let last = self.events.last().unwrap();
            if event.prev_hash.value() != last.hash.value() {
                return Err(EventGraphError::ChainIntegrity {
                    position: self.events.len(),
                    detail: format!(
                        "prev_hash {} != head hash {}",
                        event.prev_hash.value(),
                        last.hash.value()
                    ),
                });
            }
        }
        let id_str = event.id.value().to_string();
        self.events.push(event);
        let idx = self.events.len() - 1;
        self.index.insert(id_str, idx);
        Ok(self.events[idx].clone())
    }

    fn get(&self, event_id: &EventId) -> Result<&Event> {
        self.index
            .get(event_id.value())
            .map(|&i| &self.events[i])
            .ok_or_else(|| EventGraphError::EventNotFound {
                event_id: event_id.value().to_string(),
            })
    }

    fn head(&self) -> Option<&Event> {
        self.events.last()
    }

    fn count(&self) -> usize {
        self.events.len()
    }

    fn verify_chain(&self) -> ChainVerification {
        for i in 1..self.events.len() {
            if self.events[i - 1].hash.value() != self.events[i].prev_hash.value() {
                return ChainVerification { valid: false, length: i };
            }
        }
        ChainVerification { valid: true, length: self.events.len() }
    }

    fn close(&mut self) {}
}
