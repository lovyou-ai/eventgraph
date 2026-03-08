/**
 * Social grammar module — the 15 social grammar operations + 4 named functions
 * that create properly hash-chained events on the store. Ports the full grammar
 * from the Go reference implementation.
 */
import { createEvent, type Signer, type Event } from "./event.js";
import type { Store } from "./store.js";
import { ActorId, ConversationId, DomainScope, EdgeId, EventId, EventType, Hash, Option, Weight } from "./types.js";

/** Grammar wraps a Store and provides high-level social grammar operations. */
export class Grammar {
  constructor(private readonly store: Store) {}

  // ── helpers ──────────────────────────────────────────────────────────

  private headPrevHash(): Hash {
    const head = this.store.head();
    return head.isSome ? head.unwrap().hash : Hash.zero();
  }

  private record(
    eventType: EventType,
    source: ActorId,
    content: Record<string, unknown>,
    causes: EventId[],
    conversationId: ConversationId,
    signer: Signer,
  ): Event {
    const prevHash = this.headPrevHash();
    return this.store.append(
      createEvent(eventType, source, content, causes, conversationId, prevHash, signer),
    );
  }

  // ── vertex operations ────────────────────────────────────────────────

  /** Emit creates independent content. Causes must be non-empty. */
  emit(
    source: ActorId,
    body: string,
    conversationId: ConversationId,
    causes: EventId[],
    signer: Signer,
  ): Event {
    if (causes.length === 0) {
      throw new Error("emit requires at least one cause");
    }
    return this.record(
      new EventType("grammar.emitted"), source,
      { Body: body },
      causes, conversationId, signer,
    );
  }

  /** Respond creates causally dependent, subordinate content. */
  respond(
    source: ActorId,
    body: string,
    parent: EventId,
    conversationId: ConversationId,
    signer: Signer,
  ): Event {
    return this.record(
      new EventType("grammar.responded"), source,
      { Body: body, Parent: parent.value },
      [parent], conversationId, signer,
    );
  }

  /** Derive creates causally dependent but independent content. */
  derive(
    source: ActorId,
    body: string,
    sourceEvent: EventId,
    conversationId: ConversationId,
    signer: Signer,
  ): Event {
    return this.record(
      new EventType("grammar.derived"), source,
      { Body: body, Source: sourceEvent.value },
      [sourceEvent], conversationId, signer,
    );
  }

  /** Extend creates sequential content from the same author. */
  extend(
    source: ActorId,
    body: string,
    previous: EventId,
    conversationId: ConversationId,
    signer: Signer,
  ): Event {
    return this.record(
      new EventType("grammar.extended"), source,
      { Body: body, Previous: previous.value },
      [previous], conversationId, signer,
    );
  }

  /** Retract tombstones own content. Only the original author can retract. */
  retract(
    source: ActorId,
    target: EventId,
    reason: string,
    conversationId: ConversationId,
    signer: Signer,
  ): Event {
    const targetEvent = this.store.get(target);
    if (targetEvent.source.value !== source.value) {
      throw new Error(
        `retract: actor ${source.value} cannot retract event ${target.value} authored by ${targetEvent.source.value}`,
      );
    }
    return this.record(
      new EventType("grammar.retracted"), source,
      { Target: target.value, Reason: reason },
      [target], conversationId, signer,
    );
  }

  /** Annotate attaches metadata to existing content. */
  annotate(
    source: ActorId,
    target: EventId,
    key: string,
    value: string,
    conversationId: ConversationId,
    signer: Signer,
  ): Event {
    return this.record(
      new EventType("grammar.annotated"), source,
      { Target: target.value, Key: key, Value: value },
      [target], conversationId, signer,
    );
  }

  /** Merge joins two or more independent subtrees. Requires >= 2 sources. */
  merge(
    source: ActorId,
    body: string,
    sources: EventId[],
    conversationId: ConversationId,
    signer: Signer,
  ): Event {
    if (sources.length < 2) {
      throw new Error("merge requires at least two sources");
    }
    return this.record(
      new EventType("grammar.merged"), source,
      { Body: body, Sources: sources.map((s) => s.value) },
      sources, conversationId, signer,
    );
  }

  // ── edge operations ─────────────────────────────────────────────────

  /** Acknowledge creates a content-free centripetal edge toward a vertex. (Operation 7) */
  acknowledge(
    source: ActorId,
    targetEvent: EventId,
    targetActor: ActorId,
    conversationId: ConversationId,
    signer: Signer,
  ): Event {
    return this.record(
      new EventType("edge.created"), source,
      {
        From: source.value, To: targetActor.value,
        EdgeType: "acknowledgement", Weight: 0,
        Direction: "centripetal", Scope: null,
      },
      [targetEvent], conversationId, signer,
    );
  }

  /** Propagate redistributes a vertex into the actor's subgraph. (Operation 8) */
  propagate(
    source: ActorId,
    targetEvent: EventId,
    targetActor: ActorId,
    conversationId: ConversationId,
    signer: Signer,
  ): Event {
    return this.record(
      new EventType("edge.created"), source,
      {
        From: source.value, To: targetActor.value,
        EdgeType: "reference", Weight: 0,
        Direction: "centrifugal", Scope: null,
      },
      [targetEvent], conversationId, signer,
    );
  }

  /** Endorse creates a reputation-staked edge toward a vertex. (Operation 9) */
  endorse(
    source: ActorId,
    targetEvent: EventId,
    targetActor: ActorId,
    weight: Weight,
    scope: Option<DomainScope>,
    conversationId: ConversationId,
    signer: Signer,
  ): Event {
    return this.record(
      new EventType("edge.created"), source,
      {
        From: source.value, To: targetActor.value,
        EdgeType: "endorsement", Weight: weight.value,
        Direction: "centripetal",
        Scope: scope.isSome ? scope.unwrap().value : null,
      },
      [targetEvent], conversationId, signer,
    );
  }

  /** Subscribe creates a persistent, future-oriented edge to an actor. (Operation 10) */
  subscribe(
    source: ActorId,
    target: ActorId,
    scope: Option<DomainScope>,
    cause: EventId,
    conversationId: ConversationId,
    signer: Signer,
  ): Event {
    return this.record(
      new EventType("edge.created"), source,
      {
        From: source.value, To: target.value,
        EdgeType: "subscription", Weight: 0,
        Direction: "centripetal",
        Scope: scope.isSome ? scope.unwrap().value : null,
      },
      [cause], conversationId, signer,
    );
  }

  /** Channel creates a private, bidirectional, content-bearing edge. (Operation 11) */
  channel(
    source: ActorId,
    target: ActorId,
    scope: Option<DomainScope>,
    cause: EventId,
    conversationId: ConversationId,
    signer: Signer,
  ): Event {
    return this.record(
      new EventType("edge.created"), source,
      {
        From: source.value, To: target.value,
        EdgeType: "channel", Weight: 0,
        Direction: "centripetal",
        Scope: scope.isSome ? scope.unwrap().value : null,
      },
      [cause], conversationId, signer,
    );
  }

  /** Delegate grants authority for another actor to operate as you. (Operation 12) */
  delegate(
    source: ActorId,
    target: ActorId,
    scope: DomainScope,
    weight: Weight,
    cause: EventId,
    conversationId: ConversationId,
    signer: Signer,
  ): Event {
    return this.record(
      new EventType("edge.created"), source,
      {
        From: source.value, To: target.value,
        EdgeType: "delegation", Weight: weight.value,
        Direction: "centrifugal",
        Scope: scope.value,
      },
      [cause], conversationId, signer,
    );
  }

  /** Consent records a consent proposal signed by partyA. (Operation 13) */
  consent(
    partyA: ActorId,
    partyB: ActorId,
    agreement: string,
    scope: DomainScope,
    cause: EventId,
    conversationId: ConversationId,
    signer: Signer,
  ): Event {
    return this.record(
      new EventType("grammar.consent"), partyA,
      {
        PartyA: partyA.value, PartyB: partyB.value,
        Agreement: agreement, Scope: scope.value,
      },
      [cause], conversationId, signer,
    );
  }

  /** Sever removes a subscription, channel, or delegation via edge supersession. (Operation 14) */
  sever(
    source: ActorId,
    previousEdgeId: EdgeId,
    cause: EventId,
    conversationId: ConversationId,
    signer: Signer,
  ): Event {
    // Convert EdgeId to EventId to look up the edge event
    const edgeEventId = new EventId(previousEdgeId.value);
    const edgeEvent = this.store.get(edgeEventId);
    const ec = edgeEvent.content;

    // Only subscriptions, channels, and delegations are severable
    const edgeType = ec.EdgeType as string;
    if (edgeType !== "subscription" && edgeType !== "channel" && edgeType !== "delegation") {
      throw new Error(
        `sever: edge type ${edgeType} is not severable (only subscription, channel, delegation)`,
      );
    }

    // Only a party to the edge can sever it
    if (ec.From !== source.value && ec.To !== source.value) {
      throw new Error(
        `sever: actor ${source.value} is not a party to edge ${previousEdgeId.value}`,
      );
    }

    const causes: EventId[] = [edgeEventId];
    if (cause.value !== edgeEventId.value) {
      causes.push(cause);
    }

    return this.record(
      new EventType("edge.superseded"), source,
      {
        PreviousEdge: previousEdgeId.value,
        NewEdge: null,
        Reason: cause.value,
      },
      causes, conversationId, signer,
    );
  }

  // ── named functions (compositions of primitives) ────────────────────

  /** Challenge is Respond + dispute flag: formal dispute that follows content. */
  challenge(
    source: ActorId,
    body: string,
    target: EventId,
    conversationId: ConversationId,
    signer: Signer,
  ): { response: Event; disputeFlag: Event } {
    const response = this.respond(source, body, target, conversationId, signer);
    const disputeFlag = this.annotate(
      source, response.id, "dispute", "challenged", conversationId, signer,
    );
    return { response, disputeFlag };
  }

  /** Recommend is Propagate + Channel: directed sharing to a specific person. */
  recommend(
    source: ActorId,
    target: EventId,
    targetActor: ActorId,
    conversationId: ConversationId,
    signer: Signer,
  ): { propagateEv: Event; channelEv: Event } {
    const propagateEv = this.propagate(source, target, targetActor, conversationId, signer);
    const channelEv = this.channel(
      source, targetActor, Option.none<DomainScope>(), propagateEv.id, conversationId, signer,
    );
    return { propagateEv, channelEv };
  }

  /** Invite is Endorse + Subscribe: trust-staked introduction of a new actor. */
  invite(
    source: ActorId,
    target: ActorId,
    weight: Weight,
    scope: Option<DomainScope>,
    cause: EventId,
    conversationId: ConversationId,
    signer: Signer,
  ): { endorseEv: Event; subscribeEv: Event } {
    const endorseEv = this.endorse(source, cause, target, weight, scope, conversationId, signer);
    const subscribeEv = this.subscribe(source, target, scope, endorseEv.id, conversationId, signer);
    return { endorseEv, subscribeEv };
  }

  /** Forgive is Subscribe after Sever: reconciliation with history intact. */
  forgive(
    source: ActorId,
    severEvent: EventId,
    target: ActorId,
    scope: Option<DomainScope>,
    conversationId: ConversationId,
    signer: Signer,
  ): Event {
    return this.subscribe(source, target, scope, severEvent, conversationId, signer);
  }
}
