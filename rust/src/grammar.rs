use std::collections::BTreeMap;

use serde_json::Value;

use crate::errors::{EventGraphError, Result};
use crate::event::{create_event, Event, Signer};
use crate::store::{InMemoryStore, Store};
use crate::types::{ActorId, ConversationId, DomainScope, EventId, EventType, Hash, Weight};

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

    // ── Edge operations ──────────────────────────────────────────────────

    /// Helper to build edge content as a BTreeMap.
    fn edge_content(
        from: &ActorId,
        to: &ActorId,
        edge_type: &str,
        weight: f64,
        direction: &str,
        scope: Option<&DomainScope>,
    ) -> BTreeMap<String, Value> {
        let mut content = BTreeMap::new();
        content.insert("Direction".to_string(), Value::String(direction.to_string()));
        content.insert("EdgeType".to_string(), Value::String(edge_type.to_string()));
        content.insert("From".to_string(), Value::String(from.value().to_string()));
        content.insert(
            "Scope".to_string(),
            match scope {
                Some(s) => Value::String(s.value().to_string()),
                None => Value::Null,
            },
        );
        content.insert("To".to_string(), Value::String(to.value().to_string()));
        content.insert(
            "Weight".to_string(),
            Value::Number(serde_json::Number::from_f64(weight).unwrap()),
        );
        content
    }

    /// Acknowledge creates a content-free centripetal edge toward a vertex. (Operation 7)
    ///
    /// Type: `edge.created`
    /// Causes: `[target]`
    pub fn acknowledge(
        &mut self,
        source: ActorId,
        target: EventId,
        target_actor: ActorId,
        conversation_id: ConversationId,
        signer: &dyn Signer,
    ) -> Result<Event> {
        let content = Self::edge_content(
            &source, &target_actor, "Acknowledgement", 0.0, "Centripetal", None,
        );
        let prev_hash = self.prev_hash();
        let event = create_event(
            EventType::new("edge.created")?,
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

    /// Propagate redistributes a vertex into the actor's subgraph. (Operation 8)
    ///
    /// Type: `edge.created`
    /// Causes: `[target]`
    pub fn propagate(
        &mut self,
        source: ActorId,
        target: EventId,
        target_actor: ActorId,
        conversation_id: ConversationId,
        signer: &dyn Signer,
    ) -> Result<Event> {
        let content = Self::edge_content(
            &source, &target_actor, "Reference", 0.0, "Centrifugal", None,
        );
        let prev_hash = self.prev_hash();
        let event = create_event(
            EventType::new("edge.created")?,
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

    /// Endorse creates a reputation-staked edge toward a vertex. (Operation 9)
    ///
    /// Type: `edge.created`
    /// Causes: `[target]`
    pub fn endorse(
        &mut self,
        source: ActorId,
        target: EventId,
        target_actor: ActorId,
        weight: Weight,
        scope: Option<&DomainScope>,
        conversation_id: ConversationId,
        signer: &dyn Signer,
    ) -> Result<Event> {
        let content = Self::edge_content(
            &source, &target_actor, "Endorsement", weight.value(), "Centripetal", scope,
        );
        let prev_hash = self.prev_hash();
        let event = create_event(
            EventType::new("edge.created")?,
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

    /// Subscribe creates a persistent, future-oriented edge to an actor. (Operation 10)
    ///
    /// Type: `edge.created`
    /// Causes: `[cause]`
    pub fn subscribe(
        &mut self,
        source: ActorId,
        target: ActorId,
        scope: Option<&DomainScope>,
        cause: EventId,
        conversation_id: ConversationId,
        signer: &dyn Signer,
    ) -> Result<Event> {
        let content = Self::edge_content(
            &source, &target, "Subscription", 0.0, "Centripetal", scope,
        );
        let prev_hash = self.prev_hash();
        let event = create_event(
            EventType::new("edge.created")?,
            source,
            content,
            vec![cause],
            conversation_id,
            prev_hash,
            signer,
            1,
        );
        self.store.append(event)
    }

    /// Channel creates a private, bidirectional, content-bearing edge. (Operation 11)
    ///
    /// Type: `edge.created`
    /// Causes: `[cause]`
    pub fn channel(
        &mut self,
        source: ActorId,
        target: ActorId,
        scope: Option<&DomainScope>,
        cause: EventId,
        conversation_id: ConversationId,
        signer: &dyn Signer,
    ) -> Result<Event> {
        let content = Self::edge_content(
            &source, &target, "Channel", 0.0, "Centripetal", scope,
        );
        let prev_hash = self.prev_hash();
        let event = create_event(
            EventType::new("edge.created")?,
            source,
            content,
            vec![cause],
            conversation_id,
            prev_hash,
            signer,
            1,
        );
        self.store.append(event)
    }

    /// Delegate grants authority for another actor to operate as you. (Operation 12)
    ///
    /// Type: `edge.created`
    /// Causes: `[cause]`
    pub fn delegate(
        &mut self,
        source: ActorId,
        target: ActorId,
        scope: &DomainScope,
        weight: Weight,
        cause: EventId,
        conversation_id: ConversationId,
        signer: &dyn Signer,
    ) -> Result<Event> {
        let content = Self::edge_content(
            &source, &target, "Delegation", weight.value(), "Centrifugal", Some(scope),
        );
        let prev_hash = self.prev_hash();
        let event = create_event(
            EventType::new("edge.created")?,
            source,
            content,
            vec![cause],
            conversation_id,
            prev_hash,
            signer,
            1,
        );
        self.store.append(event)
    }

    /// Consent records a consent proposal. (Operation 13)
    ///
    /// Type: `grammar.consented`
    /// Content: `{"Agreement": agreement, "Parties": [sorted parties], "Scope": scope}`
    /// Causes: `[cause]`
    ///
    /// LIMITATION: This is currently single-signed (party_a only). A full dual-consent
    /// protocol requires a two-phase flow.
    pub fn consent(
        &mut self,
        party_a: ActorId,
        party_b: ActorId,
        agreement: &str,
        scope: &DomainScope,
        cause: EventId,
        conversation_id: ConversationId,
        signer: &dyn Signer,
    ) -> Result<Event> {
        // Sort parties lexicographically for deterministic hashing
        let (first, second) = if party_a.value() <= party_b.value() {
            (party_a.value().to_string(), party_b.value().to_string())
        } else {
            (party_b.value().to_string(), party_a.value().to_string())
        };

        let mut content = BTreeMap::new();
        content.insert("Agreement".to_string(), Value::String(agreement.to_string()));
        content.insert(
            "Parties".to_string(),
            Value::Array(vec![
                Value::String(first),
                Value::String(second),
            ]),
        );
        content.insert("Scope".to_string(), Value::String(scope.value().to_string()));

        let prev_hash = self.prev_hash();
        let event = create_event(
            EventType::new("grammar.consented")?,
            party_a,
            content,
            vec![cause],
            conversation_id,
            prev_hash,
            signer,
            1,
        );
        self.store.append(event)
    }

    /// Sever removes a subscription, channel, or delegation via edge supersession. (Operation 14)
    ///
    /// Type: `edge.superseded`
    /// Content: `{"NewEdge": null, "PreviousEdge": edge_id, "Reason": cause_id}`
    ///
    /// Only a party to the edge (From or To) can sever it.
    /// Only subscriptions, channels, and delegations are severable.
    pub fn sever(
        &mut self,
        source: ActorId,
        previous_edge_id: EventId,
        cause: EventId,
        conversation_id: ConversationId,
        signer: &dyn Signer,
    ) -> Result<Event> {
        if cause.value().is_empty() {
            return Err(EventGraphError::GrammarViolation {
                detail: "sever: cause must not be empty".to_string(),
            });
        }

        // Verify the edge event exists and check permissions
        let edge_event = self.store.get(&previous_edge_id)?;
        let edge_content = edge_event.content();

        // Verify this is an edge.created event
        if edge_event.event_type.value() != "edge.created" {
            return Err(EventGraphError::GrammarViolation {
                detail: format!(
                    "sever: event {} is not an edge.created event",
                    previous_edge_id.value()
                ),
            });
        }

        // Check edge type is severable
        let edge_type = edge_content
            .get("EdgeType")
            .and_then(|v| v.as_str())
            .unwrap_or("");
        match edge_type {
            "Subscription" | "Channel" | "Delegation" => {}
            _ => {
                return Err(EventGraphError::GrammarViolation {
                    detail: format!(
                        "sever: edge type {} is not severable (only Subscription, Channel, Delegation)",
                        edge_type
                    ),
                });
            }
        }

        // Check source is a party to the edge
        let from = edge_content
            .get("From")
            .and_then(|v| v.as_str())
            .unwrap_or("");
        let to = edge_content
            .get("To")
            .and_then(|v| v.as_str())
            .unwrap_or("");
        if from != source.value() && to != source.value() {
            return Err(EventGraphError::GrammarViolation {
                detail: format!(
                    "sever: actor {} is not a party to edge {} (from={}, to={})",
                    source.value(),
                    previous_edge_id.value(),
                    from,
                    to
                ),
            });
        }

        // Build causes: edge event + trigger cause (if different)
        let mut causes = vec![previous_edge_id.clone()];
        if cause.value() != previous_edge_id.value() {
            causes.push(cause.clone());
        }

        let mut content = BTreeMap::new();
        content.insert("NewEdge".to_string(), Value::Null);
        content.insert(
            "PreviousEdge".to_string(),
            Value::String(previous_edge_id.value().to_string()),
        );
        content.insert(
            "Reason".to_string(),
            Value::String(cause.value().to_string()),
        );

        let prev_hash = self.prev_hash();
        let event = create_event(
            EventType::new("edge.superseded")?,
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

    // ── Named functions (compositions of base operations) ────────────────

    /// Challenge is Respond + dispute flag: formal dispute that follows content.
    ///
    /// Returns (response_event, dispute_flag_event).
    pub fn challenge(
        &mut self,
        source: ActorId,
        body: &str,
        target: EventId,
        conversation_id: ConversationId,
        signer: &dyn Signer,
    ) -> Result<(Event, Event)> {
        let response = self.respond(
            source.clone(),
            body,
            target,
            conversation_id.clone(),
            signer,
        )?;
        let flag = self.annotate(
            source,
            response.id.clone(),
            "dispute",
            "challenged",
            conversation_id,
            signer,
        )?;
        Ok((response, flag))
    }

    /// Recommend is Propagate + Channel: directed sharing to a specific person.
    ///
    /// Returns (propagate_event, channel_event).
    pub fn recommend(
        &mut self,
        source: ActorId,
        target: EventId,
        target_actor: ActorId,
        conversation_id: ConversationId,
        signer: &dyn Signer,
    ) -> Result<(Event, Event)> {
        let propagate_ev = self.propagate(
            source.clone(),
            target,
            target_actor.clone(),
            conversation_id.clone(),
            signer,
        )?;
        let channel_ev = self.channel(
            source,
            target_actor,
            None,
            propagate_ev.id.clone(),
            conversation_id,
            signer,
        )?;
        Ok((propagate_ev, channel_ev))
    }

    /// Invite is Endorse + Subscribe: trust-staked introduction of a new actor.
    ///
    /// Returns (endorse_event, subscribe_event).
    pub fn invite(
        &mut self,
        source: ActorId,
        target: ActorId,
        weight: Weight,
        scope: Option<&DomainScope>,
        cause: EventId,
        conversation_id: ConversationId,
        signer: &dyn Signer,
    ) -> Result<(Event, Event)> {
        let endorse_ev = self.endorse(
            source.clone(),
            cause,
            target.clone(),
            weight,
            scope,
            conversation_id.clone(),
            signer,
        )?;
        let subscribe_ev = self.subscribe(
            source,
            target,
            scope,
            endorse_ev.id.clone(),
            conversation_id,
            signer,
        )?;
        Ok((endorse_ev, subscribe_ev))
    }

    /// Forgive is Subscribe after Sever: reconciliation with history intact.
    pub fn forgive(
        &mut self,
        source: ActorId,
        sever_event: EventId,
        target: ActorId,
        scope: Option<&DomainScope>,
        conversation_id: ConversationId,
        signer: &dyn Signer,
    ) -> Result<Event> {
        self.subscribe(source, target, scope, sever_event, conversation_id, signer)
    }
}
