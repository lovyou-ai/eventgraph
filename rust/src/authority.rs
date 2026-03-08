//! Authority module — three-tier approval for significant actions.
//!
//! Ports the Go `authority` package. Provides weighted authority evaluation,
//! policy-based action matching, and trust-based downgrade from Required
//! to Recommended.

use std::sync::Mutex;

use crate::actor::Actor;
use crate::decision::AuthorityLevel;
use crate::errors::Result;
use crate::trust::TrustModel;
use crate::types::{ActorId, DomainScope, Score};

// ── AuthorityLink ────────────────────────────────────────────────────

/// A single link in an authority chain.
#[derive(Debug, Clone)]
pub struct AuthorityLink {
    pub actor: ActorId,
    pub level: AuthorityLevel,
    pub weight: Score,
}

// ── AuthorityResult ──────────────────────────────────────────────────

/// The result of evaluating authority for an action.
#[derive(Debug, Clone)]
pub struct AuthorityResult {
    pub level: AuthorityLevel,
    pub weight: Score,
    pub chain: Vec<AuthorityLink>,
}

// ── AuthorityPolicy ──────────────────────────────────────────────────

/// Defines the authority requirements for an action pattern.
#[derive(Debug, Clone)]
pub struct AuthorityPolicy {
    pub action: String,
    pub level: AuthorityLevel,
    pub min_trust: Option<Score>,
    pub scope: Option<DomainScope>,
}

// ── AuthorityChain trait ─────────────────────────────────────────────

/// Evaluates authority. Returns weighted authority, not binary permission.
pub trait AuthorityChain {
    /// Evaluates the authority level required for the given actor and action.
    fn evaluate(&self, actor: &Actor, action: &str) -> Result<AuthorityResult>;

    /// Returns the authority chain (list of links) for the given actor and action.
    fn chain(&self, actor: &Actor, action: &str) -> Result<Vec<AuthorityLink>>;

    /// Grants authority from one actor to another in a scope with a weight.
    /// Returns Ok(()) in the flat model (no-op stub).
    fn grant(
        &self,
        from: &Actor,
        to: &Actor,
        scope: &DomainScope,
        weight: Score,
    ) -> Result<()>;

    /// Revokes authority from one actor to another in a scope.
    /// Returns Ok(()) in the flat model (no-op stub).
    fn revoke(&self, from: &Actor, to: &Actor, scope: &DomainScope) -> Result<()>;
}

// ── DefaultAuthorityChain ────────────────────────────────────────────

/// A flat authority model -- no delegation chain.
/// All actions default to Notification unless a policy says otherwise.
pub struct DefaultAuthorityChain {
    policies: Mutex<Vec<AuthorityPolicy>>,
    trust_model: Box<dyn TrustModel + Send>,
}

impl DefaultAuthorityChain {
    /// Creates a new flat authority chain backed by the given trust model.
    pub fn new(trust_model: Box<dyn TrustModel + Send>) -> Self {
        Self {
            policies: Mutex::new(Vec::new()),
            trust_model,
        }
    }

    /// Registers an authority policy. Policies are checked in order; first match wins.
    pub fn add_policy(&self, policy: AuthorityPolicy) {
        let mut policies = self.policies.lock().unwrap();
        policies.push(policy);
    }

    fn find_policy(&self, action: &str) -> AuthorityPolicy {
        let policies = self.policies.lock().unwrap();
        for p in policies.iter() {
            if matches_action(&p.action, action) {
                return p.clone();
            }
        }
        // Default: Notification level
        AuthorityPolicy {
            action: "*".to_string(),
            level: AuthorityLevel::Notification,
            min_trust: None,
            scope: None,
        }
    }
}

impl AuthorityChain for DefaultAuthorityChain {
    fn evaluate(&self, actor: &Actor, action: &str) -> Result<AuthorityResult> {
        let policy = self.find_policy(action);
        let mut level = policy.level;

        // If actor has high enough trust, downgrade Required -> Recommended
        if level == AuthorityLevel::Required {
            if let Some(min_trust) = &policy.min_trust {
                let metrics = self.trust_model.score(actor)?;
                if metrics.overall.value() >= min_trust.value() {
                    level = AuthorityLevel::Recommended;
                }
            }
        }

        let link = AuthorityLink {
            actor: actor.id().clone(),
            level,
            weight: Score::new(1.0).unwrap(),
        };

        Ok(AuthorityResult {
            level,
            weight: Score::new(1.0).unwrap(),
            chain: vec![link],
        })
    }

    fn chain(&self, actor: &Actor, action: &str) -> Result<Vec<AuthorityLink>> {
        let policy = self.find_policy(action);
        Ok(vec![AuthorityLink {
            actor: actor.id().clone(),
            level: policy.level,
            weight: Score::new(1.0).unwrap(),
        }])
    }

    fn grant(
        &self,
        _from: &Actor,
        _to: &Actor,
        _scope: &DomainScope,
        _weight: Score,
    ) -> Result<()> {
        // No-op in the flat model
        Ok(())
    }

    fn revoke(&self, _from: &Actor, _to: &Actor, _scope: &DomainScope) -> Result<()> {
        // No-op in the flat model
        Ok(())
    }
}

// ── matches_action ───────────────────────────────────────────────────

/// Matches an action string against a pattern.
/// Supports exact match, prefix wildcard (e.g. "trust.*"), and global wildcard ("*").
pub fn matches_action(pattern: &str, action: &str) -> bool {
    if pattern == "*" {
        return true;
    }
    if let Some(prefix) = pattern.strip_suffix('*') {
        return action.len() >= prefix.len() && &action[..prefix.len()] == prefix;
    }
    pattern == action
}
