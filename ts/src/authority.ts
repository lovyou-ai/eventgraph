import { ActorId, DomainScope, Score } from "./types.js";
import { Actor } from "./actor.js";
import { AuthorityLevel } from "./decision.js";
import { TrustModel } from "./trust.js";

// Re-export AuthorityLevel for convenience
export { AuthorityLevel } from "./decision.js";

// ── AuthorityLink ─────────────────────────────────────────────────────────

/** A link in an authority chain. */
export interface AuthorityLink {
  actor: ActorId;
  level: AuthorityLevel;
  weight: Score;
}

// ── AuthorityResult ───────────────────────────────────────────────────────

/** The result of evaluating authority for an action. */
export interface AuthorityResult {
  level: AuthorityLevel;
  weight: Score;
  chain: AuthorityLink[];
}

// ── AuthorityPolicy ───────────────────────────────────────────────────────

/** Defines the authority requirements for an action. */
export interface AuthorityPolicy {
  action: string;
  level: AuthorityLevel;
  minTrust?: Score;
  scope?: DomainScope;
}

// ── AuthorityChain interface ──────────────────────────────────────────────

/** Evaluates authority. Returns weighted authority, not binary permission. */
export interface AuthorityChain {
  evaluate(actor: Actor, action: string): AuthorityResult;
  chain(actor: Actor, action: string): AuthorityLink[];
  grant(from: Actor, to: Actor, scope: DomainScope, weight: Score): void;
  revoke(from: Actor, to: Actor, scope: DomainScope): void;
}

// ── matchesAction ─────────────────────────────────────────────────────────

/** Matches an action against a policy pattern. Supports exact, prefix*, and * (catch-all). */
export function matchesAction(pattern: string, action: string): boolean {
  if (pattern === "*") {
    return true;
  }
  if (pattern.length > 0 && pattern[pattern.length - 1] === "*") {
    const prefix = pattern.slice(0, -1);
    return action.length >= prefix.length && action.slice(0, prefix.length) === prefix;
  }
  return pattern === action;
}

// ── DefaultAuthorityChain ─────────────────────────────────────────────────

/**
 * Flat authority model — no delegation chain.
 * All actions default to Notification unless a policy says otherwise.
 */
export class DefaultAuthorityChain implements AuthorityChain {
  private readonly policies: AuthorityPolicy[] = [];
  private readonly trustModel: TrustModel;

  constructor(trustModel: TrustModel) {
    this.trustModel = trustModel;
  }

  /** Registers an authority policy. Policies are checked in order; first match wins. */
  addPolicy(policy: AuthorityPolicy): void {
    this.policies.push(policy);
  }

  evaluate(actor: Actor, action: string): AuthorityResult {
    const policy = this.findPolicy(action);
    let level = policy.level;

    // If actor has high enough trust, downgrade Required -> Recommended
    if (level === AuthorityLevel.Required && policy.minTrust !== undefined) {
      const metrics = this.trustModel.score(actor);
      if (metrics.overall.value >= policy.minTrust.value) {
        level = AuthorityLevel.Recommended;
      }
    }

    const link: AuthorityLink = {
      actor: actor.id,
      level,
      weight: new Score(1.0),
    };

    return {
      level,
      weight: new Score(1.0),
      chain: [link],
    };
  }

  chain(actor: Actor, action: string): AuthorityLink[] {
    const policy = this.findPolicy(action);
    return [
      {
        actor: actor.id,
        level: policy.level,
        weight: new Score(1.0),
      },
    ];
  }

  grant(_from: Actor, _to: Actor, _scope: DomainScope, _weight: Score): void {
    // No-op in flat model — would emit an edge.created event in full implementation
  }

  revoke(_from: Actor, _to: Actor, _scope: DomainScope): void {
    // No-op in flat model — would emit an edge.superseded event in full implementation
  }

  private findPolicy(action: string): AuthorityPolicy {
    for (const p of this.policies) {
      if (matchesAction(p.action, action)) {
        return p;
      }
    }
    // Default: Notification level
    return {
      action: "*",
      level: AuthorityLevel.Notification,
    };
  }
}
