use regex::Regex;
use std::sync::LazyLock;

use crate::errors::{EventGraphError, Result};

// ── Regex patterns ─────────────────────────────────────────────────────

static UUID_V7_RE: LazyLock<Regex> = LazyLock::new(|| {
    Regex::new(r"^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$").unwrap()
});

static UUID_RE: LazyLock<Regex> = LazyLock::new(|| {
    Regex::new(r"^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$").unwrap()
});

static EVENT_TYPE_RE: LazyLock<Regex> = LazyLock::new(|| {
    Regex::new(r"^[a-z][a-z0-9]*(\.[a-z][a-z0-9]*)*$").unwrap()
});

static DOMAIN_SCOPE_RE: LazyLock<Regex> = LazyLock::new(|| {
    Regex::new(r"^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)*$").unwrap()
});

static SUBSCRIPTION_RE: LazyLock<Regex> = LazyLock::new(|| {
    Regex::new(r"^(\*|[a-z][a-z0-9]*(\.[a-z][a-z0-9]*)*(\.\*)?)$").unwrap()
});

static HEX64_RE: LazyLock<Regex> = LazyLock::new(|| {
    Regex::new(r"^[0-9a-f]{64}$").unwrap()
});

// ── Constrained numerics ───────────────────────────────────────────────

macro_rules! constrained_float {
    ($name:ident, $min:expr, $max:expr) => {
        #[derive(Debug, Clone, Copy, PartialEq)]
        pub struct $name(f64);

        impl $name {
            pub fn new(value: f64) -> Result<Self> {
                if value.is_nan() || value < $min || value > $max {
                    return Err(EventGraphError::OutOfRange {
                        type_name: stringify!($name),
                        value,
                        min: $min,
                        max: $max,
                    });
                }
                let v = if value == 0.0 { 0.0 } else { value };
                Ok(Self(v))
            }

            pub fn value(self) -> f64 { self.0 }
        }
    };
}

constrained_float!(Score, 0.0, 1.0);
constrained_float!(Weight, -1.0, 1.0);
constrained_float!(Activation, 0.0, 1.0);

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Layer(u8);

impl Layer {
    pub fn new(value: u8) -> Result<Self> {
        if value > 13 {
            return Err(EventGraphError::OutOfRange {
                type_name: "Layer",
                value: value as f64,
                min: 0.0,
                max: 13.0,
            });
        }
        Ok(Self(value))
    }
    pub fn value(self) -> u8 { self.0 }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Cadence(u32);

impl Cadence {
    pub fn new(value: u32) -> Result<Self> {
        if value < 1 {
            return Err(EventGraphError::OutOfRange {
                type_name: "Cadence",
                value: value as f64,
                min: 1.0,
                max: f64::INFINITY,
            });
        }
        Ok(Self(value))
    }
    pub fn value(self) -> u32 { self.0 }
}

// ── Typed IDs ──────────────────────────────────────────────────────────

macro_rules! validated_string {
    ($name:ident, $re:expr, $fmt:expr) => {
        #[derive(Debug, Clone, PartialEq, Eq, Hash)]
        pub struct $name(String);

        impl $name {
            pub fn new(value: impl Into<String>) -> Result<Self> {
                let v: String = value.into().to_lowercase();
                if !$re.is_match(&v) {
                    return Err(EventGraphError::InvalidFormat {
                        type_name: stringify!($name),
                        value: v,
                        expected: $fmt,
                    });
                }
                Ok(Self(v))
            }
            pub fn value(&self) -> &str { &self.0 }
        }

        impl std::fmt::Display for $name {
            fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str(&self.0)
            }
        }
    };
}

validated_string!(EventId, UUID_V7_RE, "UUID v7");
validated_string!(EdgeId, UUID_V7_RE, "UUID v7");
validated_string!(EnvelopeId, UUID_RE, "UUID");
validated_string!(TreatyId, UUID_RE, "UUID");

#[derive(Debug, Clone, PartialEq, Eq, Hash)]
pub struct Hash(String);

impl Hash {
    pub fn new(value: impl Into<String>) -> Result<Self> {
        let v: String = value.into().to_lowercase();
        if !HEX64_RE.is_match(&v) {
            return Err(EventGraphError::InvalidFormat {
                type_name: "Hash",
                value: v,
                expected: "64 hex characters (SHA-256)",
            });
        }
        Ok(Self(v))
    }

    pub fn zero() -> Self { Self("0".repeat(64)) }
    pub fn is_zero(&self) -> bool { self.0.chars().all(|c| c == '0') }
    pub fn value(&self) -> &str { &self.0 }
}

impl std::fmt::Display for Hash {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.write_str(&self.0)
    }
}

#[derive(Debug, Clone, PartialEq, Eq, Hash)]
pub struct EventType(String);

impl EventType {
    pub fn new(value: impl Into<String>) -> Result<Self> {
        let v: String = value.into();
        if !EVENT_TYPE_RE.is_match(&v) {
            return Err(EventGraphError::InvalidFormat {
                type_name: "EventType",
                value: v,
                expected: "dot-separated lowercase segments",
            });
        }
        Ok(Self(v))
    }
    pub fn value(&self) -> &str { &self.0 }
}

impl std::fmt::Display for EventType {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.write_str(&self.0)
    }
}

macro_rules! nonempty_string {
    ($name:ident) => {
        #[derive(Debug, Clone, PartialEq, Eq, Hash)]
        pub struct $name(String);

        impl $name {
            pub fn new(value: impl Into<String>) -> Result<Self> {
                let v: String = value.into();
                if v.is_empty() {
                    return Err(EventGraphError::EmptyRequired {
                        type_name: stringify!($name),
                    });
                }
                Ok(Self(v))
            }
            pub fn value(&self) -> &str { &self.0 }
        }

        impl std::fmt::Display for $name {
            fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str(&self.0)
            }
        }
    };
}

nonempty_string!(ActorId);
nonempty_string!(ConversationId);
nonempty_string!(SystemUri);
nonempty_string!(PrimitiveId);

#[derive(Debug, Clone, PartialEq, Eq, Hash)]
pub struct DomainScope(String);

impl DomainScope {
    pub fn new(value: impl Into<String>) -> Result<Self> {
        let v: String = value.into();
        if !DOMAIN_SCOPE_RE.is_match(&v) {
            return Err(EventGraphError::InvalidFormat {
                type_name: "DomainScope",
                value: v,
                expected: "lowercase dot/underscore-separated namespace",
            });
        }
        Ok(Self(v))
    }
    pub fn value(&self) -> &str { &self.0 }
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct SubscriptionPattern(String);

impl SubscriptionPattern {
    pub fn new(value: impl Into<String>) -> Result<Self> {
        let v: String = value.into();
        if !SUBSCRIPTION_RE.is_match(&v) {
            return Err(EventGraphError::InvalidFormat {
                type_name: "SubscriptionPattern",
                value: v,
                expected: "dot-separated with optional trailing .* or bare *",
            });
        }
        Ok(Self(v))
    }

    pub fn matches(&self, et: &EventType) -> bool {
        if self.0 == "*" { return true; }
        if let Some(prefix) = self.0.strip_suffix(".*") {
            return et.value() == prefix || et.value().starts_with(&format!("{prefix}."));
        }
        self.0 == et.value()
    }

    pub fn value(&self) -> &str { &self.0 }
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct PublicKey([u8; 32]);

impl PublicKey {
    pub fn new(bytes: [u8; 32]) -> Self { Self(bytes) }
    pub fn bytes(&self) -> &[u8; 32] { &self.0 }
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Signature(Vec<u8>);

impl Signature {
    pub fn new(bytes: Vec<u8>) -> Result<Self> {
        if bytes.len() != 64 {
            return Err(EventGraphError::InvalidFormat {
                type_name: "Signature",
                value: hex::encode(&bytes),
                expected: "64 bytes (Ed25519 signature)",
            });
        }
        Ok(Self(bytes))
    }
    pub fn zero() -> Self { Self(vec![0u8; 64]) }
    pub fn bytes(&self) -> &[u8] { &self.0 }
}

// Inline hex encoding to avoid adding a `hex` dep just for error messages
mod hex {
    pub fn encode(bytes: &[u8]) -> String {
        bytes.iter().map(|b| format!("{b:02x}")).collect()
    }
}

// ── NonEmpty ───────────────────────────────────────────────────────────

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct NonEmpty<T>(Vec<T>);

impl<T> NonEmpty<T> {
    pub fn of(items: Vec<T>) -> std::result::Result<Self, &'static str> {
        if items.is_empty() {
            return Err("NonEmpty requires at least one element");
        }
        Ok(Self(items))
    }

    pub fn len(&self) -> usize { self.0.len() }
    pub fn get(&self, index: usize) -> Option<&T> { self.0.get(index) }
    pub fn iter(&self) -> std::slice::Iter<'_, T> { self.0.iter() }
    pub fn as_slice(&self) -> &[T] { &self.0 }
}

impl<T> IntoIterator for NonEmpty<T> {
    type Item = T;
    type IntoIter = std::vec::IntoIter<T>;
    fn into_iter(self) -> Self::IntoIter { self.0.into_iter() }
}

// ── Lifecycle state machine ────────────────────────────────────────────

#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum LifecycleState {
    Dormant,
    Activating,
    Active,
    Processing,
    Emitting,
    Suspending,
    Suspended,
    Memorial,
}

impl LifecycleState {
    pub fn can_transition_to(self, target: Self) -> bool {
        matches!(
            (self, target),
            (Self::Dormant, Self::Activating)
            | (Self::Activating, Self::Active)
            | (Self::Active, Self::Processing)
            | (Self::Active, Self::Suspending)
            | (Self::Active, Self::Memorial)
            | (Self::Processing, Self::Emitting)
            | (Self::Processing, Self::Active)
            | (Self::Emitting, Self::Active)
            | (Self::Suspending, Self::Suspended)
            | (Self::Suspended, Self::Activating)
            | (Self::Suspended, Self::Memorial)
        )
    }

    pub fn as_str(self) -> &'static str {
        match self {
            Self::Dormant => "dormant",
            Self::Activating => "activating",
            Self::Active => "active",
            Self::Processing => "processing",
            Self::Emitting => "emitting",
            Self::Suspending => "suspending",
            Self::Suspended => "suspended",
            Self::Memorial => "memorial",
        }
    }
}

impl std::fmt::Display for LifecycleState {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.write_str(self.as_str())
    }
}
