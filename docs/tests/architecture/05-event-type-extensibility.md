# Architecture Test 5: Event Type Extensibility

Verify that custom event types can be registered with schemas, and that the EventTypeRegistry validates content correctly.

## Purpose

SDK users need to define custom event types for their domains. This test verifies that the EventTypeRegistry is a genuine extension point — not just a fixed list of known types.

## Setup

```
registry: EventTypeRegistry (pre-loaded with built-in types)
custom_type: "custom.widget_created" with schema {
    name: string (required)
    color: string (required)
    weight: Score (optional)
}
```

## Test Cases

### TC-5.1: Register Custom Event Type

**Input:** Register "custom.widget_created" with its schema.
**Assertions:**
- Type appears in registry
- Schema is queryable: `registry.Schema("custom.widget_created")` returns the schema
- Built-in types are unaffected

### TC-5.2: Valid Custom Event

**Input:** Create event with type "custom.widget_created" and valid content.
**Assertions:**
- EventFactory creates the event without error
- Event is stored in Store
- Content matches the registered schema

### TC-5.3: Invalid Custom Event — Missing Required Field

**Input:** Create event with type "custom.widget_created" but missing "color" field.
**Assertions:**
- EventFactory returns `Err(ValidationError)` with field name
- Event is NOT stored

### TC-5.4: Invalid Custom Event — Wrong Type

**Input:** Create event with "weight" as a string instead of Score.
**Assertions:**
- `Err(ValidationError)` identifying the type mismatch
- Event is NOT stored

### TC-5.5: Unregistered Event Type

**Input:** Create event with type "unknown.something" (not registered).
**Assertions:**
- EventFactory returns `Err(ValidationError.UnknownType)`
- Event is NOT stored

### TC-5.6: Event Type Namespace Enforcement

**Input:** Attempt to register "trust.custom" (collides with built-in "trust.*" namespace).
**Assertions:**
- Registration fails with `Err(ValidationError.ReservedNamespace)`
- Built-in types unaffected

### TC-5.7: Custom Type Subscriptions

**Input:** Register primitive subscribing to "custom.*". Emit "custom.widget_created".
**Assertions:**
- Primitive receives the event
- Event content is correctly typed

### TC-5.8: Schema Evolution

**Input:** Register v1 of custom type, create events. Register v2 with additional optional field.
**Assertions:**
- v1 events still valid and queryable
- v2 events include new field
- Event.Version distinguishes v1 from v2

### TC-5.9: Visitor Pattern Extension

**Input:** Register custom type, then attempt to visit it with EventContentVisitor.
**Assertions:**
- Custom types route to a catch-all visitor method (e.g., `VisitCustom`)
- Or: visitor pattern is extensible (language-dependent)

## Error Cases

| Case | Input | Expected |
|------|-------|----------|
| Empty type name | Register "" | `Err(ValidationError)` |
| Duplicate registration | Register same type twice | `Err(ValidationError.DuplicateType)` |
| Invalid schema | Register with nil/empty schema | `Err(ValidationError)` |
| Nested custom types | Custom type referencing another custom type | Supported if both registered |

## Reference

- `docs/interfaces.md` — EventTypeRegistry, EventFactory, EventContentVisitor
