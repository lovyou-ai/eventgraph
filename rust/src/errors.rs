use std::fmt;

#[derive(Debug, Clone)]
pub enum EventGraphError {
    OutOfRange { type_name: &'static str, value: f64, min: f64, max: f64 },
    EmptyRequired { type_name: &'static str },
    InvalidFormat { type_name: &'static str, value: String, expected: &'static str },
    InvalidTransition { from: String, to: String },
    EventNotFound { event_id: String },
    ChainIntegrity { position: usize, detail: String },
    GrammarViolation { detail: String },
}

impl fmt::Display for EventGraphError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::OutOfRange { type_name, value, min, max } =>
                write!(f, "{type_name} value {value} is out of range [{min}, {max}]"),
            Self::EmptyRequired { type_name } =>
                write!(f, "{type_name} cannot be empty"),
            Self::InvalidFormat { type_name, value, expected } =>
                write!(f, "{type_name} value \"{value}\" does not match expected format: {expected}"),
            Self::InvalidTransition { from, to } =>
                write!(f, "Invalid transition from {from} to {to}"),
            Self::EventNotFound { event_id } =>
                write!(f, "Event {event_id} not found"),
            Self::ChainIntegrity { position, detail } =>
                write!(f, "Chain integrity violation at position {position}: {detail}"),
            Self::GrammarViolation { detail } =>
                write!(f, "Grammar violation: {detail}"),
        }
    }
}

impl std::error::Error for EventGraphError {}

pub type Result<T> = std::result::Result<T, EventGraphError>;
