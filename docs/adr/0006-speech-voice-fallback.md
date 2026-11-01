# ADR-0006 — Web Speech voice fallback matrix

Status: Accepted (v1.0)
Date: 2026
Deciders: Eliezer Tavares de Oliveira

## Context

The "No-Eyes Experience" and the alt-text demo read descriptions aloud with the Web Speech API. Voice availability for a locale is **not guaranteed** — it depends on OS/browser installed voices (Chrome ships Google voices; Windows adds SAPI voices; Linux varies). A screen-reader user hitting a missing voice gets silence or gibberish-pronounced text.

## Decision

A single `useSpeech(text, locale)` hook with a deterministic fallback chain:

1. Pick an installed voice whose `lang` matches the selected locale (exact `pt-BR` before prefix `pt`).
2. If none, fall back to the **default locale voice** (`en-US`) and keep the utterance `lang` as `en-US` so pronunciation is at least consistent.
3. If `speechSynthesis` is unavailable at all (some environments/AT combos), return a status the UI uses to show the text on screen instead of pretending to speak.
4. The chosen voice (or fallback reason) is always discoverable: the UI keeps the text visible regardless of speech success — speech is an enhancement, never the only channel.

The tested voice matrix per OS/browser is documented in the README as coverage grows.

## Consequences

- No silent failures; the text is always visible, satisfying "speech is enhancement" (WCAG 1.4.x-friendly).
- The fallback logic is unit-tested, including the "no speechSynthesis" path (jsdom has none).
- Cost: slight complexity in the demo UI to surface fallback status — deliberate.

## Links

- ADR-0003 (language of page/parts guarantees the reading voice matches content whenever a real voice exists).
