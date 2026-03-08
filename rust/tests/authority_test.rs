use std::collections::BTreeMap;

use eventgraph::actor::{Actor, ActorStatus, ActorType};
use eventgraph::authority::{
    matches_action, AuthorityChain, AuthorityPolicy, DefaultAuthorityChain,
};
use eventgraph::decision::AuthorityLevel;
use eventgraph::trust::{DefaultTrustModel, TrustConfig};
use eventgraph::types::{ActorId, DomainScope, PublicKey, Score};

fn test_actor(name: &str, key_byte: u8) -> Actor {
    let mut key = [0u8; 32];
    key[0] = key_byte;
    Actor::new(
        ActorId::new(name).unwrap(),
        PublicKey::new(key),
        name.to_string(),
        ActorType::Human,
        BTreeMap::new(),
        1_000_000_000,
        ActorStatus::Active,
    )
}

fn default_chain() -> DefaultAuthorityChain {
    let model = DefaultTrustModel::new(TrustConfig::default());
    DefaultAuthorityChain::new(Box::new(model))
}

// ── Tests ────────────────────────────────────────────────────────────

#[test]
fn test_default_level_is_notification() {
    let chain = default_chain();
    let actor = test_actor("alice", 1);
    let result = chain.evaluate(&actor, "some.random.action").unwrap();
    assert_eq!(result.level, AuthorityLevel::Notification);
    assert_eq!(result.chain.len(), 1);
}

#[test]
fn test_policy_exact_match() {
    let chain = default_chain();
    chain.add_policy(AuthorityPolicy {
        action: "actor.suspend".to_string(),
        level: AuthorityLevel::Required,
        min_trust: None,
        scope: None,
    });

    let actor = test_actor("alice", 1);
    let result = chain.evaluate(&actor, "actor.suspend").unwrap();
    assert_eq!(result.level, AuthorityLevel::Required);
}

#[test]
fn test_policy_wildcard() {
    let chain = default_chain();
    chain.add_policy(AuthorityPolicy {
        action: "trust.*".to_string(),
        level: AuthorityLevel::Recommended,
        min_trust: None,
        scope: None,
    });

    let actor = test_actor("alice", 1);
    let result = chain.evaluate(&actor, "trust.update").unwrap();
    assert_eq!(result.level, AuthorityLevel::Recommended);
}

#[test]
fn test_policy_global_wildcard() {
    let chain = default_chain();
    chain.add_policy(AuthorityPolicy {
        action: "*".to_string(),
        level: AuthorityLevel::Required,
        min_trust: None,
        scope: None,
    });

    let actor = test_actor("alice", 1);
    let result = chain.evaluate(&actor, "anything.at.all").unwrap();
    assert_eq!(result.level, AuthorityLevel::Required);
}

#[test]
fn test_policy_first_match_wins() {
    let chain = default_chain();
    chain.add_policy(AuthorityPolicy {
        action: "deploy".to_string(),
        level: AuthorityLevel::Required,
        min_trust: None,
        scope: None,
    });
    chain.add_policy(AuthorityPolicy {
        action: "deploy".to_string(),
        level: AuthorityLevel::Notification,
        min_trust: None,
        scope: None,
    });

    let actor = test_actor("alice", 1);
    let result = chain.evaluate(&actor, "deploy").unwrap();
    assert_eq!(result.level, AuthorityLevel::Required);
}

#[test]
fn test_policy_no_match_defaults_to_notification() {
    let chain = default_chain();
    chain.add_policy(AuthorityPolicy {
        action: "deploy".to_string(),
        level: AuthorityLevel::Required,
        min_trust: None,
        scope: None,
    });

    let actor = test_actor("alice", 1);
    let result = chain.evaluate(&actor, "review").unwrap();
    assert_eq!(result.level, AuthorityLevel::Notification);
}

#[test]
fn test_trust_downgrade_required_to_recommended() {
    // Initial trust is 0.0, threshold is 0.001 — trust too low, stays Required
    let chain = default_chain();
    chain.add_policy(AuthorityPolicy {
        action: "deploy".to_string(),
        level: AuthorityLevel::Required,
        min_trust: Some(Score::new(0.001).unwrap()),
        scope: None,
    });

    let actor = test_actor("alice", 1);
    let result = chain.evaluate(&actor, "deploy").unwrap();
    assert_eq!(result.level, AuthorityLevel::Required);
}

#[test]
fn test_chain_returns_single_link() {
    let chain = default_chain();
    let actor = test_actor("alice", 1);
    let links = chain.chain(&actor, "any.action").unwrap();
    assert_eq!(links.len(), 1);
    assert_eq!(links[0].actor, *actor.id());
}

#[test]
fn test_grant_and_revoke_are_noop() {
    let chain = default_chain();
    let alice = test_actor("alice", 1);
    let bob = test_actor("bob", 2);
    let scope = DomainScope::new("code_review").unwrap();
    let weight = Score::new(0.8).unwrap();

    assert!(chain.grant(&alice, &bob, &scope, weight).is_ok());
    assert!(chain.revoke(&alice, &bob, &scope).is_ok());
}

#[test]
fn test_authority_result_weight() {
    let chain = default_chain();
    let actor = test_actor("alice", 1);
    let result = chain.evaluate(&actor, "test").unwrap();
    assert!((result.weight.value() - 1.0).abs() < f64::EPSILON);
}

// ── matches_action unit tests ────────────────────────────────────────

#[test]
fn test_matches_action_exact() {
    assert!(matches_action("deploy", "deploy"));
    assert!(!matches_action("deploy", "review"));
}

#[test]
fn test_matches_action_prefix_wildcard() {
    assert!(matches_action("trust.*", "trust.update"));
    assert!(matches_action("trust.*", "trust.decay"));
    assert!(!matches_action("trust.*", "actor.suspend"));
}

#[test]
fn test_matches_action_global_wildcard() {
    assert!(matches_action("*", "anything"));
    assert!(matches_action("*", ""));
}
