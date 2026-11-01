import { useTranslation } from "react-i18next";

import type { Product } from "../api";

interface Props {
  product: Product;
  onOpen: (product: Product) => void;
}

export function ProductCard({ product, onOpen }: Props) {
  const { t } = useTranslation();
  return (
    <li className="product-card">
      <h3>{product.name}</h3>
      <img
        className="product-image"
        src={`https://placehold.co/480x300?text=${encodeURIComponent(product.name)}`}
        alt={product.alt_text}
        loading="lazy"
        width={480}
        height={300}
      />
      <p className="price">{product.price.formatted}</p>
      <button type="button" onClick={() => onOpen(product)}>
        {t("products.details", { name: product.name })}
      </button>
    </li>
  );
}
