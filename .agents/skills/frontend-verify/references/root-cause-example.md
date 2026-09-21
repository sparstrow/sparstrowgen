# Root-causing a finding — a worked example

This is the pattern step 5 of the main loop points at: don't fix the symptom,
find the actual cause, and know the difference between a real bug and a
limitation of the tool you're testing with. Both matter, and confusing them
in either direction is a real failure mode — patching a harness quirk as if
it were a product bug wastes time and can introduce a regression; waving away
a real bug as "just the test environment" ships it.

## The finding

A prototype has an inline-rename field: click a name, an `<input>` appears,
Enter commits the new value, Escape cancels. Testing it by dispatching a
keydown event —

```js
input.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter" }));
```

— the rename silently doesn't happen. The row still shows the old name.

## Resist the first fix

The fast, wrong move here is to patch the test: dispatch a `blur` event
manually after the keydown, see the rename now happen, and move on. That
"fixes" the symptom in the harness without establishing whether the actual
product code has the same problem for a real user, or whether the harness
just can't simulate this one interaction correctly. Shipping a fix aimed at
the test instead of the code is how a real bug survives verification.

## Isolate before diagnosing

Strip it down. A bare `<input>` with nothing else, one `blur` listener:

```js
const el = document.createElement("input");
document.body.appendChild(el);
el.focus();
let fired = 0;
el.addEventListener("blur", () => fired++);
el.blur();
// fired: 0
```

Even a direct `.blur()` call on an isolated element doesn't fire the
listener. That's not plausible as a bug in the app's rename logic — it points
at something upstream of the app entirely. Checking `document.hasFocus()`
confirms it: `false`. In this particular automated browser pane, the document
itself never receives real OS-level focus, so focus/blur transitions between
elements don't fire the way they would for an actual user in an actual
window. **That's a genuine, confirmed environment limitation** — not a guess,
established by an isolated test that removes the app entirely from the
picture.

## But don't stop there

Knowing the harness can't trigger blur here doesn't mean the finding is
closed — it means the *test method* was wrong, and the underlying question
("does Enter actually commit the rename?") is still open. Go read the real
implementation, if one exists to compare against — an existing component
this prototype is modeling, a similar pattern elsewhere in the codebase.

In this case, the real component this prototype was based on turned out to
call its `commit()` function **directly** from the Enter keydown handler, and
wired `blur` separately, only for the click-away case. The prototype had
instead routed Enter through `input.blur()`, relying on the blur event to do
the committing — a real, unforced difference from the source, not a harness
artifact. It happened to work for a mouse-driving human (native blur fires
fine in a real browser) but was needlessly fragile, and broke the moment a
test tried to drive it programmatically.

## Fix at the right layer

The fix matches the real source: call `commit()`/`cancel()` directly from the
keydown handler, and keep `blur` wired only as an additional (now
idempotent — guarded so it can't double-fire) trigger for click-away. Not a
patch aimed at making the dispatched event succeed — a change that makes the
Enter path work the same way, for the same reason, as the pattern it was
modeling.

## Report both halves

The eventual report named both things, because collapsing them into one line
would hide information a reader needs:

- **Environment caveat**: this browser pane doesn't grant the document real
  focus, so blur-only interactions can't be exercised by scripted events here
  — confirmed via an isolated element test, not assumed.
- **Real bug found & fixed**: the Enter-key path didn't match its source
  component's pattern and was fixed to match it, independent of the harness
  finding.

If the isolation test had shown `document.hasFocus()` was `true` and blur
still didn't fire, that would have pointed the other way — a real bug in the
component's event wiring, not a harness limitation. The isolation step is
what tells you which one you're looking at; skipping it means guessing.
