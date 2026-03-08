//! Per-layer grammar operations as compositions of the 15 base social grammar operations.
//!
//! Each layer gets its own grammar type with domain-specific operations and named
//! multi-step functions. Composition operations are the vocabulary developers use
//! when building on a product graph.

pub mod work;
pub mod market;
pub mod social;
pub mod justice;
pub mod build;
pub mod knowledge;
pub mod alignment;
pub mod identity;
pub mod bond;
pub mod belonging;
pub mod meaning;
pub mod evolution;
pub mod being;
