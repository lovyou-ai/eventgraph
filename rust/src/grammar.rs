use std::collections::BTreeMap;

use serde_json::Value;

use crate::errors::{EventGraphError, Result};
use crate::event::{create_event, Event, Signer};
use crate::store::{InMemoryStore, Store};
use crate::types::{ActorId, ConversationId, EventId, EventType, Hash};

/// High-level social grammar operations that create properly hash-chained events.
///
/// Each operation gets the current chain head, creates an event with the
/// appropriate type and content, appends it to the store, and returns it.
pub struct Grammar<'a> {
    store: &'a mut InMemoryStore,
}

impl<'a> Grammar<'a> {
    pub fn new(store: &'a mut InMemoryStore) -> Self {
        Self { store }
    }

    /// Returns the prev_hash for the next event: head's hash or zero if empty.
    fn prev_hash(&self) -> Hash {
        self.store
            .head()
            .map(|e| e.hash.clone())
            .unwrap_or_else(Hash::zero)
    }

    /// Emit a new utterance with explicit causal links.
    ///
    /// Type: `grammar.emitted`
    /// Content: `{"Body": body}`
    /// Causes must be non-empty.
    pub fn emit(
        &mut self,
        source: ActorId,
        body: &str,
        conversation_id: ConversationId,
        causes: Vec<EventId>,
        signer: &dyn Signer,
    ) -> Result<Event> {
        if causes.is_empty() {
            return Err(EventGraphError::GrammarViolation {
                detail: "emit requires at least one cause".to_string(),
            });
        }

        let mut content = BTreeMap::new();
        content.insert("Body".to_string(), Value::String(body.to_string()));

        let prev_hash = self.prev_hash();
        let event = create_event(
            EventType::new("grammar.emitted")?,
            source,
            content,
            causes,
            conversation_id,
            prev_hash,
            signer,
            1,
        );
        self.store.append(event)
    }

    /// Respond to a specific event.
    ///
    /// Type: `grammar.responded`
    /// Content: `{"Body": body, "Parent": parent_id}`
    /// Causes: `[parent]`
    pub fn respond(
        &mut self,
        source: ActorId,
        body: &str,
        parent: EventId,
        conversation_id: ConversationId,
        signer: &dyn Signer,
    ) -> Result<Event> {
        let mut content = BTreeMap::new();
        content.insert("Body".to_string(), Value::String(body.to_string()));
        content.insert(
            "Parent".to_string(),
            Value::String(parent.value().to_string()),
        );

        let prev_hash = self.prev_hash();
        let event = create_event(
            EventType::new("grammar.responded")?,
            source,
            content,
            vec![parent],
            conversation_id,
            prev_hash,
            signer,
            1,
        );
        self.store.append(event)
    }

    /// Derive a new statement from an existing event.
    ///
    /// Type: `grammar.derived`
    /// Content: `{"Body": body, "Source": source_event_id}`
    /// Causes: `[source_event]`
    pub fn derive(
        &mut self,
        source: ActorId,
        body: &str,
        source_event: EventId,
        conversation_id: ConversationId,
        signer: &dyn Signer,
    ) -> Result<Event> {
        let mut content = BTreeMap::new();
        content.insert("Body".to_string(), Value::String(body.to_string()));
        content.insert(
            "Source".to_string(),
            Value::String(source_event.value().to_string()),
        );

        let prev_hash = self.prev_hash();
        let event = create_event(
            EventType::new("grammar.derived")?,
            source,
            content,
            vec![source_event],
            conversation_id,
            prev_hash,
            signer,
            1,
        );
        self.store.append(event)
    }

    /// Extend a previous statement with additional content.
    ///
    /// Type: `grammar.extended`
    /// Content: `{"Body": body, "Previous": previous_id}`
    /// Causes: `[previous]`
    pub fn extend(
        &mut self,
        source: ActorId,
        body: &str,
        previous: EventId,
        conversation_id: ConversationId,
        signer: &dyn Signer,
    ) -> Result<Event> {
        let mut content = BTreeMap::new();
        content.insert("Body".to_string(), Value::String(body.to_string()));
        content.insert(
            "Previous".to_string(),
            Value::String(previous.value().to_string()),
        );

        let prev_hash = self.prev_hash();
        let event = create_event(
            EventType::new("grammar.extended")?,
            source,
            content,
            vec![previous],
            conversation_id,
            prev_hash,
            signer,
            1,
        );
        self.store.append(event)
    }

    /// Retract a previously authored event.
    ///
    /// Type: `grammar.retracted`
    /// Content: `{"Target": target_id, "Reason": reason}`
    /// Causes: `[target]`
    ///
    /// The source must be the author of the target event.
    pub fn retract(
        &mut self,
        source: ActorId,
        target: EventId,
        reason: &str,
        conversation_id: ConversationId,
        signer: &dyn Signer,
    ) -> Result<Event> {
        // Verify source authored the target event
        let target_event = self.store.get(&target)?;
        if target_event.source.value() != source.value() {
            return Err(EventGraphError::GrammarViolation {
                detail: format!(
                    "actor {} cannot retract event authored by {}",
                    source.value(),
                    target_event.source.value()
                ),
            });
        }

        let mut content = BTreeMap::new();
        content.insert(
            "Target".to_string(),
            Value::String(target.value().to_string()),
        );
        content.insert("Reason".to_string(), Value::String(reason.to_string()));

        let prev_hash = self.prev_hash();
        let event = create_event(
            EventType::new("grammar.retracted")?,
            source,
            content,
            vec![target],
            conversation_id,
            prev_hash,
            signer,
            1,
        );
        self.store.append(event)
    }

    /// Annotate an existing event with a key-value pair.
    ///
    /// Type: `grammar.annotated`
    /// Content: `{"Target": target_id, "Key": key, "Value": value}`
    /// Causes: `[target]`
    pub fn annotate(
        &mut self,
        source: ActorId,
        target: EventId,
        key: &str,
        value: &str,
        conversation_id: ConversationId,
        signer: &dyn Signer,
    ) -> Result<Event> {
        let mut content = BTreeMap::new();
        content.insert(
            "Target".to_string(),
            Value::String(target.value().to_string()),
        );
        content.insert("Key".to_string(), Value::String(key.to_string()));
        content.insert("Value".to_string(), Value::String(value.to_string()));

        let prev_hash = self.prev_hash();
        let event = create_event(
            EventType::new("grammar.annotated")?,
            source,
            content,
            vec![target],
            conversation_id,
            prev_hash,
            signer,
            1,
        );
        self.store.append(event)
    }

    /// Merge multiple events into a single synthesised statement.
    ///
    /// Type: `grammar.merged`
    /// Content: `{"Body": body, "Sources": [source_ids...]}`
    /// Causes: sources (must be >= 2)
    pub fn merge(
        &mut self,
        source: ActorId,
        body: &str,
        sources: Vec<EventId>,
        conversation_id: ConversationId,
        signer: &dyn Signer,
    ) -> Result<Event> {
        if sources.len() < 2 {
            return Err(EventGraphError::GrammarViolation {
                detail: "merge requires at least two source events".to_string(),
            });
        }

        let source_values: Vec<Value> = sources
            .iter()
            .map(|s| Value::String(s.value().to_string()))
            .collect();

        let mut content = BTreeMap::new();
        content.insert("Body".to_string(), Value::String(body.to_string()));
        content.insert("Sources".to_string(), Value::Array(source_values));

        let prev_hash = self.prev_hash();
        let event = create_event(
            EventType::new("grammar.merged")?,
            source,
            content,
            sources,
            conversation_id,
            prev_hash,
            signer,
            1,
        );
        self.store.append(event)
    }
}
