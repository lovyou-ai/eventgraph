use sha2::{Sha256, Digest};
use serde_json::Value;
use std::collections::BTreeMap;
use uuid::Uuid;

use crate::types::*;

// ── Signer trait ───────────────────────────────────────────────────────

pub trait Signer {
    fn sign(&self, data: &[u8]) -> Signature;
}

pub struct NoopSigner;

impl Signer for NoopSigner {
    fn sign(&self, _data: &[u8]) -> Signature { Signature::zero() }
}

// ── Canonical form ─────────────────────────────────────────────────────

pub fn canonical_content_json(content: &BTreeMap<String, Value>) -> String {
    // Omit null values, match Go's number formatting (1.0 → 1)
    let filtered: BTreeMap<&String, &Value> = content
        .iter()
        .filter(|(_, v)| !v.is_null())
        .collect();
    let mut s = String::from("{");
    let mut first = true;
    for (k, v) in &filtered {
        if !first { s.push(','); }
        first = false;
        s.push('"');
        s.push_str(k);
        s.push_str("\":");
        canonical_value_to_string(v, &mut s);
    }
    s.push('}');
    s
}

fn canonical_value_to_string(v: &Value, s: &mut String) {
    match v {
        Value::Object(map) => {
            // Recursively sort keys and omit nulls in nested objects
            let filtered: BTreeMap<&String, &Value> = map
                .iter()
                .filter(|(_, v)| !v.is_null())
                .collect();
            s.push('{');
            let mut first = true;
            for (k, val) in &filtered {
                if !first { s.push(','); }
                first = false;
                s.push('"');
                s.push_str(k);
                s.push_str("\":");
                canonical_value_to_string(val, s);
            }
            s.push('}');
        }
        Value::Number(n) => {
            if let Some(f) = n.as_f64() {
                if f == (f as i64) as f64 && f.is_finite() {
                    s.push_str(&(f as i64).to_string());
                } else {
                    s.push_str(&serde_json::to_string(v).unwrap());
                }
            } else {
                s.push_str(&serde_json::to_string(v).unwrap());
            }
        }
        Value::Array(arr) => {
            s.push('[');
            for (i, item) in arr.iter().enumerate() {
                if i > 0 { s.push(','); }
                canonical_value_to_string(item, s);
            }
            s.push(']');
        }
        _ => s.push_str(&serde_json::to_string(v).unwrap()),
    }
}

pub fn canonical_form(
    version: u32,
    prev_hash: &str,
    causes: &[&str],
    event_id: &str,
    event_type: &str,
    source: &str,
    conversation_id: &str,
    timestamp_nanos: u64,
    content_json: &str,
) -> String {
    let mut sorted: Vec<&str> = causes.to_vec();
    sorted.sort();
    let causes_str = sorted.join(",");
    format!("{version}|{prev_hash}|{causes_str}|{event_id}|{event_type}|{source}|{conversation_id}|{timestamp_nanos}|{content_json}")
}

pub fn compute_hash(canonical: &str) -> Hash {
    let mut hasher = Sha256::new();
    hasher.update(canonical.as_bytes());
    let result = hasher.finalize();
    let hex: String = result.iter().map(|b| format!("{b:02x}")).collect();
    Hash::new(hex).expect("SHA-256 always produces 64 hex chars")
}

// ── Event ──────────────────────────────────────────────────────────────

#[derive(Debug, Clone)]
pub struct Event {
    pub version: u32,
    pub id: EventId,
    pub event_type: EventType,
    pub timestamp_nanos: u64,
    pub source: ActorId,
    content: BTreeMap<String, Value>,
    pub causes: NonEmpty<EventId>,
    pub conversation_id: ConversationId,
    pub hash: Hash,
    pub prev_hash: Hash,
    pub signature: Signature,
}

impl Event {
    pub fn content(&self) -> BTreeMap<String, Value> {
        self.content.clone()
    }
}

// ── UUID v7 ────────────────────────────────────────────────────────────

pub fn new_event_id() -> EventId {
    let id = Uuid::now_v7();
    EventId::new(id.to_string()).expect("UUID v7 is always valid")
}

// ── Event factories ────────────────────────────────────────────────────

pub fn create_event(
    event_type: EventType,
    source: ActorId,
    content: BTreeMap<String, Value>,
    causes: Vec<EventId>,
    conversation_id: ConversationId,
    prev_hash: Hash,
    signer: &dyn Signer,
    version: u32,
) -> Event {
    let event_id = new_event_id();
    let timestamp_nanos = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .unwrap()
        .as_nanos() as u64;
    let content_json = canonical_content_json(&content);

    let cause_strs: Vec<&str> = causes.iter().map(|c| c.value()).collect();
    let canon = canonical_form(
        version, prev_hash.value(), &cause_strs,
        event_id.value(), event_type.value(),
        source.value(), conversation_id.value(),
        timestamp_nanos, &content_json,
    );

    let hash = compute_hash(&canon);
    let sig = signer.sign(canon.as_bytes());

    Event {
        version,
        id: event_id,
        event_type,
        timestamp_nanos,
        source,
        content,
        causes: NonEmpty::of(causes).expect("causes must be non-empty"),
        conversation_id,
        hash,
        prev_hash,
        signature: sig,
    }
}

pub fn create_bootstrap(source: ActorId, signer: &dyn Signer, version: u32) -> Event {
    let event_id = new_event_id();
    let timestamp_nanos = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .unwrap()
        .as_nanos() as u64;
    let conversation_id = ConversationId::new(format!("conv_{}", source.value())).unwrap();

    let now = chrono_like_utc();
    let mut content = BTreeMap::new();
    content.insert("ActorID".to_string(), Value::String(source.value().to_string()));
    content.insert("ChainGenesis".to_string(), Value::String(Hash::zero().value().to_string()));
    content.insert("Timestamp".to_string(), Value::String(now));

    let content_json = canonical_content_json(&content);

    let canon = canonical_form(
        version, "", &[],
        event_id.value(), "system.bootstrapped",
        source.value(), conversation_id.value(),
        timestamp_nanos, &content_json,
    );

    let hash = compute_hash(&canon);
    let sig = signer.sign(canon.as_bytes());

    Event {
        version,
        id: event_id.clone(),
        event_type: EventType::new("system.bootstrapped").unwrap(),
        timestamp_nanos,
        source,
        content,
        causes: NonEmpty::of(vec![event_id]).expect("self-ref is non-empty"),
        conversation_id,
        hash,
        prev_hash: Hash::zero(),
        signature: sig,
    }
}

fn chrono_like_utc() -> String {
    use std::time::SystemTime;
    let d = SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .unwrap();
    let secs = d.as_secs();
    let days = secs / 86400;
    let rem = secs % 86400;
    let h = rem / 3600;
    let m = (rem % 3600) / 60;
    let s = rem % 60;

    // Simplified date from days since epoch
    let (year, month, day) = days_to_ymd(days);
    format!("{year:04}-{month:02}-{day:02}T{h:02}:{m:02}:{s:02}Z")
}

fn days_to_ymd(days: u64) -> (u64, u64, u64) {
    // Civil calendar from days since 1970-01-01
    let z = days + 719468;
    let era = z / 146097;
    let doe = z - era * 146097;
    let yoe = (doe - doe / 1460 + doe / 36524 - doe / 146096) / 365;
    let y = yoe + era * 400;
    let doy = doe - (365 * yoe + yoe / 4 - yoe / 100);
    let mp = (5 * doy + 2) / 153;
    let d = doy - (153 * mp + 2) / 5 + 1;
    let m = if mp < 10 { mp + 3 } else { mp - 9 };
    let y = if m <= 2 { y + 1 } else { y };
    (y, m, d)
}
