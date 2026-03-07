# Compositions

Named compositions of primitives that form the **grammar** of each product layer. Each composition is a higher-level operation that combines multiple primitives into a single, comprehensible action — the vocabulary developers use when building on that product graph.

## Method

The same derivation method used for the social grammar (Post 35) applies to every layer:

1. **Identify base operations** — What are the fundamental things you do in this product domain?
2. **Identify semantic dimensions** — What axes differentiate one operation from another?
3. **Decompose** — Apply dimensions to base operations to produce the full set
4. **Named functions** — Identify common multi-operation patterns and name them

## Structure

Each layer's composition file contains:

- **Operations** — The core grammar (like the social grammar's 15 operations)
- **Modifiers** — Cross-cutting concerns that modify any operation (like Transient, Nascent, Conditional)
- **Named Functions** — Common compositions of operations (like Recommend = Propagate + Channel)
- **Mapping** — How each operation maps to underlying primitives

## Relationship to Other Docs

- `docs/grammar.md` — The original social grammar (Layer 0 infrastructure). Compositions here BUILD ON those 15 operations by combining them with layer-specific primitives.
- `docs/primitives.md` — The raw building blocks. Compositions combine them into usable vocabulary.
- `docs/product-layers.md` — The product vision for each layer. Compositions are the API.
- `docs/layers/` — The derivations that produce the primitives. Compositions derive from the same method applied at the product level.

## Index

| # | Layer | Grammar | Operations | Modifiers | Functions |
|---|-------|---------|------------|-----------|-----------|
| 1 | Agency | Work Grammar | 12 | 3 | 6 |
| 2 | Exchange | Market Grammar | 14 | 3 | 7 |
| 3 | Society | Social Grammar | 15 | 3 | 8 |
| 4 | Legal | Justice Grammar | 12 | 2 | 6 |
| 5 | Technology | Build Grammar | 12 | 3 | 5 |
| 6 | Information | Knowledge Grammar | 12 | 2 | 6 |
| 7 | Ethics | Alignment Grammar | 10 | 2 | 5 |
| 8 | Identity | Identity Grammar | 10 | 2 | 5 |
| 9 | Relationship | Bond Grammar | 10 | 2 | 5 |
| 10 | Community | Belonging Grammar | 10 | 2 | 5 |
| 11 | Culture | Meaning Grammar | 10 | 2 | 5 |
| 12 | Emergence | Evolution Grammar | 10 | 2 | 4 |
| 13 | Existence | Being Grammar | 8 | 1 | 3 |

The social grammar (Layer 3, `docs/grammar.md`) is the infrastructure-level grammar. All other grammars compose its operations with layer-specific primitives.

## Reference

- `docs/grammar.md` — The social grammar (15 operations + 3 modifiers + 8 functions)
- `docs/derivation-method.md` — The method for deriving complete sets
- `docs/primitives.md` — All 201 primitives
- `docs/product-layers.md` — Product graph descriptions
