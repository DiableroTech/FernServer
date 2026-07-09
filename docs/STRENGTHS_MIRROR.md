# Fern — Feature Idea: Strengths & Interests Mirror

## The concept

Fern isn't just a journal — it's an **active mirror** for the things you forget about yourself. Over time, through conversations, Fern builds a picture of:

- **Strengths** — things you're good at, especially the ones you dismiss or don't notice ("hidden strengths")
- **Interests** — what lights you up, what you come back to
- **Values** — what matters to you when you cut through the noise
- **Wins** — moments you were proud, even small ones
- **Growth edges** — the stuff you're working on, not as failings but as directions

Then, like a friend who actually *remembers* — Fern brings these back to you when you need to hear them. Not performatively, but naturally, inside a session:

> "You mentioned last week you stayed calm when your manager pushed back — that's the composure you said you wanted. Did you notice that?"

> "You've talked about woodworking three times now and every time your energy completely shifts. I think that means something."

## How it works (technically)

### Phase 1 — Extraction via summaries (cheap, now)

The session summary prompt already generates insights. We enhance it to also extract:
- `strengths_observed: string[]` — strengths Fern noticed the person demonstrating
- `interests_surfaced: string[]` — things they lit up talking about
- `values_expressed: string[]` — what seemed to matter to them

These get stored on the session and aggregated into the memory context.

### Phase 2 — Dedicated profile (later)

A `user_profile` table accumulates these over time:
- Each strength/interest/value has a `frequency` (how often it comes up) and `last_mentioned` date
- Fern's memory injection includes: "Here's what I know about this person's strengths and interests..." — separate from session recaps
- A new screen in the app: **"What Fern sees in you"** — a living document of your strengths, interests, and growth, always updating

### Phase 3 — Active surfacing

Fern doesn't just store this passively:
- If someone's in a low moment, Fern may naturally reflect a relevant strength back
- If someone hasn't mentioned an interest they used to light up about, Fern may gently ask about it
- Weekly reports include a "Strengths spotlight" section

## UX — teaching users how to talk to Fern

On first login or in onboarding:

> "The more you share with Fern — your ups and downs, what you're into, what you're good at, what you're struggling with — the better it gets at reflecting back the things you've forgotten about yourself. Talk to it like a friend who actually listens."

This framing matters: it's not "journaling prompts" — it's "Fern is your buddy who remembers shit you've forgotten."

## Priority

This is a **v2 feature** that builds naturally on the session model + memory injection we already have. The extraction piece could land as soon as we enhance the summary prompt. The profile screen and active surfacing come later.
