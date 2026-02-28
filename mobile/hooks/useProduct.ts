import { useState, useEffect } from "react";
import { getProduct } from "@/lib/api";
import { getMockProduct } from "@/lib/mock";
import type { ProductResponse } from "@/lib/types";

const USE_MOCK = __DEV__;

export function useProduct(id: string, lang: string) {
  const [product, setProduct] = useState<ProductResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function fetchProduct() {
      setLoading(true);
      setError(null);
      try {
        const data = USE_MOCK
          ? getMockProduct(id)
          : await getProduct(id, lang);
        if (!cancelled) setProduct(data);
      } catch (err) {
        if (!cancelled) {
          setError(
            err instanceof Error ? err.message : "Failed to load product",
          );
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    }

    fetchProduct();
    return () => {
      cancelled = true;
    };
  }, [id, lang]);

  return { product, loading, error };
}
