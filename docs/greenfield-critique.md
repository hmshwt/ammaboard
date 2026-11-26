# **Critique of&#x20;**`kit`**&#x20;/&#x20;**`tuikit`

My overall assessment: **this is a well-reasoned infrastructure project with a coherent architectural thesis, but its biggest risk is that it is architecturally more mature than its product thesis.**

The spec is unusually disciplined. It has clear boundaries, an explicit vocabulary, strong invariants, a good escape hatch, and a thoughtful testing model. The clean-slate exercise converging on the existing architecture is meaningful evidence that the design isn't just accidental complexity.

But there are three separate questions that need to be distinguished:

1. **Is this a good architecture?** Largely yes.
2. **Is this a sufficiently compelling developer product?** Not yet proven.
3. **Can this become a business or strategically valuable asset?** Yes, but probably not through the library alone.

I'll critique it from first principles rather than just reviewing individual types.

# **1. Raisond'être: is the problem important enough?**

## **The core problem statement**

The project says developers repeatedly reinvent:

- Screen management
- Keyboard routing and capture semantics
- Help that drifts from bindings
- Confirm flows
- Cursor and scroll edge cases
- Testing without a real terminal
- Debugging key dispatch

And it packages these into a reusable shell over Bubble Tea.

This is a valid problem. However, I would sharpen the problem statement.

The real pain is probably not:

> "Developers need a better abstraction for screens and panes."

It is:

> **Building a reliable operational terminal application requires repeatedly solving interaction correctness, navigation, discoverability, and testability—and the underlying framework doesn't provide a coherent integrated solution.**

That is a stronger raison d'être because it describes the *outcome* users care about, not the implementation layer you are extracting.

### **The actual pain hierarchy**

| Problem                           | Pain              | Frequency | Your solution             |
| --------------------------------- | ----------------- | --------- | ------------------------- |
| Keybinding collisions             | High              | High      | Excellent                 |
| Contextual action discoverability | High              | High      | Excellent                 |
| Modal/capture correctness         | High              | Medium    | Excellent                 |
| Cursor drift after data changes   | High              | Medium    | Excellent                 |
| Resize/layout bugs                | Medium-high       | Medium    | Excellent                 |
| Confirm flows                     | Medium            | High      | Excellent                 |
| Testing interactive apps          | High              | Medium    | Excellent                 |
| Debugging interaction state       | High              | Medium    | Excellent                 |
| Basic screen composition          | Medium            | High      | Good                      |
| Rendering complex UI              | High in some apps | High      | Deliberately out of scope |
| Async/background orchestration    | High              | High      | Deliberately out of scope |

The project is most differentiated around **interaction correctness and developer feedback**, not around layout.

That distinction matters because layout libraries are commoditized more easily. A reliable interaction substrate with executable contracts, contextual routing, audits, receipts, and test tooling is a more interesting proposition.

## **The strongest value proposition**

I would reposition the thesis slightly:

> **A reliable interaction layer for Bubble Tea applications.**
>
> Declare your app's screens, actions, and interaction rules once. The same declarations drive routing, contextual help, command discovery, validation, debugging, and tests.

This is more compelling than "shell" or "UI kit."

The central value is not the twelve types.

It's the reduction of **interaction drift**.

## **The interaction consistency promise**

One declared interaction model produces:

- Execution
- Discoverability
- Validation
- Debugging
- Test assertions

If these disagree, it is a defect in the kit—not the application's documentation.

That is your most powerful conceptual asset.

# **2. Architectural critique**

## **A. The architecture is coherent—but potentially over-constrained**

The architectural philosophy is internally consistent:

- Structure recurses; behavior doesn't.
- Screens are places.
- Layouts are anonymous.
- Panes own behavior.
- Actions are static declarations.
- Scope is placement.
- Cursor state is anchored to identity.
- The root owns environmental facts.
- Rendering is deterministic.
- Capture semantics are explicit.
- Data remains application-owned.

This coherence is a major strength. It means the API is easier to reason about because the concepts reinforce one another.

However, the architecture is also making a lot of **strong normative decisions** before there is enough external usage evidence.

Examples:

- `esc`, `?`, and `:` are permanently pinned.
- No mouse support.
- No per-item actions.
- No behavior on containers.
- No horizontal scrolling.
- No nested screens.
- No dynamic layout mutation.
- No change hooks.
- No runtime minimum enforcement.
- Palette only shows currently routable actions.
- No search across all commands.
- No depth limits.

Each refusal has a rationale. But a list of individually defensible refusals can collectively narrow the product too much.

### **My recommendation**

Keep the principles, but separate them into three categories:

**Core invariants**\
These should be hard architectural constraints:

- Single owner of terminal geometry
- Deterministic rendering
- Static enumerable actions
- One routing resolver
- Capture does not leak input
- Cursor targets use identity
- Domain data remains external
- Testing and production use the same routing semantics

**Opinionated defaults**\
These should be changeable:

- Escape semantics
- Default bindings
- Screen navigation keys
- Confirm grammar
- Scroll behavior
- Debug key
- Theme defaults

**Current scope limitations**\
These should be acknowledged as limitations rather than design principles:

- Mouse
- Horizontal scrolling
- Rich editing
- Per-item action presentation
- Dynamic layouts
- Complex async orchestration

This would preserve the architecture without turning every present limitation into an ideology.

# **3. The biggest architectural gap: lifecycle and state ownership**

The spec is clear about ownership at a conceptual level, but less complete about lifecycle behavior.

For example:

- What happens when a screen is removed while it is on the stack?
- What happens when a pane's backing data is replaced?
- What happens when a modal refers to an item that disappears?
- What happens when an action changes the active screen?
- What happens when an action causes the pane tree to change?
- What happens when focus points to a pane that no longer exists?
- What is the behavior when the app has zero screens?
- What happens when screen registration order changes?
- What happens to cursor state when the same pane instance is reused across screens?

Some of these may be addressed in implementation, but the spec's invariants don't yet define them adequately.

### **Particularly important: identity versus lifetime**

The cursor identity model is good:

> Items carry identity; panes carry behavior.

But there are multiple identities in the system:

- Screen identity
- Pane identity
- Item identity
- Layout diagnostic identity
- Action identity
- Stack-entry identity

The project needs a formal identity and lifecycle model.

For example, is a pane identified by its name globally or only within a screen? Can two screens reuse `"queue"`? Can the same pane instance be mounted in two places? Is cursor state associated with the instance, name, or mounting location?

This will become a source of bugs and user confusion unless explicitly defined.

**I'd add a lifecycle section before the v0.1 tag.**

# **4. The biggest conceptual weakness: "everything is data" isn't fully true**

The project says:

> Bindings are data → help, palette, footer, audit, and README key tables are all generated views.

This is excellent—but actions still carry arbitrary handlers.

```
type Action struct {
    Key string
    Run Handler
    Help string
    Name string
    NeedsCursor bool
    Danger bool
}
```

The system can audit the declaration, but cannot inspect the semantics of `Run`.

This means there are several classes of "truth" that the kit cannot prove:

- Does the action actually do what its help says?
- Is the action destructive?
- Is it safe to invoke repeatedly?
- Is it async?
- Does it mutate the screen tree?
- Does it invalidate the target?
- Does it require permissions?
- Is it available only in certain data states?

The audit can prove **structural correctness**, not **semantic correctness**.

I would be careful with the wording of the promise. "Cannot lie" is compelling marketing, but technically the system can still lie about action semantics.

A better promise:

> **The kit eliminates inconsistencies it can observe and structurally guarantees.**

This distinction protects credibility.

# **5. Action model: good foundation, but likely underpowered**

The `Action` abstraction is probably the most important public API in the project.

The current model supports:

- Key
- Handler
- Help
- Name
- NeedsCursor
- Danger

This is minimalist and elegant.

But operational applications will eventually want metadata like:

| Attribute            | Why                                  |
| -------------------- | ------------------------------------ |
| Availability         | Action disabled in certain states    |
| Permission           | User lacks authorization             |
| Confirmation policy  | Different levels of danger           |
| Category             | Palette grouping                     |
| Short label          | UI space constraints                 |
| Long description     | Help                                 |
| Target type          | Job, container, file, deployment     |
| Async indicator      | Long-running operation               |
| State feedback       | Running/succeeded/failed             |
| Context requirements | Requires selection, connection, etc. |

The risk is not that you need all these now.

The risk is that the current `Action` becomes a permanent abstraction that is too small, forcing awkward conventions around handler logic.

### **I would not add all of this to v0.1.**

Instead, design a future metadata extension point. Perhaps:

```
type Action struct {
    Key  string
    Run  Handler
    Meta ActionMeta
}
```

Or retain the current ergonomics and add optional interfaces or builders.

The important architectural question is:

> Is the Action intentionally a low-level dispatch primitive, or is it intended to become the declarative representation of application commands?

The current spec is somewhere between those two positions.

That ambiguity should be resolved.

# **6. Palette: potentially your killer feature, but it is constrained**

The command palette is a high-value feature because it makes actions discoverable and contextual.

The resolver invariant is excellent:

> Every palette entry executes exactly as if its key had been pressed right now.

That eliminates a whole class of palette-routing bugs.

However, your strict contextual palette may be too restrictive for actual workflows.

For example, an operator may want to search for:

- "Restart service"
- "Open logs"
- "Switch to deployments"
- "Show network"
- "Kill job"

even if the command isn't currently bound to the focused pane.

The current palette says:

> The palette is an answer, not a search index.

Architecturally interesting, but possibly product-limiting.

### **I recommend two modes eventually**

**Contextual mode — default**

Shows only immediately executable actions. Your existing invariant applies.

**Global discovery mode — optional**

Can show actions outside the current scope but must disclose prerequisites and transitions.

For example:

```
Restart deployment
Requires: deployments screen
Action: switch screen → select deployment → execute
```

This is not a routing lie if the transition is explicit.

I think this could become one of the most useful capabilities in the entire project.

# **7. The testing architecture is one of the best parts**

The testing story is unusually strong.

The combination of:

- Headless rendering
- Golden frames
- Structural geometry assertions
- Region extraction
- Dry-run routing
- Receipts
- CI audits

is much more valuable than simply "a TUI component library."

This has strategic potential.

The test model creates a feedback loop:

\#chatgpt-mermaid-\_r_do\_{font-family:-apple-system-body,ui-sans-serif,-apple-system,system-ui,Segoe UI,Helvetica,Apple Color Emoji,Arial,sans-serif,Segoe UI Emoji,Segoe UI Symbol;font-size:16px;fill:rgb(237, 237, 237);}@keyframes edge-animation-frame{from{stroke-dashoffset:0;}}@keyframes dash{to{stroke-dashoffset:0;}}#chatgpt-mermaid-\_r_do\_ .edge-animation-slow{stroke-dasharray:9,5!important;stroke-dashoffset:900;animation:dash 50s linear infinite;stroke-linecap:round;}#chatgpt-mermaid-\_r_do\_ .edge-animation-fast{stroke-dasharray:9,5!important;stroke-dashoffset:900;animation:dash 20s linear infinite;stroke-linecap:round;}#chatgpt-mermaid-\_r_do\_ .error-icon{fill:rgb(48, 48, 48);}#chatgpt-mermaid-\_r_do\_ .error-text{fill:rgb(237, 237, 237);stroke:rgb(237, 237, 237);}#chatgpt-mermaid-\_r_do\_ .edge-thickness-normal{stroke-width:1px;}#chatgpt-mermaid-\_r_do\_ .edge-thickness-thick{stroke-width:3.5px;}#chatgpt-mermaid-\_r_do\_ .edge-pattern-solid{stroke-dasharray:0;}#chatgpt-mermaid-\_r_do\_ .edge-thickness-invisible{stroke-width:0;fill:none;}#chatgpt-mermaid-\_r_do\_ .edge-pattern-dashed{stroke-dasharray:3;}#chatgpt-mermaid-\_r_do\_ .edge-pattern-dotted{stroke-dasharray:2;}#chatgpt-mermaid-\_r_do\_ .marker{fill:rgb(175, 175, 175);stroke:rgb(175, 175, 175);}#chatgpt-mermaid-\_r_do\_ .marker.cross{stroke:rgb(175, 175, 175);}#chatgpt-mermaid-\_r_do\_ svg{font-family:-apple-system-body,ui-sans-serif,-apple-system,system-ui,Segoe UI,Helvetica,Apple Color Emoji,Arial,sans-serif,Segoe UI Emoji,Segoe UI Symbol;font-size:16px;}#chatgpt-mermaid-\_r_do\_ p{margin:0;}#chatgpt-mermaid-\_r_do\_ .label{font-family:-apple-system-body,ui-sans-serif,-apple-system,system-ui,Segoe UI,Helvetica,Apple Color Emoji,Arial,sans-serif,Segoe UI Emoji,Segoe UI Symbol;color:rgb(237, 237, 237);}#chatgpt-mermaid-\_r_do\_ .cluster-label text{fill:rgb(237, 237, 237);}#chatgpt-mermaid-\_r_do\_ .cluster-label span{color:rgb(237, 237, 237);}#chatgpt-mermaid-\_r_do\_ .cluster-label span p{background-color:transparent;}#chatgpt-mermaid-\_r_do\_ .label text,#chatgpt-mermaid-\_r_do\_ span{fill:rgb(237, 237, 237);color:rgb(237, 237, 237);}#chatgpt-mermaid-\_r_do\_ .node rect,#chatgpt-mermaid-\_r_do\_ .node circle,#chatgpt-mermaid-\_r_do\_ .node ellipse,#chatgpt-mermaid-\_r_do\_ .node polygon,#chatgpt-mermaid-\_r_do\_ .node path{fill:rgb(17, 40, 19);stroke:rgb(58, 132, 63);stroke-width:1px;}#chatgpt-mermaid-\_r_do\_ .rough-node .label text,#chatgpt-mermaid-\_r_do\_ .node .label text,#chatgpt-mermaid-\_r_do\_ .image-shape .label,#chatgpt-mermaid-\_r_do\_ .icon-shape .label{text-anchor:middle;}#chatgpt-mermaid-\_r_do\_ .node .katex path{fill:#000;stroke:#000;stroke-width:1px;}#chatgpt-mermaid-\_r_do\_ .rough-node .label,#chatgpt-mermaid-\_r_do\_ .node .label,#chatgpt-mermaid-\_r_do\_ .image-shape .label,#chatgpt-mermaid-\_r_do\_ .icon-shape .label{text-align:center;}#chatgpt-mermaid-\_r_do\_ .node.clickable{cursor:pointer;}#chatgpt-mermaid-\_r_do\_ .root .anchor path{fill:rgb(175, 175, 175)!important;stroke-width:0;stroke:rgb(175, 175, 175);}#chatgpt-mermaid-\_r_do\_ .arrowheadPath{fill:rgb(175, 175, 175);}#chatgpt-mermaid-\_r_do\_ .edgePath .path{stroke:rgb(175, 175, 175);stroke-width:1px;}#chatgpt-mermaid-\_r_do\_ .flowchart-link{stroke:rgb(175, 175, 175);fill:none;}#chatgpt-mermaid-\_r_do\_ .edgeLabel{background-color:rgb(0, 0, 0);text-align:center;}#chatgpt-mermaid-\_r_do\_ .edgeLabel p{background-color:rgb(0, 0, 0);}#chatgpt-mermaid-\_r_do\_ .edgeLabel rect{opacity:0.5;background-color:rgb(0, 0, 0);fill:rgb(0, 0, 0);}#chatgpt-mermaid-\_r_do\_ .labelBkg{background-color:rgba(0, 0, 0, 0.5);}#chatgpt-mermaid-\_r_do\_ .cluster rect{fill:rgb(48, 48, 48);stroke:rgba(255, 255, 255, 0.15);stroke-width:1px;}#chatgpt-mermaid-\_r_do\_ .cluster text{fill:rgb(237, 237, 237);}#chatgpt-mermaid-\_r_do\_ .cluster span{color:rgb(237, 237, 237);}#chatgpt-mermaid-\_r_do\_ div.mermaidTooltip{position:absolute;text-align:center;max-width:200px;padding:2px;font-family:-apple-system-body,ui-sans-serif,-apple-system,system-ui,Segoe UI,Helvetica,Apple Color Emoji,Arial,sans-serif,Segoe UI Emoji,Segoe UI Symbol;font-size:12px;background:rgb(48, 48, 48);border:1px solid rgba(255, 255, 255, 0.15);border-radius:2px;pointer-events:none;z-index:100;}#chatgpt-mermaid-\_r_do\_ .flowchartTitleText{text-anchor:middle;font-size:18px;fill:rgb(237, 237, 237);}#chatgpt-mermaid-\_r_do\_ rect.text{fill:none;stroke-width:0;}#chatgpt-mermaid-\_r_do\_ .icon-shape,#chatgpt-mermaid-\_r_do\_ .image-shape{background-color:rgb(0, 0, 0);text-align:center;}#chatgpt-mermaid-\_r_do\_ .icon-shape p,#chatgpt-mermaid-\_r_do\_ .image-shape p{background-color:rgb(0, 0, 0);padding:2px;}#chatgpt-mermaid-\_r_do\_ .icon-shape .label rect,#chatgpt-mermaid-\_r_do\_ .image-shape .label rect{opacity:0.5;background-color:rgb(0, 0, 0);fill:rgb(0, 0, 0);}#chatgpt-mermaid-\_r_do\_ .label-icon{display:inline-block;height:1em;overflow:visible;vertical-align:-0.125em;}#chatgpt-mermaid-\_r_do\_ .node .label-icon path{fill:currentColor;stroke:revert;stroke-width:revert;}#chatgpt-mermaid-\_r_do\_ .node .neo-node{stroke:rgb(58, 132, 63);}#chatgpt-mermaid-\_r_do\_ [data-look="neo"].node rect,#chatgpt-mermaid-\_r_do\_ [data-look="neo"].cluster rect,#chatgpt-mermaid-\_r_do\_ [data-look="neo"].node polygon{stroke:url(#chatgpt-mermaid-\_r_do\_-gradient);filter:drop-shadow( 1px 2px 2px rgba(185,185,185,1));}#chatgpt-mermaid-\_r_do\_ [data-look="neo"].swimlane.cluster rect{filter:none;}#chatgpt-mermaid-\_r_do\_ [data-look="neo"].node path{stroke:url(#chatgpt-mermaid-\_r_do\_-gradient);stroke-width:1px;}#chatgpt-mermaid-\_r_do\_ [data-look="neo"].node .outer-path{filter:drop-shadow( 1px 2px 2px rgba(185,185,185,1));}#chatgpt-mermaid-\_r_do\_ [data-look="neo"].node .neo-line path{stroke:rgb(58, 132, 63);filter:none;}#chatgpt-mermaid-\_r_do\_ [data-look="neo"].node circle{stroke:url(#chatgpt-mermaid-\_r_do\_-gradient);filter:drop-shadow( 1px 2px 2px rgba(185,185,185,1));}#chatgpt-mermaid-\_r_do\_ [data-look="neo"].node circle .state-start{fill:#000000;}#chatgpt-mermaid-\_r_do\_ [data-look="neo"].icon-shape .icon{fill:url(#chatgpt-mermaid-\_r_do\_-gradient);filter:drop-shadow( 1px 2px 2px rgba(185,185,185,1));}#chatgpt-mermaid-\_r_do\_ [data-look="neo"].icon-shape .icon-neo path{stroke:url(#chatgpt-mermaid-\_r_do\_-gradient);filter:drop-shadow( 1px 2px 2px rgba(185,185,185,1));}#chatgpt-mermaid-\_r_do\_ .node text{font-size:14px;font-weight:600;letter-spacing:normal;fill:rgb(140, 223, 173);}#chatgpt-mermaid-\_r_do\_ .edgeLabels text{font-size:13px;font-weight:600;letter-spacing:-0.08px;fill:rgb(140, 223, 173);}#chatgpt-mermaid-\_r_do\_ .node tspan[font-weight="normal"],#chatgpt-mermaid-\_r_do\_ .edgeLabels tspan[font-weight="normal"]{font-weight:600;}#chatgpt-mermaid-\_r_do\_ .edgeLabel .label rect{opacity:1;rx:13px;ry:13px;fill:rgb(2, 28, 12);stroke:rgb(26, 75, 45);stroke-width:1px;}#chatgpt-mermaid-\_r_do\_ .node rect,#chatgpt-mermaid-\_r_do\_ .node circle,#chatgpt-mermaid-\_r_do\_ .node ellipse,#chatgpt-mermaid-\_r_do\_ .node polygon,#chatgpt-mermaid-\_r_do\_ .node path{fill:rgb(0, 55, 22);stroke:rgba(255, 255, 255, 0.1);stroke-width:1px;}#chatgpt-mermaid-\_r_do\_ .node rect{rx:16px;ry:16px;}#chatgpt-mermaid-\_r_do\_ .mermaid-decision .label-container{fill:rgb(2, 28, 12);stroke:rgb(26, 75, 45);stroke-dasharray:2px,2px;}#chatgpt-mermaid-\_r_do\_ .edgePaths .flowchart-link{stroke:rgb(175, 175, 175);stroke-width:1px;stroke-linecap:round;stroke-linejoin:round;}#chatgpt-mermaid-\_r_do\_ .marker{fill:rgb(175, 175, 175);stroke:rgb(175, 175, 175);}#chatgpt-mermaid-\_r_do\_ :root{--mermaid-font-family:-apple-system-body,ui-sans-serif,-apple-system,system-ui,Segoe UI,Helvetica,Apple Color Emoji,Arial,sans-serif,Segoe UI Emoji,Segoe UI Symbol;}Declare appAudit wiringRun deterministic interactionsInspect receiptsAssert framesAssert geometryDebug

The important thing is to avoid making golden tests the dominant testing interface.

Golden frames can be brittle due to:

- Terminal width changes
- Styling changes
- Unicode width behavior
- Lipgloss version changes
- Color profile differences
- Rendering implementation changes

Your structural assertions (`SlotOf`, `Region`, `Dump`) are the better long-term abstraction.

I would emphasize **semantic testing first, golden rendering second**.

# **8. Receipts: underdeveloped and potentially strategically valuable**

The receipt stream is described primarily as:

- Routing trace
- Sequence number
- Target
- Stack changes
- Focus changes
- Resize facts
- Debug events

But it could become much more valuable.

Imagine a standardized interaction event:

```
sequence: 1042
trigger: key
key: K
action: kill
scope: queue
target:
  id: job-123
  index: 5
result: confirm-pushed
```

Then:

```
sequence: 1043
trigger: palette
action: confirm-yes
target: job-123
result: handler-executed
```

This creates a machine-readable behavioral trace.

Potential future capabilities:

- Reproducible interaction bugs
- Test scenario recording
- Replay
- State transition diagnostics
- Telemetry
- Interaction coverage
- Debug exports
- Regression test generation

### **Strong business angle**

The receipt stream and audit system could become the differentiator beyond the basic kit.

The library itself is easy to copy.

A **reliable developer workflow for building, testing, debugging, and validating operational TUIs** is harder to replicate.

# **9. Adoption friction: the biggest threat to success**

The adoption ladder is good:

1. Kill Model boilerplate
2. Add screens
3. Add verbs
4. Adopt audit
5. Adopt testing

This is exactly the right instinct for a library.

However, I think the current first step may not be compelling enough.

`kit.Run(app)` removes boilerplate, but developers already have relatively little Bubble Tea boilerplate. Switching to a new architecture means they must learn:

- Screen
- Pane
- Layout
- Slot
- Action
- Ctx
- Cursor
- Modal
- Track
- Theme
- Min
- Scope
- Receipt

The conceptual load could be greater than the boilerplate it eliminates.

## **The adoption test**

A developer should be able to integrate the library and receive a visible benefit within 10 minutes.

I'd optimize the first experience around this:

```
// Existing Bubble Tea app
// Add contextual bindings and validation.

kit.Audit(myApp)
kit.Help(myApp)
kit.Route(myApp, key)
```

Then progressively adopt:

- Screen management
- Layout
- Cursor
- Modal
- Testing

If the library requires adopting its entire conceptual model before it becomes useful, adoption will suffer.

The adoption ladder in the spec is directionally correct—but the implementation needs to ensure the rungs are genuinely independent.

# **10. Architecture: what I would improve**

## **A. Reconsider the single&#x20;**`App`**&#x20;abstraction**

The spec puts a lot of responsibilities in `App`:

- Terminal size
- Screen registry
- Global bindings
- Focus
- Screen stack
- Modal stack
- Theme
- Router
- Receipts
- Debugging
- Program lifecycle

This is conceptually coherent, but `App` risks becoming the central god-object.

I would keep `App` as the user-facing façade, while internally separate the implementation into explicit subsystems:

\#chatgpt-mermaid-\_r_ec\_{font-family:-apple-system-body,ui-sans-serif,-apple-system,system-ui,Segoe UI,Helvetica,Apple Color Emoji,Arial,sans-serif,Segoe UI Emoji,Segoe UI Symbol;font-size:16px;fill:rgb(237, 237, 237);}@keyframes edge-animation-frame{from{stroke-dashoffset:0;}}@keyframes dash{to{stroke-dashoffset:0;}}#chatgpt-mermaid-\_r_ec\_ .edge-animation-slow{stroke-dasharray:9,5!important;stroke-dashoffset:900;animation:dash 50s linear infinite;stroke-linecap:round;}#chatgpt-mermaid-\_r_ec\_ .edge-animation-fast{stroke-dasharray:9,5!important;stroke-dashoffset:900;animation:dash 20s linear infinite;stroke-linecap:round;}#chatgpt-mermaid-\_r_ec\_ .error-icon{fill:rgb(48, 48, 48);}#chatgpt-mermaid-\_r_ec\_ .error-text{fill:rgb(237, 237, 237);stroke:rgb(237, 237, 237);}#chatgpt-mermaid-\_r_ec\_ .edge-thickness-normal{stroke-width:1px;}#chatgpt-mermaid-\_r_ec\_ .edge-thickness-thick{stroke-width:3.5px;}#chatgpt-mermaid-\_r_ec\_ .edge-pattern-solid{stroke-dasharray:0;}#chatgpt-mermaid-\_r_ec\_ .edge-thickness-invisible{stroke-width:0;fill:none;}#chatgpt-mermaid-\_r_ec\_ .edge-pattern-dashed{stroke-dasharray:3;}#chatgpt-mermaid-\_r_ec\_ .edge-pattern-dotted{stroke-dasharray:2;}#chatgpt-mermaid-\_r_ec\_ .marker{fill:rgb(175, 175, 175);stroke:rgb(175, 175, 175);}#chatgpt-mermaid-\_r_ec\_ .marker.cross{stroke:rgb(175, 175, 175);}#chatgpt-mermaid-\_r_ec\_ svg{font-family:-apple-system-body,ui-sans-serif,-apple-system,system-ui,Segoe UI,Helvetica,Apple Color Emoji,Arial,sans-serif,Segoe UI Emoji,Segoe UI Symbol;font-size:16px;}#chatgpt-mermaid-\_r_ec\_ p{margin:0;}#chatgpt-mermaid-\_r_ec\_ .label{font-family:-apple-system-body,ui-sans-serif,-apple-system,system-ui,Segoe UI,Helvetica,Apple Color Emoji,Arial,sans-serif,Segoe UI Emoji,Segoe UI Symbol;color:rgb(237, 237, 237);}#chatgpt-mermaid-\_r_ec\_ .cluster-label text{fill:rgb(237, 237, 237);}#chatgpt-mermaid-\_r_ec\_ .cluster-label span{color:rgb(237, 237, 237);}#chatgpt-mermaid-\_r_ec\_ .cluster-label span p{background-color:transparent;}#chatgpt-mermaid-\_r_ec\_ .label text,#chatgpt-mermaid-\_r_ec\_ span{fill:rgb(237, 237, 237);color:rgb(237, 237, 237);}#chatgpt-mermaid-\_r_ec\_ .node rect,#chatgpt-mermaid-\_r_ec\_ .node circle,#chatgpt-mermaid-\_r_ec\_ .node ellipse,#chatgpt-mermaid-\_r_ec\_ .node polygon,#chatgpt-mermaid-\_r_ec\_ .node path{fill:rgb(17, 40, 19);stroke:rgb(58, 132, 63);stroke-width:1px;}#chatgpt-mermaid-\_r_ec\_ .rough-node .label text,#chatgpt-mermaid-\_r_ec\_ .node .label text,#chatgpt-mermaid-\_r_ec\_ .image-shape .label,#chatgpt-mermaid-\_r_ec\_ .icon-shape .label{text-anchor:middle;}#chatgpt-mermaid-\_r_ec\_ .node .katex path{fill:#000;stroke:#000;stroke-width:1px;}#chatgpt-mermaid-\_r_ec\_ .rough-node .label,#chatgpt-mermaid-\_r_ec\_ .node .label,#chatgpt-mermaid-\_r_ec\_ .image-shape .label,#chatgpt-mermaid-\_r_ec\_ .icon-shape .label{text-align:center;}#chatgpt-mermaid-\_r_ec\_ .node.clickable{cursor:pointer;}#chatgpt-mermaid-\_r_ec\_ .root .anchor path{fill:rgb(175, 175, 175)!important;stroke-width:0;stroke:rgb(175, 175, 175);}#chatgpt-mermaid-\_r_ec\_ .arrowheadPath{fill:rgb(175, 175, 175);}#chatgpt-mermaid-\_r_ec\_ .edgePath .path{stroke:rgb(175, 175, 175);stroke-width:1px;}#chatgpt-mermaid-\_r_ec\_ .flowchart-link{stroke:rgb(175, 175, 175);fill:none;}#chatgpt-mermaid-\_r_ec\_ .edgeLabel{background-color:rgb(0, 0, 0);text-align:center;}#chatgpt-mermaid-\_r_ec\_ .edgeLabel p{background-color:rgb(0, 0, 0);}#chatgpt-mermaid-\_r_ec\_ .edgeLabel rect{opacity:0.5;background-color:rgb(0, 0, 0);fill:rgb(0, 0, 0);}#chatgpt-mermaid-\_r_ec\_ .labelBkg{background-color:rgba(0, 0, 0, 0.5);}#chatgpt-mermaid-\_r_ec\_ .cluster rect{fill:rgb(48, 48, 48);stroke:rgba(255, 255, 255, 0.15);stroke-width:1px;}#chatgpt-mermaid-\_r_ec\_ .cluster text{fill:rgb(237, 237, 237);}#chatgpt-mermaid-\_r_ec\_ .cluster span{color:rgb(237, 237, 237);}#chatgpt-mermaid-\_r_ec\_ div.mermaidTooltip{position:absolute;text-align:center;max-width:200px;padding:2px;font-family:-apple-system-body,ui-sans-serif,-apple-system,system-ui,Segoe UI,Helvetica,Apple Color Emoji,Arial,sans-serif,Segoe UI Emoji,Segoe UI Symbol;font-size:12px;background:rgb(48, 48, 48);border:1px solid rgba(255, 255, 255, 0.15);border-radius:2px;pointer-events:none;z-index:100;}#chatgpt-mermaid-\_r_ec\_ .flowchartTitleText{text-anchor:middle;font-size:18px;fill:rgb(237, 237, 237);}#chatgpt-mermaid-\_r_ec\_ rect.text{fill:none;stroke-width:0;}#chatgpt-mermaid-\_r_ec\_ .icon-shape,#chatgpt-mermaid-\_r_ec\_ .image-shape{background-color:rgb(0, 0, 0);text-align:center;}#chatgpt-mermaid-\_r_ec\_ .icon-shape p,#chatgpt-mermaid-\_r_ec\_ .image-shape p{background-color:rgb(0, 0, 0);padding:2px;}#chatgpt-mermaid-\_r_ec\_ .icon-shape .label rect,#chatgpt-mermaid-\_r_ec\_ .image-shape .label rect{opacity:0.5;background-color:rgb(0, 0, 0);fill:rgb(0, 0, 0);}#chatgpt-mermaid-\_r_ec\_ .label-icon{display:inline-block;height:1em;overflow:visible;vertical-align:-0.125em;}#chatgpt-mermaid-\_r_ec\_ .node .label-icon path{fill:currentColor;stroke:revert;stroke-width:revert;}#chatgpt-mermaid-\_r_ec\_ .node .neo-node{stroke:rgb(58, 132, 63);}#chatgpt-mermaid-\_r_ec\_ [data-look="neo"].node rect,#chatgpt-mermaid-\_r_ec\_ [data-look="neo"].cluster rect,#chatgpt-mermaid-\_r_ec\_ [data-look="neo"].node polygon{stroke:url(#chatgpt-mermaid-\_r_ec\_-gradient);filter:drop-shadow( 1px 2px 2px rgba(185,185,185,1));}#chatgpt-mermaid-\_r_ec\_ [data-look="neo"].swimlane.cluster rect{filter:none;}#chatgpt-mermaid-\_r_ec\_ [data-look="neo"].node path{stroke:url(#chatgpt-mermaid-\_r_ec\_-gradient);stroke-width:1px;}#chatgpt-mermaid-\_r_ec\_ [data-look="neo"].node .outer-path{filter:drop-shadow( 1px 2px 2px rgba(185,185,185,1));}#chatgpt-mermaid-\_r_ec\_ [data-look="neo"].node .neo-line path{stroke:rgb(58, 132, 63);filter:none;}#chatgpt-mermaid-\_r_ec\_ [data-look="neo"].node circle{stroke:url(#chatgpt-mermaid-\_r_ec\_-gradient);filter:drop-shadow( 1px 2px 2px rgba(185,185,185,1));}#chatgpt-mermaid-\_r_ec\_ [data-look="neo"].node circle .state-start{fill:#000000;}#chatgpt-mermaid-\_r_ec\_ [data-look="neo"].icon-shape .icon{fill:url(#chatgpt-mermaid-\_r_ec\_-gradient);filter:drop-shadow( 1px 2px 2px rgba(185,185,185,1));}#chatgpt-mermaid-\_r_ec\_ [data-look="neo"].icon-shape .icon-neo path{stroke:url(#chatgpt-mermaid-\_r_ec\_-gradient);filter:drop-shadow( 1px 2px 2px rgba(185,185,185,1));}#chatgpt-mermaid-\_r_ec\_ .node text{font-size:14px;font-weight:600;letter-spacing:normal;fill:rgb(140, 223, 173);}#chatgpt-mermaid-\_r_ec\_ .edgeLabels text{font-size:13px;font-weight:600;letter-spacing:-0.08px;fill:rgb(140, 223, 173);}#chatgpt-mermaid-\_r_ec\_ .node tspan[font-weight="normal"],#chatgpt-mermaid-\_r_ec\_ .edgeLabels tspan[font-weight="normal"]{font-weight:600;}#chatgpt-mermaid-\_r_ec\_ .edgeLabel .label rect{opacity:1;rx:13px;ry:13px;fill:rgb(2, 28, 12);stroke:rgb(26, 75, 45);stroke-width:1px;}#chatgpt-mermaid-\_r_ec\_ .node rect,#chatgpt-mermaid-\_r_ec\_ .node circle,#chatgpt-mermaid-\_r_ec\_ .node ellipse,#chatgpt-mermaid-\_r_ec\_ .node polygon,#chatgpt-mermaid-\_r_ec\_ .node path{fill:rgb(0, 55, 22);stroke:rgba(255, 255, 255, 0.1);stroke-width:1px;}#chatgpt-mermaid-\_r_ec\_ .node rect{rx:16px;ry:16px;}#chatgpt-mermaid-\_r_ec\_ .mermaid-decision .label-container{fill:rgb(2, 28, 12);stroke:rgb(26, 75, 45);stroke-dasharray:2px,2px;}#chatgpt-mermaid-\_r_ec\_ .edgePaths .flowchart-link{stroke:rgb(175, 175, 175);stroke-width:1px;stroke-linecap:round;stroke-linejoin:round;}#chatgpt-mermaid-\_r_ec\_ .marker{fill:rgb(175, 175, 175);stroke:rgb(175, 175, 175);}#chatgpt-mermaid-\_r_ec\_ :root{--mermaid-font-family:-apple-system-body,ui-sans-serif,-apple-system,system-ui,Segoe UI,Helvetica,Apple Color Emoji,Arial,sans-serif,Segoe UI Emoji,Segoe UI Symbol;}AppRuntimeRegistryRouterLayoutEngineStackManagerAuditRecorderRendererTestDriver

These don't necessarily need to be public types.

The twelve-type budget is a useful constraint for the public API, but **public type minimization should not dictate internal implementation structure**.

A small public API is good. A monolithic implementation is not.

## **B. Define the rendering contract more rigorously**

The project calls rendering deterministic and pure, but the example has panes doing this:

```
func (p *QueuePane) Render(s kit.Slot) string {
    p.cursor.Track(idsOf(p.store.Open()))
    // ...
}
```

This means `Render` is not strictly pure: it updates internal cursor synchronization state.

The spec acknowledges state persistence and revalidation, but this creates a subtle semantic distinction:

> Rendering is deterministic, but rendering may perform deterministic state normalization.

That is not necessarily wrong. But the contract should say it explicitly.

I would define the lifecycle as:

1. App receives event.
2. Kit applies state transition.
3. Current data snapshot is declared to the pane.
4. State is reconciled against the declaration.
5. Geometry is resolved.
6. Rendering reads normalized state.
7. Rendering produces output.

The more rendering itself mutates state, the harder it becomes to reason about repeated render calls, tests, and side effects.

I would explore whether `Track()` should happen during an explicit synchronization phase instead of `Render()`.

Even if you retain the current design, explicitly formalize the guarantee:

> `Track()` is idempotent and normalization-only; it cannot produce observable domain-side effects.

## **C. Reconsider&#x20;**`Notify`

You explicitly say:

> Track replaces change hooks.

Yet you still have a `Notifier` interface for resize and screen activation.

This may be entirely justified because geometry and lifecycle are not data synchronization. But the distinction is subtle.

Developers could reasonably ask:

- Why is `Notify` allowed but change hooks are forbidden?
- Is it a lifecycle callback?
- Is it a synchronization callback?
- Can it mutate state?
- Can it launch commands?
- Is it called more than once?

I would rename or narrow this interface so the distinction is explicit.

Something like:

```
type Lifecycle interface {
    OnVisible()
    OnResize(kit.Slot)
}
```

Or even keep the current interface but document the semantics and guarantees precisely.

## **D. Screen stack semantics need more definition**

You have two stacks:

- Screen stack
- Modal stack

And `esc` pops the innermost pushable.

This is elegant, but the interactions become nontrivial.

Questions that need explicit answers:

- Can the same screen appear multiple times on the stack?
- Does pushing a detail screen preserve the previous screen's cursor?
- Is screen state preserved while hidden?
- What happens if the underlying screen is resized?
- Can a screen stack entry have parameters?
- Does the screen registry contain instances or constructors?
- Is screen activation a callback?
- Is pushing a screen always reversible?
- What does the footer show during a modal?
- What happens when the palette opens over a confirm modal?

These aren't peripheral. They define the semantics of navigation.

**I would write a state-machine specification for stack interactions before expanding features.**

## **E. Audit needs a severity model**

Currently, `Validate()` returns violations.

I would distinguish:

| Severity | Meaning                     | CI behavior |
| -------- | --------------------------- | ----------- |
| Error    | App is structurally invalid | Fail        |
| Warning  | Suspicious but legal        | Report      |
| Info     | Recommendation              | Optional    |

Examples:

**Error**

- Binding collision
- Reserved key rebinding
- Cursor-required action at invalid scope
- Danger action missing confirmation

**Warning**

- Screen unreachable
- Duplicate action descriptions
- Too many stack levels implied by design
- Pane minimums unrealistic
- No quit action

**Info**

- Consider adding help
- Consider adding a palette label
- Potentially confusing key choice

This makes the audit much more useful as an engineering tool.

A validator that only says "here are violations" will eventually become noisy.

# **11. Important missing capabilities**

These are gaps I'd consider—not necessarily features I'd immediately implement.

## **Priority 1 — essential foundations**

| Gap                                    | Why it matters                                      |
| -------------------------------------- | --------------------------------------------------- |
| Lifecycle semantics                    | Prevents hidden-state and stack bugs                |
| Focus transitions                      | Fundamental to navigation                           |
| Disabled/unavailable actions           | Real operational apps need state-dependent commands |
| Action outcome model                   | Users need feedback when operations succeed/fail    |
| Error presentation                     | A handler failing shouldn't disappear into logs     |
| Screen/pane identity model             | Avoids reuse and mounting ambiguity                 |
| Accessibility / terminal behavior      | Key discoverability, screen-reader compatibility    |
| Unicode width and rendering edge cases | Real terminal correctness                           |

## **Priority 2 — product quality**

| Gap                               | Why it matters                                   |
| --------------------------------- | ------------------------------------------------ |
| Contextual action states          | Palette/help should show disabled reasons        |
| Search/discovery across scopes    | Helps larger apps                                |
| Interaction recording/replay      | Dramatically improves debugging                  |
| Configurable keymaps              | Users expect customization                       |
| State persistence                 | Operational tools often want session restoration |
| App-level error boundaries        | Better failure handling                          |
| Command categories                | Palette usability                                |
| Async action feedback conventions | Long-running operations                          |

## **Priority 3 — optional expansion**

- Mouse
- Horizontal scroll
- Rich text
- Dynamic layout
- Plugin architecture
- Multi-window terminal support
- Advanced theme configuration

I would not expand into Priority 3 until there is real demand.

# **12. Business angles**

There are several, but I would separate the library business from the ecosystem business.

## **Angle 1: Open-source infrastructure library**

**The most natural starting point.**

Target:

- Go developers
- Backend engineers
- DevOps tooling authors
- Kubernetes operators
- Infrastructure CLI/TUI developers
- Internal platform teams

Value:

> Spend less time debugging terminal interaction infrastructure and more time building the actual operational tool.

### **Monetization**

Direct monetization of a Go UI kit is difficult.

Potential models:

- Open-source adoption as a distribution channel
- Sponsorship
- Consulting
- Enterprise support
- Commercial adjacent tooling

I would not put a paywall around the core library.

The value of this product depends heavily on ecosystem trust and widespread adoption.

## **Angle 2: The "operational TUI starter kit"**

This may have a better commercial opportunity than the bare library.

Ship opinionated starter templates for:

- Kubernetes dashboard
- Database administration tool
- Queue/job inspector
- Infrastructure deployment manager
- Log explorer
- Internal platform CLI

The kit could provide:

- Standard navigation
- Contextual command palette
- Confirmations
- List/detail views
- Logging
- Error presentation
- Authentication hooks
- Config
- Testing
- CI
- Packaging

This reduces time from zero to production.

You sell acceleration rather than a library.

## **Angle 3: Enterprise TUI reliability tooling**

Potentially the most interesting long-term opportunity.

Imagine a product that runs interaction audits over operational TUIs:

```
App audit

42 actions registered
39 reachable
3 unreachable

17 actions require cursor
17 have valid targets

6 destructive actions
6 have confirmation flows

4 key collisions
0 reserved-key violations

Screen coverage: 100%
Action test coverage: 73%
```

Then:

- Generate interaction documentation
- Record execution traces
- Replay bug reports
- Create regression tests
- Audit keybinding consistency
- Detect unreachable actions
- Analyze workflow completeness

This could be a developer-tooling product.

The kit becomes the instrumentation layer.

## **Angle 4: A higher-level operational application framework**

This is the more ambitious path.

The kit currently says:

> We are not a framework.

I think you should preserve that stance for the core library.

But there may be a separate package that provides a framework for common operational workflows:

```
Operational application
 ├── Navigation
 ├── Resource list
 ├── Detail inspector
 ├── Logs
 ├── Events
 ├── Actions
 ├── Confirmations
 ├── Command palette
 └── Debugging
```

Think of the core as the interaction kernel.

The framework is optional.

This separation allows you to grow without compromising the minimal library.

## **Angle 5: Ecosystem standardization**

A more strategic, longer-term possibility:

The library defines a common interaction vocabulary for Go terminal applications.

If enough applications adopt it, there may be opportunities for:

- Shared keyboard conventions
- Consistent command palettes
- Standard action metadata
- Common test tooling
- Reusable operational patterns

This is a network-effect thesis—but realistically, it is a long shot until the project has meaningful adoption.

Don't build for this yet. But the architectural design is compatible with it.

# **13. Competitive positioning**

The product should not position itself as another UI library.

There are already:

- Bubble Tea itself
- Bubbles
- Lipgloss
- Higher-level TUI frameworks
- App-specific internal abstractions

Your differentiation is the integrated contract.

| Existing ecosystem approach | Your positioning                |
| --------------------------- | ------------------------------- |
| Components                  | Interaction architecture        |
| Widgets                     | Screens and panes               |
| Keymaps                     | Auditable actions               |
| Handwritten help            | Generated help                  |
| Ad hoc routing              | Queryable routing               |
| Manual confirmations        | Derived safety flows            |
| Visual testing              | Headless deterministic testing  |
| Debug prints                | Structured interaction receipts |
| Manual edge-case handling   | Explicit invariants             |

The positioning I would use:

> **The missing interaction layer for Bubble Tea.**

Or more explicitly:

> **Build operational TUIs with predictable navigation, auditable actions, generated help, and deterministic tests.**

# **14. What I would validate before investing more**

The greenfield spec is already very mature. The next step should be empirical validation, not more architectural debate.

I'd identify five real applications and migrate them.

Ideally:

1. A very small TUI — 1 screen, 2 panes.
2. A list/detail application.
3. A log explorer.
4. An application with complex modal interactions.
5. A production operational tool with destructive actions and async data.

For each, measure:

| Metric                          | Before | After |
| ------------------------------- | ------ | ----- |
| Lines of UI infrastructure code |        |       |
| Number of custom key handlers   |        |       |
| Bugs involving focus/navigation |        |       |
| Time to add a screen            |        |       |
| Time to add an action           |        |       |
| Test setup complexity           |        |       |
| Test coverage of interactions   |        |       |
| Dependency complexity           |        |       |
| Conceptual complexity           |        |       |

The key question is:

> Does this kit make the application simpler after the initial learning cost?

I would also conduct a blind test: give a Go developer a small existing Bubble Tea app and ask them to implement a feature using the kit without you guiding them.

Observe where they struggle.

That is more valuable than another clean-slate architecture exercise.

# **15. What I would change in the roadmap**

Your current ship sequencing is technically logical. I would alter it slightly to focus on proving the product.

## **Phase 1 — Core contract**

Ship only:

- App/root geometry ownership
- Screens and layouts
- Panes
- Action declarations
- Router
- Focus
- Capture semantics
- Confirmations
- Cursor identity
- Audit

Do not let the debug palette or theme sophistication block this.

## **Phase 2 — Developer feedback loop**

Ship:

- Dry-run routing
- Receipts
- Headless testing
- Geometry assertions
- Golden frames
- CI validation

This is where the differentiated value becomes visible.

## **Phase 3 — UX projections**

Ship:

- Help
- Footer
- Palette
- Debug overlay

Make sure all are projections of the same action resolver.

## **Phase 4 — Migration proof**

Migrate two or three real applications.

Use this to validate:

- API ergonomics
- Lifecycle semantics
- Identity model
- Action model
- Escape hatches
- Actual reduction in code

## **Phase 5 — Public release**

Only after the migration evidence:

- Documentation
- Examples
- Starter template
- Stable package API
- v0.1.0

I would prioritize the **proof of adoption** over finishing every element of the spec.

# **16. My assessment by dimension**

## **Project assessment**

| Architectural coherence           | **9/10** |
| --------------------------------- | -------- |
| Problem validity                  | **8/10** |
| Technical differentiation         | **8/10** |
| API maturity                      | **7/10** |
| Adoption simplicity               | **6/10** |
| Evidence of product-market fit    | **4/10** |
| Standalone monetization potential | **3/10** |
| Strategic ecosystem potential     | **8/10** |

These are my qualitative judgments based on the spec, not measurements of the implementation or market.

# **17. The five things I'd challenge most strongly**

If I were acting as an architecture reviewer, these would be my principal challenges:

### **1. Are you solving the most painful problem, or the most intellectually satisfying one?**

The architecture is elegant. But you need evidence that users actually suffer from routing drift, modal correctness, testing, and cursor synchronization enough to adopt a new abstraction.

### **2. Is the API actually simpler than the code it replaces?**

Twelve types is a useful constraint, but type count is not a proxy for cognitive load.

The test is the first 15 minutes with the library.

### **3. Are you treating current constraints as permanent philosophy?**

Pinned escape, no global discovery, no mouse, no dynamic layout, no change hooks—these are reasonable v0 decisions. Don't let them become sacred prematurely.

### **4. Is your testing and receipt system more valuable than the UI abstraction?**

I suspect the answer may be yes.

Your strongest differentiator may ultimately be:

**an interaction contract that can be audited, simulated, recorded, and tested.**

### **5. What is the product after the library?**

If you want a business, think beyond package downloads.

The natural extension is a reliability/developer workflow layer for building operational interfaces, potentially including:

- Interaction audit tooling
- Trace recording and replay
- Workflow testing
- Starter applications
- Enterprise support

# **Final verdict**

I would continue with this project.

The spec demonstrates unusual architectural discipline, particularly around **ownership, determinism, declarative routing, capture semantics, identity-based cursors, and the one-table/multiple-projections model**. The convergence between the existing implementation and the clean-slate exercise is a positive signal.&#x20;

greenfield.md



But I would stop expanding the design for a moment.

The highest-value next step is to build and migrate real applications, then measure whether the kit:

- Reduces code
- Reduces bugs
- Makes interaction behavior easier to understand
- Makes testing easier
- Makes adding new features faster
- Is adoptable without requiring architectural evangelism

My strongest strategic recommendation is to **position the project around reliable, testable interactions—not around TUI components or screen management**.

The library is the implementation. The product promise is that operational TUIs become safer, more discoverable, and easier to validate.

That positioning has a substantially stronger long-term business and ecosystem angle.

If I were pushing this forward, I would next conduct a **red-team architecture review of the spec**: walk through 20–30 difficult runtime scenarios (stack behavior, focus loss, disappearing items, async updates, resize, nested confirmation, action failure, pane reuse) and attempt to find contradictions or undefined behavior before freezing v0.1. That would be the most valuable engineering exercise at this stage.


I rated **Basic screen composition** as:

- **Pain: Medium**
- **Frequency: High**
- **Your solution: Good**

That combination is deliberate. Screen composition is a problem developers encounter constantly, but it is usually not the most painful problem they're trying to solve. And your architecture handles it well, but hasn't yet demonstrated that it makes the *simplest possible case* exceptionally simple.

## **1. Why Medium pain?**

Basic screen composition in Bubble Tea typically involves:

- Creating screens or views
- Splitting the UI into panes
- Managing focus
- Handling screen transitions
- Coordinating dimensions and resizing
- Composing child components
- Managing state across navigation

This is recurring friction, but most developers can already solve it using Bubble Tea, Bubbles, Lipgloss, and their own abstractions.

The pain is generally:

> "I have to write and maintain this boilerplate."

Rather than:

> "I cannot build this UI without a specialized framework."

That's why I didn't rate it High pain.

The pain rises considerably when composition becomes complicated: nested navigation, panes with independent focus, screen transitions, resize behavior, modals, and preserving state between views.

## **2. Why High frequency?**

Almost every non-trivial TUI has some composition:

```
┌─────────────────────────────────────────┐
│ Header                                  │
├────────────────────┬────────────────────┤
│                    │                    │
│ List               │ Details            │
│                    │                    │
├────────────────────┴────────────────────┤
│ Footer / Help                           │
└─────────────────────────────────────────┘
```

Even a simple application may have:

- 2–5 screens
- 2–4 panes per screen
- A header and footer
- Modal dialogs
- Different screen layouts
- Screen-specific keybindings

So frequency is high. It's one of the first things a developer encounters when building an application.

But **high frequency does not necessarily mean high willingness to pay or high product differentiation**. It is a prerequisite capability, not automatically a compelling selling point.

## **3. Why I rated your solution Good rather than Excellent**

Your architecture has a good conceptual model:

```
App
 └── Screen
      └── Layout
           ├── Pane
           ├── Pane
           └── Pane
```

And it has several particularly good properties:

- Screens represent navigable contexts.
- Layouts represent geometry, not behavior.
- Panes own interaction behavior.
- The root owns terminal dimensions.
- Layouts are composable.
- Screen and pane composition is declarative.
- The architecture avoids arbitrary nested behavioral containers.
- Rendering is deterministic.
- Focus and action ownership have defined conceptual homes.

These are genuinely strong architectural decisions.

My hesitation is that **an architecture can be elegant without making composition delightful**.

The current specification defines a coherent composition system, but I would want to see evidence that common composition tasks are genuinely effortless.

For example, how easy is it to create each of these?

### **Case A: A trivial single-pane screen**

```
┌────────────────────────────────┐
│                                │
│           Content              │
│                                │
└────────────────────────────────┘
```

Does it require a screen definition, layout declaration, pane definition, cursor tracking, and registration?

If so, how much ceremony is necessary?

### **Case B: A conventional list/detail screen**

```
┌──────────────────┬─────────────┐
│                  │             │
│   List           │  Details    │
│                  │             │
└──────────────────┴─────────────┘
```

How many declarations? Can the developer express this in a few lines without sacrificing the ability to customize it?

### **Case C: A screen with a temporary third pane**

```
┌──────────────────┬─────────────┐
│                  │             │
│   List           │  Details    │
│                  ├─────────────┤
│                  │  Preview    │
└──────────────────┴─────────────┘
```

Is this easy, or does the developer have to create a new composition type?

### **Case D: Change layout dynamically**

A user presses a key:

```
Two-pane → Three-pane
```

Can the developer do this without reconstructing the entire screen or losing cursor state?

That's where the difference between Good and Excellent begins.

greenfield.md

# **4. What would make composition Excellent?**

I would define Excellent as:

> A developer can express the common TUI layouts in seconds, modify them without rewriting structure, and retain full control when the composition becomes unusual.

That requires five improvements.

## **A. Make the common case dramatically shorter**

Your current example is already reasonably concise:

```
app := kit.New(
    kit.Screen("jobs", kit.Min(100, 20),
        kit.Row(
            kit.Pane("queue", &QueuePane{jobs: store}).Fill(),
            kit.Cols(
                kit.Pane("detail", &DetailPane{}).Fixed(42),
                kit.Pane("log", &LogPane{}).Fill(),
            ),
        ),
    ),
)
```

But count the concepts the developer has to understand:

- Screen
- Min
- Row
- Pane
- Cols
- Fill
- Fixed

For a first-time user, that is seven concepts to produce a familiar layout.

The goal shouldn't necessarily be to reduce the number of types. It should be to reduce the number of *decisions* required for ordinary tasks.

### **Potential improvement: common composition presets**

Consider offering shortcuts for conventional layouts.

```
kit.Split(
    queue,
    detail,
)
```

Or:

```
kit.ListDetail(
    queue,
    detail,
)
```

Or perhaps:

```
kit.Screen("jobs",
    kit.ListDetail(
        kit.Pane("queue", queue),
        kit.Pane("detail", detail),
    ),
)
```

I would be careful not to introduce a large collection of layout-specific constructors. The abstraction should be based on common structural patterns, not every possible arrangement.

Potential useful presets:

| Pattern         | Typical use                      |
| --------------- | -------------------------------- |
| Single          | Simple screen                    |
| Split           | Two panes                        |
| ListDetail      | Navigation + selected item       |
| Sidebar         | Narrow navigation + main content |
| Stack           | Vertical arrangement             |
| MainWithPreview | Main view + secondary preview    |

The underlying `Row` and `Cols` remain the universal primitives.

**Recommendation:** build 2–3 convenience compositions only if real applications repeatedly need them.

## **B. Make layout intent obvious, rather than requiring geometry thinking**

The current system uses:

```
Fill()
Fixed(42)
Ratio(a, b)
```

These are good primitives, but developers often think in terms of *intent*:

- The sidebar should be narrow.
- The list should take approximately 30%.
- The detail pane should take the rest.
- The preview should appear only when needed.
- The layout should adapt to small terminals.

You can improve the experience by making the composition API more intention-oriented.

For example:

```
kit.Cols(
    kit.Sidebar(queue).Width(30),
    kit.Main(detail).Fill(),
)
```

Or:

```
kit.Cols(
    kit.FixedPane(queue, 30),
    kit.FlexiblePane(detail),
)
```

The important thing is not the syntax. It is the mental model:

> Developers describe the role and constraints of a pane, not manually calculate every rectangle.

Your current geometry engine is already deterministic and simple, which is good. I would not introduce sophisticated CSS-style layout negotiation.

Instead, improve the *declarative ergonomics* of your existing constraint system.

## **C. Make composition easy to change**

This is one of the biggest opportunities.

A good composition API should make this trivial:

```
Normal:
┌──────────┬─────────────────────┐
│ List     │ Details             │
└──────────┴─────────────────────┘

Expanded:
┌──────────┬─────────────────────┐
│ List     │ Details             │
│          ├─────────────────────┤
│          │ Logs                │
└──────────┴─────────────────────┘
```

For example:

```
func (p *JobsScreen) Layout() kit.Layout {
    if p.showLogs {
        return kit.Row(
            kit.Pane("queue", p.queue).Fixed(30),
            kit.Cols(
                kit.Pane("detail", p.detail).Fill(),
                kit.Pane("logs", p.logs).Fill(),
            ),
        )
    }

    return kit.Cols(
        kit.Pane("queue", p.queue).Fixed(30),
        kit.Pane("detail", p.detail).Fill(),
    )
}
```

This is not necessarily the exact API you should adopt. The architectural question is:

**Can composition be a function of application state without forcing the framework to become stateful or retained-mode?**

Your spec emphasizes static wiring and runtime state. That separation is valuable. I would explore allowing the layout expression itself to be rebuilt deterministically based on state, while retaining the invariants:

- Terminal size has one owner.
- Pane identity is stable.
- Focus/cursor state doesn't accidentally reset.
- Geometry remains deterministic.
- Behavior still belongs to panes.
- Dynamic layouts don't introduce dynamic routing tables.

The most important requirement is stable identity.

If the queue pane disappears temporarily and then returns, should its cursor state return? If it changes position in the tree, should it still be the same pane? These should be clearly defined.

### **My recommendation**

Do not add a full dynamic layout engine.

Instead, support a small number of composition alternatives, with stable pane IDs and explicit lifecycle semantics.

## **D. Provide sensible defaults for the boring things**

Excellent composition doesn't make developers specify everything.

Consider:

```
kit.Screen("jobs",
    kit.Cols(
        kit.Pane("queue", queue),
        kit.Pane("detail", detail),
    ),
)
```

Could the system sensibly infer:

- First pane receives focus initially.
- Both panes are flexible.
- The split is evenly distributed.
- A pane has a default minimum.
- The screen has a reasonable default minimum.
- Default chrome is applied consistently.

Then the developer only overrides when needed:

```
kit.Screen("jobs",
    kit.Cols(
        kit.Pane("queue", queue).Ratio(1, 2),
        kit.Pane("detail", detail).Ratio(2, 3),
    ),
)
```

There is an important trade-off here.

You have a philosophy of explicitness, auditability, and checked promises. Don't undermine it with mysterious defaults.

I would distinguish:

**Safe ergonomic defaults:**

- Default focus
- Equal flexible splits
- Standard chrome
- Default minimums
- Default pane behavior

**Potentially dangerous implicit behavior:**

- Automatically changing focus based on layout
- Silently resizing fixed panes
- Automatically hiding panes when dimensions become small
- Implicitly changing key scopes
- Implicitly resetting state

Defaults should remove ceremony, not create hidden behavior.

## **E. Make pane reuse and screen reuse natural**

Your model currently treats panes as leaves, which is a good architectural constraint. But users should be able to reuse pane implementations easily.

Imagine someone building:

- A list of jobs
- A list of containers
- A list of deployments
- A list of queues

All four may share the same interaction pattern, but have different data.

Ideally, composition doesn't force them to create four entirely separate pane implementations.

A reusable pane could be parameterized:

```
type ListPane[T any] struct {
    Items    []T
    Selected func(T) string
    Render   func(T, kit.Slot) string
}
```

Or, in a more idiomatic design, the application could implement the existing `Pane` interface and use composition internally.

The kit itself shouldn't become a generic widget library. That's explicitly outside its purpose. But the composition experience should make it easy to reuse *application-defined panes*.

This is an important distinction:

> Kit should make composition easy; it shouldn't try to own every reusable component.

## **5. The biggest opportunity: composition should feel like a visual language**

I think the best version of this project would let developers read composition declarations almost as if they were looking at a wireframe.

For example:

```
kit.Screen("deployments",
    kit.Layout(
        kit.Sidebar(
            kit.Pane("clusters", clusters),
        ),
        kit.Main(
            kit.Stack(
                kit.Pane("deployments", deployments),
                kit.Pane("events", events),
            ),
        ),
    ),
)
```

A developer should be able to glance at that and immediately understand:

```
Deployments
┌─────────────┬─────────────────────────────┐
│ Clusters    │ Deployments                │
│             ├─────────────────────────────┤
│             │ Events                      │
└─────────────┴─────────────────────────────┘
```

This is where composition becomes excellent: **the API communicates the resulting structure with very little translation required in the developer's head.**

However, there is a potential trap.

Do not build an elaborate DSL just to make declarations look pretty. If `Sidebar`, `Main`, `Stack`, `Layout`, `Split`, `Region`, `Area`, and `Group` all become synonyms for layout constructors, the vocabulary becomes worse.

I'd aim for:

- `Row`
- `Cols`
- `Pane`
- `Title`
- `Fill`
- `Fixed`
- `Ratio`

These are already strong primitives.

Then add convenience abstractions only where they demonstrably reduce common code.

## **6. What I would actually change in your project**

I would prioritize these changes in this order:

| Priority | Improvement                                          | Why                                              |
| -------- | ---------------------------------------------------- | ------------------------------------------------ |
| 1        | Make the simplest screen minimal                     | First impression and adoption                    |
| 2        | Add 2–3 common composition recipes                   | Reduce repetitive layout declarations            |
| 3        | Define stable pane identity and lifecycle            | Enable dynamic composition without state bugs    |
| 4        | Test composition under resize                        | Real-world terminal behavior                     |
| 5        | Improve layout diagnostics                           | Developers understand why the layout doesn't fit |
| 6        | Add convenience APIs only after migration experience | Avoid premature DSL expansion                    |

### **1. Define a composition complexity benchmark**

Take five layouts that real TUI developers routinely need:

**Layout 1 — Simple**

```
┌────────────────────┐
│ Single content     │
└────────────────────┘
```

**Layout 2 — List/detail**

```
┌──────────┬─────────┐
│ List     │ Detail  │
└──────────┴─────────┘
```

**Layout 3 — Three-pane**

```
┌────────┬──────────┬──────────┐
│ List   │ Detail   │ Logs     │
└────────┴──────────┴──────────┘
```

**Layout 4 — Nested**

```
┌──────────┬───────────────────┐
│ Sidebar  │ Main              │
│          ├───────────────────┤
│          │ Bottom panel      │
└──────────┴───────────────────┘
```

**Layout 5 — Adaptive**

```
Wide terminal:
┌──────────┬───────────────────┐
│ List     │ Detail            │
└──────────┴───────────────────┘

Narrow terminal:
┌──────────────────────────────┐
│ List                         │
├──────────────────────────────┤
│ Detail                       │
└──────────────────────────────┘
```

Then measure your API against an equivalent implementation using plain Bubble Tea/Lipgloss.

Track:

- Lines of code
- Number of concepts required
- Number of custom layout functions
- Time to implement
- Time to change the layout
- Bugs during resize
- State lost during transitions
- Developer comprehension after reading the code

This is more useful than asking whether your API is elegant.

## **7. What would make me change the rating?**

## **My Excellent threshold**

A developer should be able to create a standard screen in under a minute, understand the layout at a glance, and modify it without touching interaction logic.

1. Simple screen requires minimal declaration.
2. List/detail and common splits are concise.
3. Nested layouts remain readable.
4. Changing composition doesn't reset pane state accidentally.
5. Resize behavior is deterministic and diagnosable.
6. Unusual layouts can use the primitives without fighting the framework.
7. Composition is independent from routing and action logic.

## **My actual recommendation**

I wouldn't redesign your architecture to improve this dimension. **Your composition primitives are probably already sufficient.**

I would improve the experience *around* them:

1. Make the default declarations less ceremonious.
2. Create a cookbook of common patterns.
3. Test the composition API against real applications.
4. Define the lifecycle and identity rules of panes in dynamic layouts.
5. Add layout diagnostics that tell the developer what constraint caused the problem.
6. Resist building a widget framework or elaborate layout DSL.

The key insight is that composition is likely a **table-stakes capability**, not the product's primary differentiator.

Your project will win more strongly if composition is *good enough that users barely think about it*, while routing, action consistency, generated discoverability, auditing, and deterministic testing are things they cannot easily get elsewhere.

That is why I rated it **Good**: structurally strong, useful, and well-designed—but not yet demonstrated to be exceptionally frictionless. I'd move it to **Excellent** after proving that the common layouts are genuinely easier to build and evolve than the alternatives.