import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";

interface BlindModeValue {
  enabled: boolean;
  toggle: () => void;
}

const BlindModeContext = createContext<BlindModeValue>({
  enabled: false,
  toggle: () => undefined,
});

const STORAGE_KEY = "vqf.blind-mode";

/**
 * "No-Eyes Experience": hides product images so sighted visitors experience
 * the store the way a screen-reader user does. State is exposed through
 * data-blind-mode on <html> (CSS does the hiding) and persisted.
 */
export function BlindModeProvider({ children }: { children: ReactNode }) {
  const [enabled, setEnabled] = useState<boolean>(() => {
    try {
      return localStorage.getItem(STORAGE_KEY) === "on";
    } catch {
      return false;
    }
  });

  useEffect(() => {
    document.documentElement.dataset.blindMode = enabled ? "on" : "off";
    try {
      localStorage.setItem(STORAGE_KEY, enabled ? "on" : "off");
    } catch {
      // storage unavailable: mode still applies for this session
    }
    return () => {
      delete document.documentElement.dataset.blindMode;
    };
  }, [enabled]);

  const toggle = useCallback(() => setEnabled((v) => !v), []);
  const value = useMemo(() => ({ enabled, toggle }), [enabled, toggle]);

  return (
    <BlindModeContext.Provider value={value}>{children}</BlindModeContext.Provider>
  );
}

export function useBlindMode(): BlindModeValue {
  return useContext(BlindModeContext);
}
