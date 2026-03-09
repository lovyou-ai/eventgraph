# Code Graph Specification Examples

Code Graph primitives are a **declarative specification language**, not components to be compiled. Every atom — Entity, View, Layout, List, Query, Command — is a defined primitive with documented semantics. A coding agent reads this specification and emits platform-native code for whatever target is required: React, SwiftUI, a terminal UI, or anything else.

The spec describes **what** the application is. The agent decides **how** to build it.

These three examples progress from trivial to production-grade, demonstrating how the 61 primitives compose into complete application specifications.

---

## Example 1: Counter

The simplest possible app — a number that goes up and down.

**Primitives used:** Entity, Property, View, Layout, Display, Action, Command.

```
Entity(Counter, properties: [
  Property(value, type: integer, default: 0)
])

View(name: Counter,
  layout: Layout(row, align: center, gap: lg, [
    Action(label: "−", command: Command(update, Counter.value, delta: -1)),
    Display(Counter.value, style: heading),
    Action(label: "+", command: Command(update, Counter.value, delta: +1)),
  ]))
```

Six lines. One entity, one view. The agent reads this and knows: there is a counter with an integer value, it starts at zero, and there is a row with a decrement button, the current value displayed as a heading, and an increment button. That is the entire application.

---

## Example 2: Chat

A real-time chat room with message history, composition, and presence.

**Primitives used:** Entity, Property, Relation, View, Layout, List, Query, Display, Input, Action, Command, Avatar, Presence, Subscribe, Recency, Announce, Focus, Empty, Loop, Transform.

```
Entity(Message, properties: [
  Property(text, type: text, required: true),
  Property(author, type: Relation(Agent | Human), required: true),
  Property(sent_at, type: datetime)
])

View(name: Chat,
  layout: Layout(stack, direction: vertical, height: fill, [

    // Messages
    List(Query(Message, sort: sent_at.asc),
      template: Layout(row, align: start, gap: sm, [
        Avatar(message.author, size: sm),
        Layout(stack, gap: xs, [
          Display(message.author.name, style: caption, weight: bold),
          Display(message.text, style: body),
          Display(Recency(message.sent_at), style: caption, opacity: 0.5),
        ])
      ]),
      subscribe: Subscribe(Message, live: true),
      empty: Empty(message: "No messages yet. Say hello.")),

    // Compose
    Layout(row, gap: sm, [
      Input(type: text, placeholder: "Type a message...",
        submit_on: enter,
        command: Command(create, Entity(Message, {
          text: input.value,
          author: current_user
        }))),
    ]),

    // Who's here
    Presence(scope: View(Chat),
      display: Layout(row, [
        Loop(active_sessions, each: Avatar(session.user, size: xs)),
        Display(Transform(count, format: "{n} online"))
      ])),
  ]))

// Accessibility
Announce(View(Chat), label: "Chat room with {count} messages")
Focus(order: [message_list, compose_input])
Focus(shortcut: "/", target: compose_input)
```

This spec tells the agent everything it needs: the data model (messages with text, author, timestamp), the live subscription for real-time updates, the message list with avatars and relative timestamps, the compose input that creates messages on enter, presence tracking for active users, and keyboard accessibility. The agent builds all of it for the target platform.

---

## Example 3: Task Manager

The canonical example. A complete task management application with state machines, agent participation, drag-and-drop, gestures, audit trails, accessibility, and visual theming.

**Primitives used:** Entity, Property, State, Relation, Trigger, Constraint, Authorize, Command, Sequence, View, Layout, List, Query, Display, Input, Action, Avatar, Presence, Loop, Empty, Drag, Gesture, Confirmation, Thread, Audit, Transform, Announce, Focus, Contrast, Simplify, Skin, Palette, Typography, Spacing, Elevation, Motion, Density, Shape, Subscribe, Event.emit.

```
// === DATA ===

Entity(Task, properties: [
  Property(title, type: text, required: true),
  Property(state, type: State(todo, doing, done)),
  Property(owner, type: Relation(Agent | Human), required: true),
  Property(created_by, type: Relation(Agent | Human)),
  Property(created_at, type: datetime)
])

State(TaskState,
  values: [todo, doing, done],
  transitions: {
    todo -> doing,    // commitment — signed event
    doing -> done,    // completion — signed event with evidence
    done -> todo      // reopen — requires reason
  })


// === LOGIC ===

Trigger(Task.state -> doing,
  Command(Event.emit(commitment, actor: owner, evidence: Task)))

Trigger(Task.state -> done,
  Command(Event.emit(completion, actor: owner, evidence: Task)))

Trigger(Task.state.reopen,
  Constraint(reason, required: true),
  Command(Event.emit(reopen, actor: current_user, reason: Input)))

Constraint(Task.owner.transition,
  valid: Authorize(current_user == owner || current_user.authority >= recommended))


// === SINGLE VIEW ===

View(name: Everything,
  layout: Layout(stack, direction: vertical, max_width: 640px, center: true),

  // --- Create ---
  header: Layout(row, [
    Input(title, type: text, placeholder: "What needs doing?",
      style: minimal,
      submit_on: enter,
      command: Command(create, Entity(Task, {
        title: input.value,
        state: todo,
        owner: current_user,
        created_by: current_user
      }))),
  ]),

  // --- Group by state ---
  content: Loop([todo, doing, done], each: state ->
    Layout(stack, [
      Display(state, style: caption, opacity: 0.5),

      List(Query(Task, filter: { state: state }, sort: created_at.desc),
        template: TaskCard(task),
        drag: Drag(on_drop: Command(transition, Task.state, to: target.state)),
        empty: Empty(
          condition: state == todo,
          message: "Nothing to do. Enjoy the quiet.",
          else: null
        )),
    ])),

  // --- Presence ---
  footer: Presence(scope: View(Everything),
    display: Layout(row, Loop(active_sessions, each: Avatar(session.user, size: xs))),
    label: Display(Transform(count, format: "{n} here")))
)


// === TASK CARD ===

View(name: TaskCard, props: [task],
  collapsed: Layout(row, align: center, [
    Drag(handle: true),
    Display(task.title, style: body),
    Avatar(task.owner, size: xs),
    Action(
      icon: next_state_icon(task.state),
      command: Command(transition, Task.state.next),
      announce: "Move to {next_state}")
  ]),

  expanded: Layout(stack, [
    Input(task.title, type: text, inline: true, style: subheading),

    Input(task.owner, type: entity_picker, inline: true,
      scope: Query(Agent | Human, filter: team.current)),

    Thread(entity: task,
      events: Query(Event, filter: { target: task, type: comment }, sort: time.asc),
      compose: Input(type: text, placeholder: "Say something...", minimal: true),
      submit: Command(Event.emit(comment, target: task))),

    Layout(expandable, label: Display("History", style: caption, opacity: 0.4),
      content: Audit(entity: task, compact: true,
        template: Layout(row, [
          Display(event.timestamp, style: caption),
          Avatar(event.actor, size: xs),
          Display(Transform(event, format: human_readable), style: caption)
        ])))
  ]),

  toggle: Action(type: tap, command: View.toggle(collapsed, expanded)),

  gestures: [
    Gesture(swipe_right, command: Command(transition, Task.state.next),
      feedback: Display(next_state, style: preview)),
    Gesture(swipe_left,
      condition: task.state == todo,
      command: Confirmation("Remove this task?",
        confirm: Command(tombstone, task)))
  ]
)


// === AGENT PARTICIPATION ===

Trigger(Event(type: task_created, state: todo),
  condition: Agent.available && Agent.authority.includes(task_management),
  do: Agent.Evaluate(task,
    if: suitable_for_agent,
    then: Sequence([
      Command(transition, Task.owner, to: agent),
      Command(transition, Task.state, to: doing),
      Command(Event.emit(claim, actor: agent, reason: "Within my capability scope"))
    ]),
    else: null
  ))


// === ACCESSIBILITY ===

Announce(View(Everything), label: "Task board with {count} tasks")
Announce(TaskCard, label: "{title}, {state}, owned by {owner}")
Focus(order: [create_input, task_cards_by_state, presence])
Focus(shortcut: "/", target: create_input)
Focus(shortcut: "j/k", target: next/prev_card)
Contrast(minimum: 4.5, context: all_text)
Simplify(option: reduce_motion)


// === SKIN ===

Skin(name: notebook,
  palette: Palette(
    background: #FDFCFA,
    surface: #F5F0EB,
    text: #2C2C2C,
    text_secondary: #8B8685,
    accent: #C4956A,
    border: #E8E2DC,
    state_todo: #8B8685,
    state_doing: #C4956A,
    state_done: #7A9B7E,
  ),
  typography: Typography(
    family: "Inter",
    body: { size: 15, weight: 400, leading: 1.6 },
    subheading: { size: 17, weight: 500, leading: 1.4 },
    caption: { size: 13, weight: 400, leading: 1.5 },
  ),
  spacing: Spacing(base: 4px, default: lg),
  elevation: Elevation(flat, border: 1px solid palette.border),
  motion: Motion(gentle, duration: 250ms, easing: ease-in-out),
  density: Density(relaxed),
  shape: Shape(radius: 8px)
)
```

This single specification defines data, state machines, business rules, event emission, authorization, UI layout, drag-and-drop, gestures, threading, audit history, agent autonomy, accessibility, and visual theming. The agent reads it and produces a working application for any platform.

---

## Primitive Reference

All 61 Code Graph primitives, grouped by category.

### Data (6)

| Primitive | Description |
|-----------|-------------|
| **Entity** | A domain object with typed properties |
| **Property** | A typed field on an entity |
| **Relation** | A typed reference between entities |
| **State** | A finite state machine with enforced transitions |
| **Event.emit** | Emit a signed event into the event graph |
| **Tombstone** | Soft-delete marker for an entity |

### Logic (7)

| Primitive | Description |
|-----------|-------------|
| **Trigger** | Reacts to a state change or event by executing commands |
| **Constraint** | A validation rule that must be satisfied |
| **Authorize** | An authorization predicate against the current user |
| **Command** | A mutation instruction (create, update, transition, delete) |
| **Sequence** | An ordered list of commands executed atomically |
| **Confirmation** | Requires user confirmation before executing a command |
| **Agent.Evaluate** | Invokes agent reasoning with conditional outcomes |

### Layout (8)

| Primitive | Description |
|-----------|-------------|
| **View** | A named, composable UI surface |
| **Layout** | A spatial container (row, stack, grid, expandable) |
| **List** | A data-bound repeating container |
| **Loop** | Iterates over a static collection, rendering each item |
| **Display** | Renders a value with a given style |
| **Input** | A data-entry field bound to a property or command |
| **Action** | A user-triggerable interaction (button, tap, icon) |
| **Empty** | Placeholder content shown when a list has no items |

### Query (4)

| Primitive | Description |
|-----------|-------------|
| **Query** | Retrieves entities with filtering, sorting, and pagination |
| **Filter** | A predicate applied to narrow query results |
| **Sort** | An ordering directive on a query |
| **Subscribe** | Registers a live subscription for real-time updates |

### Identity (3)

| Primitive | Description |
|-----------|-------------|
| **Avatar** | Displays a user or agent's visual identity |
| **Presence** | Shows who is currently active in a scope |
| **Session** | Represents an active user connection |

### Temporal (2)

| Primitive | Description |
|-----------|-------------|
| **Recency** | Displays a timestamp as relative time ("3 min ago") |
| **Timestamp** | Displays an absolute date or time value |

### Communication (3)

| Primitive | Description |
|-----------|-------------|
| **Thread** | A comment thread attached to an entity |
| **Announce** | Declares an accessibility announcement for a view or component |
| **Notification** | Pushes an alert or message to a user |

### Interaction (3)

| Primitive | Description |
|-----------|-------------|
| **Drag** | Enables drag-and-drop on a list item or handle |
| **Gesture** | Binds a touch or pointer gesture to a command |
| **Shortcut** | Binds a keyboard shortcut to a command |

### Accessibility (4)

| Primitive | Description |
|-----------|-------------|
| **Focus** | Declares focus order and keyboard shortcuts for a view |
| **Contrast** | Enforces minimum contrast ratios for text |
| **Simplify** | Provides reduced-motion or simplified alternatives |
| **Aria** | Maps semantic roles to accessibility tree nodes |

### Transform (3)

| Primitive | Description |
|-----------|-------------|
| **Transform** | Formats a value for display (count, date, human-readable) |
| **Compute** | Derives a value from other properties |
| **Aggregate** | Computes a summary over a collection (count, sum, avg) |

### History (2)

| Primitive | Description |
|-----------|-------------|
| **Audit** | Renders the event history of an entity |
| **Diff** | Shows what changed between two states |

### Theming (10)

| Primitive | Description |
|-----------|-------------|
| **Skin** | A named visual theme applied to the entire application |
| **Palette** | A set of named colors |
| **Typography** | Font family, sizes, weights, and leading |
| **Spacing** | Base unit and scale for margins and padding |
| **Elevation** | Shadow or border treatment for depth |
| **Motion** | Animation duration, easing, and intensity |
| **Density** | Compact, default, or relaxed spacing mode |
| **Shape** | Border radius and geometric treatment |
| **Icon** | A named icon from a declared icon set |
| **Illustration** | A decorative or informational image |

### Navigation (4)

| Primitive | Description |
|-----------|-------------|
| **Route** | Maps a URL path to a view |
| **Navigate** | A command that changes the current route |
| **Breadcrumb** | Displays the current location in a hierarchy |
| **Tab** | A switchable section within a view |

### Error (2)

| Primitive | Description |
|-----------|-------------|
| **Error** | Displays an error state with recovery options |
| **Retry** | A command that re-attempts a failed operation |

---

## What the Agent Does

A coding agent reads this specification and understands the application at every layer:

1. **Data model.** Entity, Property, Relation, and State define what exists, what it contains, and how it transitions. The agent creates database schemas, type definitions, and state machine implementations.

2. **Business logic.** Trigger, Constraint, Authorize, and Command define what happens, when, and who is allowed. The agent generates event handlers, validation logic, authorization checks, and mutation functions.

3. **UI composition.** View, Layout, List, Loop, Display, Input, and Action define what the user sees and interacts with. The agent produces component trees, data bindings, and interaction handlers.

4. **Accessibility requirements.** Announce, Focus, Contrast, and Simplify define how the application serves all users. The agent emits ARIA attributes, focus management, contrast-compliant styles, and reduced-motion alternatives.

5. **Visual identity.** Skin, Palette, Typography, Spacing, Elevation, Motion, Density, and Shape define how the application looks and feels. The agent generates design tokens, CSS custom properties, or platform-native style constants.

6. **Platform-native output.** The agent emits code for the target: React components, SwiftUI views, Jetpack Compose, terminal UI, or anything else. The specification is platform-independent. The same spec produces a web app, a native app, or a CLI.

The specification says **what**. The agent decides **how**. The primitives are the contract between the two.
