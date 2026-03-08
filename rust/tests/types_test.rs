use eventgraph::types::*;

// ── Score ──────────────────────────────────────────────────────────────

#[test] fn score_valid_zero() { assert!(Score::new(0.0).is_ok()); }
#[test] fn score_valid_one() { assert!(Score::new(1.0).is_ok()); }
#[test] fn score_valid_mid() { assert_eq!(Score::new(0.5).unwrap().value(), 0.5); }
#[test] fn score_reject_negative() { assert!(Score::new(-0.1).is_err()); }
#[test] fn score_reject_over() { assert!(Score::new(1.1).is_err()); }
#[test] fn score_reject_nan() { assert!(Score::new(f64::NAN).is_err()); }

// ── Weight ─────────────────────────────────────────────────────────────

#[test] fn weight_valid_neg_one() { assert!(Weight::new(-1.0).is_ok()); }
#[test] fn weight_valid_one() { assert!(Weight::new(1.0).is_ok()); }
#[test] fn weight_valid_zero() { assert_eq!(Weight::new(0.0).unwrap().value(), 0.0); }
#[test] fn weight_reject_under() { assert!(Weight::new(-1.1).is_err()); }
#[test] fn weight_reject_over() { assert!(Weight::new(1.1).is_err()); }

// ── Activation ─────────────────────────────────────────────────────────

#[test] fn activation_valid() { assert!(Activation::new(0.5).is_ok()); }
#[test] fn activation_reject_neg() { assert!(Activation::new(-0.01).is_err()); }

// ── Layer ──────────────────────────────────────────────────────────────

#[test] fn layer_valid_zero() { assert!(Layer::new(0).is_ok()); }
#[test] fn layer_valid_13() { assert!(Layer::new(13).is_ok()); }
#[test] fn layer_reject_14() { assert!(Layer::new(14).is_err()); }

// ── Cadence ────────────────────────────────────────────────────────────

#[test] fn cadence_valid_1() { assert!(Cadence::new(1).is_ok()); }
#[test] fn cadence_valid_100() { assert_eq!(Cadence::new(100).unwrap().value(), 100); }
#[test] fn cadence_reject_0() { assert!(Cadence::new(0).is_err()); }

// ── Hash ───────────────────────────────────────────────────────────────

#[test] fn hash_valid_64_hex() { assert!(Hash::new("a".repeat(64)).is_ok()); }
#[test] fn hash_reject_short() { assert!(Hash::new("abc").is_err()); }
#[test] fn hash_reject_empty() { assert!(Hash::new("").is_err()); }
#[test] fn hash_zero() { assert!(Hash::zero().is_zero()); }
#[test] fn hash_not_zero() { assert!(!Hash::new("a".repeat(64)).unwrap().is_zero()); }

// ── EventId ────────────────────────────────────────────────────────────

#[test] fn event_id_valid() { assert!(EventId::new("019462a0-0000-7000-8000-000000000001").is_ok()); }
#[test] fn event_id_reject_non_v7() { assert!(EventId::new("12345678-1234-4123-8123-123456789012").is_err()); }
#[test] fn event_id_reject_garbage() { assert!(EventId::new("not-a-uuid").is_err()); }

// ── EventType ──────────────────────────────────────────────────────────

#[test] fn event_type_valid() { assert!(EventType::new("trust.updated").is_ok()); }
#[test] fn event_type_valid_single() { assert!(EventType::new("system").is_ok()); }
#[test] fn event_type_reject_uppercase() { assert!(EventType::new("Trust.Updated").is_err()); }
#[test] fn event_type_reject_empty() { assert!(EventType::new("").is_err()); }

// ── ActorId ────────────────────────────────────────────────────────────

#[test] fn actor_id_valid() { assert!(ActorId::new("alice").is_ok()); }
#[test] fn actor_id_reject_empty() { assert!(ActorId::new("").is_err()); }

// ── SubscriptionPattern ────────────────────────────────────────────────

#[test] fn sub_wildcard_matches_all() {
    let p = SubscriptionPattern::new("*").unwrap();
    assert!(p.matches(&EventType::new("trust.updated").unwrap()));
}

#[test] fn sub_prefix_match() {
    let p = SubscriptionPattern::new("trust.*").unwrap();
    assert!(p.matches(&EventType::new("trust.updated").unwrap()));
    assert!(!p.matches(&EventType::new("system.bootstrapped").unwrap()));
}

#[test] fn sub_exact_match() {
    let p = SubscriptionPattern::new("trust.updated").unwrap();
    assert!(p.matches(&EventType::new("trust.updated").unwrap()));
    assert!(!p.matches(&EventType::new("trust.created").unwrap()));
}

// ── Lifecycle ──────────────────────────────────────────────────────────

#[test] fn lifecycle_valid_dormant_to_activating() {
    assert!(LifecycleState::Dormant.can_transition_to(LifecycleState::Activating));
}

#[test] fn lifecycle_valid_active_to_processing() {
    assert!(LifecycleState::Active.can_transition_to(LifecycleState::Processing));
}

#[test] fn lifecycle_invalid_dormant_to_active() {
    assert!(!LifecycleState::Dormant.can_transition_to(LifecycleState::Active));
}

#[test] fn lifecycle_invalid_memorial_terminal() {
    assert!(!LifecycleState::Memorial.can_transition_to(LifecycleState::Active));
    assert!(!LifecycleState::Memorial.can_transition_to(LifecycleState::Dormant));
}

// ── NonEmpty ───────────────────────────────────────────────────────────

#[test] fn nonempty_valid() { assert!(NonEmpty::of(vec![1, 2, 3]).is_ok()); }
#[test] fn nonempty_reject_empty() { assert!(NonEmpty::<i32>::of(vec![]).is_err()); }
#[test] fn nonempty_len() { assert_eq!(NonEmpty::of(vec![1, 2]).unwrap().len(), 2); }

// ── Signature ──────────────────────────────────────────────────────────

#[test] fn signature_valid_64_bytes() { assert!(Signature::new(vec![0u8; 64]).is_ok()); }
#[test] fn signature_reject_wrong_len() { assert!(Signature::new(vec![0u8; 32]).is_err()); }
