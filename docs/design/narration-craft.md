# Prosody narration craft

The Go rewrite composes Narrate's conversational craft with Prosody's briefing
policy. The debug Flutter live provider has its own bounded rewrite prompt;
it does not dynamically load local skills. Both now include the same adapted
single-listener craft guidance. Supplied audio/video remains verbatim.

Adapted from David's skills:
- `voice-conversation-craft`: contractions, short thought groups, varied rhythm,
  specific responses, precision where it matters.
- `podcast-craft-emotional-pacing`: give weighty information room to land and
  match tone to the source. Use punctuation, not unsupported performance tags.
- `podcast-craft-concreteness`: retain source-supported details and meaningful
  numbers; do not invent scenes, examples or listener circumstances.
- `podcast-craft-narrative-spine`: connect change, significance and an explicit
  next step when the material warrants that structure. Never force an arc.
- `podcast-craft-rapport`: accessible language and specificity convey attention;
  fabricated personal memories, intimate familiarity and imaginary dialogue do not.

Cold-open suspense, interview questions, vulnerability elicitation, teasing,
false starts and backchannels are not appropriate defaults for a one-way agent
briefing. Do not hide actionable news behind a hook or pretend a person is there.
Skill files inform checked-in instructions; they are not sent to the provider.

Illustrative editorial target (authored example, not generated/audio evidence):

Source: “The launch has moved from Tuesday to Thursday. Two checkout defects are
still unresolved. Maya will retest at 15:00. Thursday remains provisional.”

Briefing: “The launch is moving to Thursday. Two checkout problems still need
fixing, and Maya will retest at fifteen hundred. Thursday still isn't confirmed.”

Preserve timezone/date ambiguity rather than inventing “this afternoon” or a
local timezone. Natural phrasing must not change meaning.

Listen for clarity, varied cadence, appropriate warmth, retained uncertainty and
absence of canned enthusiasm. Compare script with source before judging voice.
Prompt changes alone do not prove expressive synthesis; Laura's delivered audio
still needs listening acceptance. Existing bundled macOS samples do not change.
Output/byte/duration limits, model/voice choices and the existing preview spending
allowance remain in force. No automatic generation or extra provider request is
introduced by this change.
