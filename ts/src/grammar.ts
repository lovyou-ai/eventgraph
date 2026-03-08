/**
 * Social grammar module — high-level vertex operations that create properly
 * hash-chained events on the store. Ports the vertex operations from the Go
 * reference implementation.
 */
import { createEvent, type Signer, type Event } from "./event.js";
import type { Store } from "./store.js";
import { ActorId, ConversationId, EventId, EventType, Hash } from "./types.js";

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
}
