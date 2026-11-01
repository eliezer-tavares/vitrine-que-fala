import { useTranslation } from "react-i18next";

import type { Product } from "../api";
import { ProductCard } from "./ProductCard";

interface Props {
  products: Product[];
  onOpen: (product: Product) => void;
}

export function ProductList({ products, onOpen }: Props) {
  const { t } = useTranslation();
  return (
    <section aria-labelledby="products-title">
      <h2 id="products-title">{t("products.title")}</h2>
      <ul className="product-list">
        {products.map((product) => (
          <ProductCard key={product.id} product={product} onOpen={onOpen} />
        ))}
      </ul>
    </section>
  );
}
