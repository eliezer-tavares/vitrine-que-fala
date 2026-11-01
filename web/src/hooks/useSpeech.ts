import { useCallback } from "react";

export interface SpeechStatus {
  spoken: boolean;
  reason?: "speech-unavailable" | "no-voice-for-locale";
}

/**
 * Web Speech API playback with the fallback chain from ADR-0006:
 * 1. exact voice for the locale, 2. same language prefix,
 * 3. default-locale voice, 4. if speech is unavailable at all, report it so
 * the UI can keep the text visible (speech is an enhancement, never the only
 * channel).
 */
export function useSpeech(): (text: string, locale: string) => SpeechStatus {
  return useCallback((text: string, locale: string): SpeechStatus => {
    if (typeof window === "undefined" || !("speechSynthesis" in window)) {
      return { spoken: false, reason: "speech-unavailable" };
    }
    const synth = window.speechSynthesis;
    const normalize = (lang: string) => lang.replace("_", "-").toLowerCase();
    const voices = synth.getVoices();
    const prefix = locale.split("-")[0].toLowerCase();
    const exact = voices.find((v) => normalize(v.lang) === locale.toLowerCase());
    const partial = voices.find((v) => normalize(v.lang).startsWith(prefix));
    const fallback = voices.find((v) => normalize(v.lang).startsWith("en"));
    const voice = exact ?? partial ?? fallback;

    if (!voice) {
      // Some browsers populate voices asynchronously; without any voice we
      // degrade gracefully instead of speaking with a random default.
      return { spoken: false, reason: "no-voice-for-locale" };
    }

    const utterance = new SpeechSynthesisUtterance(text);
    utterance.lang = voice.lang;
    utterance.voice = voice;
    utterance.rate = 1;
    synth.cancel();
    synth.speak(utterance);
    return { spoken: true };
  }, []);
}
