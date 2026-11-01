import { useEffect, useRef, type KeyboardEvent as ReactKeyboardEvent } from "react";
import { useTranslation } from "react-i18next";

import type { Product } from "../api";

interface Props {
  product: Product | null;
  onClose: () => void;
}

const FOCUSABLE =
  'a[href], button:not([disabled]), textarea, input, select, [tabindex]:not([tabindex="-1"])';

/**
 * Accessible modal dialog: focus moves in on open, Tab is trapped, Escape
 * closes, and focus returns to the element that opened it. Implements the
 * pattern asserted by the keyboard-only E2E suite.
 */
export function ProductDialog({ product, onClose }: Props) {
  const { t } = useTranslation();
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!product) return;
    const opener = document.activeElement as HTMLElement | null;
    const first = containerRef.current?.querySelector<HTMLElement>(FOCUSABLE);
    first?.focus();
    return () => opener?.focus();
  }, [product]);

  if (!product) return null;

  const handleKeyDown = (event: ReactKeyboardEvent<HTMLDivElement>) => {
    if (event.key === "Escape") {
      onClose();
      return;
    }
    if (event.key !== "Tab" || !containerRef.current) return;
    const items = Array.from(
      containerRef.current.querySelectorAll<HTMLElement>(FOCUSABLE),
    );
    if (items.length === 0) return;
    const first = items[0];
    const last = items[items.length - 1];
    if (event.shiftKey && document.activeElement === first) {
      event.preventDefault();
      last?.focus();
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault();
      first?.focus();
    }
  };

  return (
    <div className="dialog-overlay" onKeyDown={handleKeyDown}>
      <div
        className="dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby="dialog-title"
        ref={containerRef}
      >
        <h2 id="dialog-title">{product.name}</h2>
        <img
          className="product-image"
          src={`https://placehold.co/480x300?text=${encodeURIComponent(product.name)}`}
          alt={product.alt_text}
          width={480}
          height={300}
        />
        <p>{product.description}</p>
        <p className="price">{product.price.formatted}</p>
        <button type="button" onClick={onClose}>
          {t("dialog.close")}
        </button>
      </div>
    </div>
  );
}
