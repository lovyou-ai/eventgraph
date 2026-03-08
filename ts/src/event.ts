import { createHash, randomBytes } from "node:crypto";
import {
  ActorId, ConversationId, EventId, EventType, Hash, NonEmpty, Signature,
} from "./types.js";

// ── Signer ──────────────────────────────────────────────────────────────

export interface Signer {
  sign(data: Uint8Array): Signature;
}

export class NoopSigner implements Signer {
  sign(_data: Uint8Array): Signature {
    return new Signature(new Uint8Array(64));
  }
}

// ── Canonical form ──────────────────────────────────────────────────────

export function canonicalContentJson(content: Record<string, unknown>): string {
  // Omit null/undefined values, sort keys
  const filtered: Record<string, unknown> = {};
  for (const key of Object.keys(content).sort()) {
    if (content[key] != null) filtered[key] = content[key];
  }
  return JSON.stringify(filtered, Object.keys(filtered).sort());
}

export function canonicalForm(
  version: number,
  prevHash: string,
  causes: string[],
  eventId: string,
  eventType: string,
  source: string,
  conversationId: string,
  timestampNanos: number,
  contentJson: string,
): string {
  const sortedCauses = [...causes].sort().join(",");
  return `${version}|${prevHash}|${sortedCauses}|${eventId}|${eventType}|${source}|${conversationId}|${timestampNanos}|${contentJson}`;
}

export function computeHash(canonical: string): Hash {
  const digest = createHash("sha256").update(canonical, "utf-8").digest("hex");
  return new Hash(digest);
}

// ── Event ───────────────────────────────────────────────────────────────

export class Event {
  constructor(
    readonly version: number,
    readonly id: EventId,
    readonly type: EventType,
    readonly timestampNanos: number,
    readonly source: ActorId,
    private readonly _content: Record<string, unknown>,
    readonly causes: NonEmpty<EventId>,
    readonly conversationId: ConversationId,
    readonly hash: Hash,
    readonly prevHash: Hash,
    readonly signature: Signature,
  ) {
    Object.freeze(this);
  }

  get content(): Record<string, unknown> {
    return { ...this._content };
  }
}

// ── UUID v7 generation ──────────────────────────────────────────────────

export function newEventId(): EventId {
  const ms = Date.now();
  const b = new Uint8Array(16);

  // Timestamp: 48 bits
  const view = new DataView(b.buffer);
  view.setUint16(0, (ms / 0x100000000) & 0xFFFF);
  view.setUint32(2, ms & 0xFFFFFFFF);

  // Random fill
  const rand = randomBytes(10);
  b.set(rand, 6);

  // Version 7
  b[6] = (b[6] & 0x0F) | 0x70;
  // Variant 10xx
  b[8] = (b[8] & 0x3F) | 0x80;

  const hex = (i: number) => b[i].toString(16).padStart(2, "0");
  const s = `${hex(0)}${hex(1)}${hex(2)}${hex(3)}-${hex(4)}${hex(5)}-${hex(6)}${hex(7)}-${hex(8)}${hex(9)}-${hex(10)}${hex(11)}${hex(12)}${hex(13)}${hex(14)}${hex(15)}`;
  return new EventId(s);
}

// ── Event factories ─────────────────────────────────────────────────────

export function createEvent(
  eventType: EventType,
  source: ActorId,
  content: Record<string, unknown>,
  causes: EventId[],
  conversationId: ConversationId,
  prevHash: Hash,
  signer: Signer,
  version = 1,
): Event {
  const eventId = newEventId();
  const timestampNanos = Date.now() * 1_000_000;
  const contentJson = canonicalContentJson(content);

  const canon = canonicalForm(
    version, prevHash.value,
    causes.map((c) => c.value),
    eventId.value, eventType.value,
    source.value, conversationId.value,
    timestampNanos, contentJson,
  );

  const hash = computeHash(canon);
  const sig = signer.sign(new TextEncoder().encode(canon));

  return new Event(
    version, eventId, eventType, timestampNanos,
    source, content, NonEmpty.of(causes),
    conversationId, hash, prevHash, sig,
  );
}

export function createBootstrap(
  source: ActorId,
  signer: Signer,
  version = 1,
): Event {
  const eventId = newEventId();
  const timestampNanos = Date.now() * 1_000_000;
  const conversationId = new ConversationId(`conv_${source.value}`);

  const content: Record<string, unknown> = {
    ActorID: source.value,
    ChainGenesis: Hash.zero().value,
    Timestamp: new Date().toISOString().replace(/\.\d{3}Z$/, "Z"),
  };
  const contentJson = canonicalContentJson(content);

  const canon = canonicalForm(
    version, "",
    [],
    eventId.value, "system.bootstrapped",
    source.value, conversationId.value,
    timestampNanos, contentJson,
  );

  const hash = computeHash(canon);
  const sig = signer.sign(new TextEncoder().encode(canon));

  return new Event(
    version, eventId, new EventType("system.bootstrapped"),
    timestampNanos, source, content,
    NonEmpty.of([eventId]),
    conversationId, hash, Hash.zero(), sig,
  );
}
