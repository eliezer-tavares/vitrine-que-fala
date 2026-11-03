import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";

import { api, type Product } from "./api";
import { AltTextDemo } from "./components/AltTextDemo";
import { AnnouncerProvider } from "./components/LiveRegion";
import { Header } from "./components/Header";
import { ProductDialog } from "./components/ProductDialog";
import { ProductList } from "./components/ProductList";
import { SearchBox } from "./components/SearchBox";
import { SkipLink } from "./components/SkipLink";
import { BlindModeProvider } from "./context/BlindModeContext";

export default function App() {
  const { t, i18n } = useTranslation();
  const [products, setProducts] = useState<Product[]>([]);
  const [loadError, setLoadError] = useState(false);
  const [selected, setSelected] = useState<Product | null>(null);

  useEffect(() => {
    let cancelled = false;
    setLoadError(false);
    api
      .listProducts(i18n.language)
      .then((res) => {
        if (!cancelled) setProducts(res.items);
      })
      .catch(() => {
        if (!cancelled) setLoadError(true);
      });
    return () => {
      cancelled = true;
    };
  }, [i18n.language]);

  return (
    <BlindModeProvider>
      <AnnouncerProvider>
        <SkipLink />
        <Header />
        <main id="main" tabIndex={-1}>
          {loadError ? (
            <p role="alert">{t("app.loadError")}</p>
          ) : (
            <>
              {/* v2.0: multilingual semantic search filters the list in
                  place; switching language reloads the full catalog. */}
              <SearchBox onResults={setProducts} />
              <ProductList products={products} onOpen={setSelected} />
            </>
          )}
          <AltTextDemo />
        </main>
        <ProductDialog product={selected} onClose={() => setSelected(null)} />
        <footer>
          <p>
            {t("footer.architectedBy")} — contact@eliezertavaresdeoliveira.com
          </p>
          <p>{t("footer.tagline")}</p>
        </footer>
      </AnnouncerProvider>
    </BlindModeProvider>
  );
}
