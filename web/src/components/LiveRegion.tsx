import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useState,
  type ReactNode,
} from "react";

type Announce = (message: string) => void;

const AnnounceContext = createContext<Announce>(() => undefined);

/**
 * Global polite live region: async UI events (mode changes, language
 * switches) are announced to screen readers instead of only being shown.
 */
export function AnnouncerProvider({ children }: { children: ReactNode }) {
  const [message, setMessage] = useState<string | null>(null);
  const announce = useCallback((msg: string) => setMessage(msg), []);
  const value = useMemo(() => announce, [announce]);

  return (
    <AnnounceContext.Provider value={value}>
      {children}
      <div role="status" aria-live="polite" className="visually-hidden">
        {message}
      </div>
    </AnnounceContext.Provider>
  );
}

export function useAnnouncer(): Announce {
  return useContext(AnnounceContext);
}
