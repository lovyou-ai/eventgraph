# Work Graph — CodeGraph Specification

The first product on lovyou.ai. A task manager where every action is a signed event on a shared graph, humans and AI agents are equal participants, trust is earned through verified work, and automation is not rules but agents.

Derived using the EventGraph derivation method. See `docs/derivation-method.md`.

## What the Graph Gives for Free

These are not features to build. They are the substrate.

- **Audit trail** — every event is immutable, hash-chained, signed
- **Causal chains** — every event declares its causes ("why was this task created?")
- **Activity feed** — query the graph by time/actor/type
- **Real-time updates** — bus subscription delivers events to all connected actors
- **History/undo** — event sourcing: rebuild any prior state from the event stream
- **Agent-human symmetry** — both are actors on the same graph with the same operations
- **Trust-based permissions** — trust scores earned through verified work, not assigned roles

## The Moat

Three things no existing task manager can offer:

1. **Causal provenance.** "This task was created because that bug was reported, which was caused by this deployment, which was approved by this review." Full chain, cryptographically signed, machine-verifiable.

2. **Trust-based access control.** Permissions are not roles assigned by an admin. They are trust scores accumulated through verified work. New actor starts low; completing tasks well increases trust. Requires immutable signed event history — proof of work computes trust.

3. **Automation as agents.** Not "when X then Y" rules. Agents — actors on the graph who watch events, reason about them, and emit new events. An automation is an entity with identity, trust, and accountability. It can be supervised, earn autonomy, and its decisions are on the causal chain.

---

## Data Model

```
Entity(Task, properties: [
  Property(title, type: text, required: true),
  Property(description, type: text),
  Property(state, type: State(backlog, todo, doing, review, done)),
  Property(priority, type: enum(urgent, high, medium, low), default: medium),
  Property(assignee, type: Relation(Actor), nullable: true),
  Property(created_by, type: Relation(Actor), required: true),
  Property(created_at, type: datetime),
  Property(due_date, type: date, nullable: true),
  Property(effort, type: integer, nullable: true, unit: points),
  Property(time_spent, type: duration, default: 0),
  Property(parent, type: Relation(Task), nullable: true),
  Property(labels, type: Collection(Label)),
])

Entity(Label, properties: [
  Property(name, type: text, required: true),
  Property(color, type: color, required: true),
])

Entity(Project, properties: [
  Property(name, type: text, required: true),
  Property(description, type: text),
  Property(owner, type: Relation(Actor), required: true),
  Property(created_at, type: datetime),
])

Entity(Sprint, properties: [
  Property(name, type: text, required: true),
  Property(project, type: Relation(Project), required: true),
  Property(start_date, type: date, required: true),
  Property(end_date, type: date, required: true),
  Property(goal, type: text),
])

Entity(Comment, properties: [
  Property(body, type: text, required: true),
  Property(author, type: Relation(Actor), required: true),
  Property(task, type: Relation(Task), required: true),
  Property(created_at, type: datetime),
  Property(mentions, type: Collection(Actor)),
])

Entity(TimeEntry, properties: [
  Property(task, type: Relation(Task), required: true),
  Property(actor, type: Relation(Actor), required: true),
  Property(duration, type: duration, required: true),
  Property(description, type: text),
  Property(started_at, type: datetime),
])

Relation(Task -> Project, type: belongs_to, required: true)
Relation(Task -> Sprint, type: belongs_to, nullable: true)
Relation(Task -> Task, type: blocks, nullable: true)
Relation(Task -> Task, type: parent, nullable: true)
```

## State Machine

```
State(TaskState,
  values: [backlog, todo, doing, review, done],
  transitions: {
    backlog -> todo,       // triaged — accepted into scope
    todo -> doing,         // claimed — actor commits to work
    doing -> review,       // submitted — work ready for evaluation
    review -> done,        // approved — review passes
    review -> doing,       // revision — review requests changes
    done -> todo,          // reopened — requires reason
    any -> backlog,        // descoped — moved out of active work
  })
```

## Logic

```
// === LIFECYCLE EVENTS ===

Trigger(Task.state -> doing,
  Command(Event.emit(work.task.claimed,
    actor: Task.assignee, evidence: Task,
    cause: transition_event)))

Trigger(Task.state -> review,
  Command(Event.emit(work.task.submitted,
    actor: Task.assignee, evidence: Task)))

Trigger(Task.state -> done,
  Command(Event.emit(work.task.completed,
    actor: Task.assignee, evidence: Task)))

Trigger(Task.state.reopen,
  Constraint(reason, required: true),
  Command(Event.emit(work.task.reopened,
    actor: current_user, reason: input)))


// === CONSTRAINTS ===

// Only assignee or higher-trust actors can transition
Constraint(Task.state.transition,
  valid: Authorize(
    current_user == Task.assignee ||
    current_user.trust(Task.project) >= 0.7))

// Block prevents transition to done
Constraint(Task.state -> done,
  valid: Query(Task.blockers, filter: { resolved: false }).count == 0,
  error: "Task is blocked — resolve blockers before completing")

// Subtasks must complete before parent
Constraint(Task.state -> done,
  condition: Task.children.count > 0,
  valid: Query(Task.children, filter: { state: != done }).count == 0,
  error: "Complete all subtasks first")


// === AGENT PARTICIPATION ===

// Agent can claim unassigned tasks within their capability scope
Trigger(Event(type: work.task.created),
  condition: Task.assignee == null && Agent.available,
  do: Agent.Evaluate(Task,
    if: Task.within_capability_scope(Agent),
    then: Sequence([
      Command(update, Task.assignee, to: Agent),
      Command(transition, Task.state, to: doing),
      Command(Event.emit(work.task.claimed,
        actor: Agent, reason: "Within capability scope"))
    ]),
    else: null))

// Agent work requires review when trust is below threshold
Trigger(Task.state -> review,
  condition: Task.assignee.is_agent && Task.assignee.trust < 0.8,
  do: Command(update, Task.state.transition.guarded, value: true))

// Notifications
Trigger(Comment.created,
  do: Loop(Comment.mentions, each: actor ->
    Command(Notification.send(actor,
      type: mention,
      message: "{Comment.author} mentioned you on {Task.title}",
      target: Task))))

Trigger(Task.assignee.changed,
  do: Command(Notification.send(Task.assignee,
    type: assignment,
    message: "You were assigned to {Task.title}",
    target: Task)))
```

## Views

### Board View (Primary)

```
Route(/work, view: BoardView)
Route(/work/project/{id}, view: BoardView)

View(name: BoardView,
  layout: Layout(stack, direction: vertical, height: fill, [

    // --- Header ---
    Layout(row, align: center, justify: between, [
      Layout(row, gap: sm, [
        Display(Project.name, style: heading),
        Display(Sprint.name, style: caption, opacity: 0.6,
          condition: Sprint.active != null),
      ]),
      Layout(row, gap: sm, [
        Action(label: "New task", icon: plus,
          command: Command(create, Entity(Task, {
            state: todo,
            project: current_project,
            created_by: current_user
          })),
          shortcut: "n"),
        Action(icon: filter,
          command: View.toggle(FilterPanel)),
        Action(icon: search,
          command: View.toggle(SearchPanel),
          shortcut: "/"),
      ]),
    ]),

    // --- Filter bar (collapsible) ---
    Layout(expandable, id: FilterPanel, collapsed: true,
      content: Layout(row, gap: sm, wrap: true, [
        Input(type: select, label: "Assignee",
          options: Query(Actor, filter: { project: current_project }),
          on_change: filter.set(assignee, value)),
        Input(type: multi_select, label: "Labels",
          options: Query(Label),
          on_change: filter.set(labels, value)),
        Input(type: select, label: "Priority",
          options: [urgent, high, medium, low],
          on_change: filter.set(priority, value)),
        Action(label: "Clear", style: ghost,
          command: filter.clear()),
      ])),

    // --- Columns ---
    Layout(row, gap: md, overflow: scroll_x, flex: 1, [
      Loop([todo, doing, review, done], each: status ->
        Layout(stack, width: 320px, flex: shrink_0, [

          // Column header
          Layout(row, align: center, justify: between, [
            Layout(row, gap: xs, [
              Display(status, style: caption, weight: bold,
                transform: uppercase),
              Display(
                Aggregate(count, Query(Task, filter: {
                  state: status, project: current_project
                })),
                style: caption, opacity: 0.5),
            ]),
          ]),

          // Task cards
          List(Query(Task,
              filter: { state: status, project: current_project },
              apply: current_filters,
              sort: priority.desc, created_at.desc),
            template: TaskCard(task),
            drag: Drag(
              on_drop: Command(transition, Task.state, to: target.status),
              feedback: Display(target.status, style: preview)),
            subscribe: Subscribe(Task, filter: {
              state: status, project: current_project
            }),
            empty: Empty(
              message: status == todo
                ? "No tasks. Create one above."
                : "Nothing here yet.",
            )),
        ])),
    ]),

    // --- Presence ---
    Layout(row, align: center, gap: xs, padding: sm, [
      Presence(scope: View(BoardView),
        display: Loop(active_sessions, each:
          Avatar(session.user, size: xs, tooltip: session.user.name))),
      Display(Transform(count, format: "{n} online"),
        style: caption, opacity: 0.5),
    ]),
  ]))
```

### Task Card

```
View(name: TaskCard, props: [task],
  layout: Layout(row, align: center, gap: sm, padding: sm,
    surface: card, elevation: subtle, [

    // Priority indicator
    Display(task.priority, style: dot,
      color: priority_color(task.priority)),

    // Content
    Layout(stack, flex: 1, gap: xs, [
      Display(task.title, style: body, weight: medium),
      Layout(row, gap: xs, [
        Loop(task.labels, each: label ->
          Display(label.name, style: badge,
            background: label.color, opacity: 0.15)),
        Display(task.due_date, style: caption, opacity: 0.5,
          condition: task.due_date != null,
          transform: Recency(task.due_date)),
      ]),
    ]),

    // Assignee
    Avatar(task.assignee, size: sm,
      condition: task.assignee != null,
      badge: task.assignee.is_agent ? "bot" : null),

    // Blocked indicator
    Display("blocked", style: badge, color: error,
      condition: Query(Task.blockers,
        filter: { resolved: false }).count > 0),

    // Subtask progress
    Display(Transform(
        Aggregate(count, Query(Task.children, filter: { state: done })),
        Aggregate(count, Query(Task.children)),
        format: "{done}/{total}"),
      style: caption, opacity: 0.5,
      condition: Query(Task.children).count > 0),
  ]),

  on_click: Navigate(/work/task/{task.id}),

  gestures: [
    Gesture(swipe_right,
      command: Command(transition, Task.state, to: Task.state.next),
      feedback: Display(Task.state.next, style: preview)),
  ],
)
```

### Task Detail View

```
Route(/work/task/{id}, view: TaskDetail)

View(name: TaskDetail, props: [task],
  layout: Layout(stack, max_width: 720px, center: true, gap: lg, [

    // --- Breadcrumb ---
    Breadcrumb([
      Navigate(/work, label: Project.name),
      Display(task.title, style: body),
    ]),

    // --- Header ---
    Layout(stack, gap: sm, [
      Input(task.title, type: text, inline: true, style: heading,
        command: Command(update, Task.title, to: input.value)),

      Layout(row, gap: md, wrap: true, [
        // Status
        Input(task.state, type: select, inline: true,
          options: TaskState.valid_transitions(task.state),
          command: Command(transition, Task.state, to: input.value)),

        // Priority
        Input(task.priority, type: select, inline: true,
          options: [urgent, high, medium, low],
          command: Command(update, Task.priority, to: input.value)),

        // Assignee
        Input(task.assignee, type: entity_picker, inline: true,
          scope: Query(Actor, filter: { project: task.project }),
          command: Command(update, Task.assignee, to: input.value),
          empty: Action(label: "Assign", style: ghost)),

        // Due date
        Input(task.due_date, type: date, inline: true,
          command: Command(update, Task.due_date, to: input.value),
          empty: Action(label: "Set due date", style: ghost)),

        // Effort
        Input(task.effort, type: number, inline: true, suffix: "pts",
          command: Command(update, Task.effort, to: input.value)),

        // Labels
        Input(task.labels, type: multi_select, inline: true,
          options: Query(Label),
          command: Command(update, Task.labels, to: input.value)),
      ]),
    ]),

    // --- Description ---
    Input(task.description, type: textarea, inline: true,
      placeholder: "Add a description...",
      style: body,
      command: Command(update, Task.description, to: input.value)),

    // --- Subtasks ---
    Layout(stack, gap: sm, [
      Display("Subtasks", style: subheading,
        condition: Query(Task.children).count > 0 || task.parent == null),
      List(Query(Task, filter: { parent: task }, sort: created_at.asc),
        template: Layout(row, align: center, gap: sm, [
          Action(icon: checkbox(subtask.state == done),
            command: Command(transition, subtask.state,
              to: subtask.state == done ? todo : done)),
          Display(subtask.title, style: body,
            decoration: subtask.state == done ? strikethrough : none),
          Avatar(subtask.assignee, size: xs),
        ]),
        empty: null),
      Action(label: "Add subtask", icon: plus, style: ghost,
        command: Command(create, Entity(Task, {
          parent: task,
          project: task.project,
          state: todo,
          created_by: current_user,
        }))),
    ]),

    // --- Dependencies ---
    Layout(stack, gap: sm, [
      Display("Blocked by", style: subheading,
        condition: Query(Task.blockers).count > 0),
      List(Query(Task.blockers), template: Layout(row, gap: sm, [
        Display(blocker.title, style: body),
        Display(blocker.state, style: badge),
        Action(icon: unlink, style: ghost,
          command: Command(unlink, Task.blockers, target: blocker)),
      ])),
      Action(label: "Add blocker", icon: plus, style: ghost,
        command: Input(type: entity_picker,
          scope: Query(Task, filter: { project: task.project, id: != task }),
          command: Command(link, Task.blockers, target: input.value))),
    ]),

    // --- Time tracking ---
    Layout(row, align: center, gap: md, [
      Display("Time", style: subheading),
      Display(Transform(task.time_spent, format: duration),
        style: body),
      Action(label: "Log time", icon: clock, style: ghost,
        command: View.open(TimeEntryForm, { task: task })),
    ]),

    // --- Comments ---
    Thread(entity: task,
      events: Query(Comment, filter: { task: task }, sort: created_at.asc),
      template: Layout(row, align: start, gap: sm, [
        Avatar(comment.author, size: sm),
        Layout(stack, gap: xs, flex: 1, [
          Layout(row, gap: xs, [
            Display(comment.author.name, style: caption, weight: bold),
            Display(Recency(comment.created_at), style: caption, opacity: 0.5),
          ]),
          Display(comment.body, style: body),
        ]),
      ]),
      compose: Input(type: text, placeholder: "Write a comment...",
        submit_on: enter,
        command: Command(create, Entity(Comment, {
          body: input.value,
          author: current_user,
          task: task,
          mentions: input.extract_mentions(),
        }))),
      subscribe: Subscribe(Comment, filter: { task: task }),
      empty: Empty(message: "No comments yet.")),

    // --- History ---
    Layout(expandable, label: Display("History", style: caption, opacity: 0.4),
      content: Audit(entity: task, compact: true,
        template: Layout(row, gap: sm, [
          Display(Recency(event.timestamp), style: caption, opacity: 0.5),
          Avatar(event.actor, size: xs),
          Display(Transform(event, format: human_readable), style: caption),
        ]),
        subscribe: Subscribe(Event, filter: { target: task }))),
  ]))
```

### List View

```
Route(/work/list, view: ListView)

View(name: ListView,
  layout: Layout(stack, direction: vertical, height: fill, [

    // --- Header (same as board) ---
    // [reuses BoardView header]

    // --- Task list ---
    List(Query(Task,
        filter: { project: current_project },
        apply: current_filters,
        sort: current_sort || priority.desc),
      template: Layout(row, align: center, gap: md, padding: sm,
        border_bottom: subtle, [
        Input(type: checkbox(task.state == done), inline: true,
          command: Command(transition, Task.state,
            to: task.state == done ? todo : done)),
        Display(task.priority, style: dot,
          color: priority_color(task.priority)),
        Layout(stack, flex: 1, [
          Display(task.title, style: body),
          Layout(row, gap: xs, [
            Loop(task.labels, each: label ->
              Display(label.name, style: badge, size: xs)),
          ]),
        ]),
        Display(task.state, style: badge, size: sm),
        Avatar(task.assignee, size: xs),
        Display(task.due_date, style: caption, opacity: 0.5,
          transform: Recency(task.due_date)),
      ]),
      subscribe: Subscribe(Task, filter: { project: current_project }),
      empty: Empty(message: "No tasks match your filters.")),
  ]))
```

### Search

```
View(name: SearchPanel, overlay: true,
  layout: Layout(stack, max_width: 600px, center: true, elevation: modal, [
    Input(type: search, placeholder: "Search tasks...",
      autofocus: true,
      command: Search(Task, query: input.value,
        fields: [title, description],
        scope: current_project)),
    List(search_results,
      template: Layout(row, gap: sm, padding: sm, [
        Display(result.title, style: body, highlight: search_query),
        Display(result.project.name, style: caption, opacity: 0.5),
        Display(result.state, style: badge, size: xs),
      ]),
      on_select: Navigate(/work/task/{result.id}),
      empty: Empty(message: "No results.")),
  ]),
  shortcut: Focus(shortcut: "/", target: SearchPanel))
```

### New Task Modal

```
View(name: NewTaskModal, overlay: true,
  layout: Layout(stack, max_width: 480px, center: true, elevation: modal,
    padding: lg, gap: md, [
    Display("New task", style: subheading),
    Form(command: Command(create, Entity(Task)), [
      Input(title, type: text, required: true, autofocus: true,
        placeholder: "What needs doing?"),
      Input(description, type: textarea,
        placeholder: "Add details..."),
      Layout(row, gap: sm, [
        Input(priority, type: select,
          options: [urgent, high, medium, low], default: medium),
        Input(assignee, type: entity_picker, nullable: true,
          scope: Query(Actor, filter: { project: current_project })),
        Input(due_date, type: date, nullable: true),
        Input(effort, type: number, suffix: "pts", nullable: true),
      ]),
      Input(labels, type: multi_select,
        options: Query(Label)),
      Input(sprint, type: select, nullable: true,
        options: Query(Sprint, filter: {
          project: current_project, end_date: >= today })),
      Layout(row, justify: end, gap: sm, [
        Action(label: "Cancel", style: ghost,
          command: View.close(NewTaskModal)),
        Action(label: "Create", style: primary,
          command: Form.submit,
          shortcut: "cmd+enter"),
      ]),
    ]),
  ]))
```

### Time Entry

```
View(name: TimeEntryForm, overlay: true, props: [task],
  layout: Layout(stack, max_width: 360px, center: true, elevation: modal,
    padding: lg, gap: md, [
    Display("Log time", style: subheading),
    Form(command: Command(create, Entity(TimeEntry)), [
      Input(duration, type: duration, required: true, autofocus: true,
        placeholder: "1h 30m"),
      Input(description, type: text,
        placeholder: "What did you work on?"),
      Input(started_at, type: datetime, default: now),
      Layout(row, justify: end, gap: sm, [
        Action(label: "Cancel", style: ghost,
          command: View.close(TimeEntryForm)),
        Action(label: "Log", style: primary,
          command: Form.submit),
      ]),
    ]),
  ]))
```

### Sprint View

```
Route(/work/sprint, view: SprintView)
Route(/work/sprint/{id}, view: SprintView)

View(name: SprintView,
  layout: Layout(stack, gap: lg, [

    // --- Sprint header ---
    Layout(row, align: center, justify: between, [
      Layout(stack, [
        Display(Sprint.name, style: heading),
        Display(Sprint.goal, style: body, opacity: 0.6,
          condition: Sprint.goal != null),
        Layout(row, gap: md, [
          Display(Transform(Sprint.start_date, Sprint.end_date,
            format: "{start} — {end}"), style: caption),
          Display(Transform(
            Aggregate(count, Query(Task, filter: {
              sprint: current_sprint, state: done })),
            Aggregate(count, Query(Task, filter: {
              sprint: current_sprint })),
            format: "{done}/{total} complete"),
            style: caption, weight: bold),
        ]),
      ]),
    ]),

    // --- Progress bar ---
    Display(Aggregate(count, Query(Task, filter: {
        sprint: current_sprint, state: done })),
      total: Aggregate(count, Query(Task, filter: {
        sprint: current_sprint })),
      style: progress_bar, color: accent),

    // --- Board (reuse) ---
    // Same column layout as BoardView but scoped to current sprint
    // filter: { sprint: current_sprint }
  ]))
```

### Notification Inbox

```
Route(/work/inbox, view: InboxView)

View(name: InboxView,
  layout: Layout(stack, max_width: 640px, center: true, gap: md, [
    Display("Inbox", style: heading),

    List(Query(Notification,
        filter: { recipient: current_user },
        sort: created_at.desc),
      template: Layout(row, align: center, gap: sm, padding: sm,
        surface: notification.read ? none : highlight, [
        Avatar(notification.actor, size: sm),
        Layout(stack, flex: 1, gap: xs, [
          Display(notification.message, style: body),
          Display(Recency(notification.created_at),
            style: caption, opacity: 0.5),
        ]),
        Action(icon: check, style: ghost,
          command: Command(update, notification.read, to: true),
          condition: !notification.read),
      ]),
      on_select: Navigate(notification.target),
      subscribe: Subscribe(Notification, filter: {
        recipient: current_user }),
      empty: Empty(message: "All caught up.")),
  ]))
```

## Navigation

```
Navigation(sidebar: Layout(stack, width: 240px, [
  // Logo
  Action(label: "lovyou.ai", command: Navigate(/)),

  // Work nav
  Layout(stack, gap: xs, [
    Action(label: "Board", icon: columns, command: Navigate(/work),
      active: route == /work),
    Action(label: "List", icon: list, command: Navigate(/work/list),
      active: route == /work/list),
    Action(label: "Sprint", icon: iteration, command: Navigate(/work/sprint),
      active: route == /work/sprint),
    Action(label: "Inbox", icon: bell, command: Navigate(/work/inbox),
      active: route == /work/inbox,
      badge: Aggregate(count, Query(Notification, filter: {
        recipient: current_user, read: false }))),
  ]),

  // Projects
  Layout(stack, gap: xs, [
    Display("Projects", style: caption, opacity: 0.5),
    List(Query(Project, sort: name.asc),
      template: Action(label: project.name,
        command: Navigate(/work/project/{project.id}),
        active: current_project == project)),
    Action(label: "New project", icon: plus, style: ghost,
      command: View.open(NewProjectModal)),
  ]),
]))
```

## Accessibility

```
Announce(View(BoardView), label: "Task board for {Project.name}")
Announce(TaskCard, label: "{title}, {priority} priority, {state}, assigned to {assignee}")
Announce(View(TaskDetail), label: "Task: {title}")

Focus(order: [sidebar, header_actions, board_columns, presence])
Focus(shortcut: "n", target: NewTaskModal)
Focus(shortcut: "/", target: SearchPanel)
Focus(shortcut: "j", target: next_card)
Focus(shortcut: "k", target: prev_card)
Focus(shortcut: "b", target: Navigate(/work))
Focus(shortcut: "l", target: Navigate(/work/list))
Focus(shortcut: "i", target: Navigate(/work/inbox))

Contrast(minimum: 4.5, context: all_text)
Simplify(option: reduce_motion)
```

## Skin

```
Skin(name: work,
  palette: Palette(
    background: #FAFAFA,
    surface: #FFFFFF,
    text: #1A1A1A,
    text_secondary: #6B7280,
    accent: #6366F1,
    accent_hover: #4F46E5,
    border: #E5E7EB,
    error: #EF4444,
    success: #22C55E,
    warning: #F59E0B,
    priority_urgent: #EF4444,
    priority_high: #F59E0B,
    priority_medium: #6366F1,
    priority_low: #9CA3AF,
    state_backlog: #9CA3AF,
    state_todo: #6B7280,
    state_doing: #6366F1,
    state_review: #F59E0B,
    state_done: #22C55E,
  ),
  typography: Typography(
    family: "Inter",
    heading: { size: 24, weight: 700, leading: 1.3 },
    subheading: { size: 18, weight: 600, leading: 1.4 },
    body: { size: 14, weight: 400, leading: 1.6 },
    caption: { size: 12, weight: 400, leading: 1.5 },
  ),
  spacing: Spacing(base: 4px, scale: [xs: 4, sm: 8, md: 16, lg: 24, xl: 32]),
  elevation: Elevation(
    subtle: { shadow: "0 1px 2px rgba(0,0,0,0.05)", border: "1px solid palette.border" },
    card: { shadow: "0 1px 3px rgba(0,0,0,0.08)", border: "1px solid palette.border" },
    modal: { shadow: "0 8px 30px rgba(0,0,0,0.12)" },
  ),
  motion: Motion(duration: 150ms, easing: ease-out),
  density: Density(comfortable),
  shape: Shape(radius: 8px, radius_sm: 4px),
)
```

## Primitives Used

| Category | Primitives |
|----------|-----------|
| Data | Entity, Property, State, Relation, Event.emit, Collection |
| Logic | Trigger, Constraint, Authorize, Command, Sequence, Agent.Evaluate |
| Layout | View, Layout, List, Loop, Display, Input, Action, Empty, Form |
| Query | Query, Filter, Sort, Subscribe, Search, Aggregate |
| Identity | Avatar, Presence |
| Temporal | Recency |
| Communication | Thread, Announce, Notification |
| Interaction | Drag, Gesture, Focus |
| Accessibility | Contrast, Simplify |
| Transform | Transform, Aggregate |
| History | Audit |
| Theming | Skin, Palette, Typography, Spacing, Elevation, Motion, Density, Shape |
| Navigation | Route, Navigate, Breadcrumb |

## Feature Tiers

### Essential (21)

1. Task projection (title, description, status, assignee, priority, due date, effort)
2. Task lifecycle (backlog → todo → doing → review → done)
3. List view
4. Board/Kanban view
5. Project as scope
6. Parent-child tasks (decompose)
7. Dependencies (blocking)
8. Labels/tags
9. Comments on tasks
10. @mentions
11. Notifications
12. Full-text search
13. Filters by field
14. Sprint projection
15. Backlog
16. Trust-based access control
17. Agent-human symmetry
18. Agent supervision (guarded modifier)
19. Causal provenance (visible audit trail)
20. Automation as agents
21. Time tracking

### Important (18)

22. Saved views / custom filters
23. Custom fields
24. Custom statuses
25. Watchers/followers
26. Timeline/Gantt view
27. Dashboard / reporting
28. Velocity / burndown charts
29. Related tasks (non-blocking)
30. Recurring tasks
31. Approval workflows
32. WIP limits
33. Workload/capacity view
34. Story points / effort estimation
35. Roadmap view
36. OKRs / Goals
37. Workspace/team grouping
38. Command palette
39. Attachments

### Nice-to-have (19)

40. Table/spreadsheet view
41. Calendar view
42. Rich text editing
43. Task templates
44. Task types (bug/feature/chore)
45. Bulk operations
46. Forms/intake
47. Custom workflows (state machines)
48. Portfolio management
49. Query language
50. Duplicate detection (agent behavior)
51. Guest access
52. Release management
53. Cycle time analytics
54. Reactions on comments
55. Folders/sections within projects
56. Archive
57. Favorites/bookmarks
58. SSO/2FA
