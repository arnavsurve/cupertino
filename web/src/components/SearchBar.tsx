"use client";

import { useState, useRef, useEffect } from "react";
import type { PackageInfo } from "@/lib/types";

export default function SearchBar() {
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<PackageInfo[] | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(false);
  const [visible, setVisible] = useState(false);
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const formRef = useRef<HTMLFormElement>(null);

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (formRef.current && !formRef.current.contains(e.target as Node)) {
        setTimeout(() => setVisible(false), 200);
      }
    }
    document.addEventListener("click", handleClickOutside);
    return () => document.removeEventListener("click", handleClickOutside);
  }, []);

  function handleInput(value: string) {
    setQuery(value);
    if (timeoutRef.current) clearTimeout(timeoutRef.current);

    if (value.trim().length < 2) {
      setResults(null);
      setVisible(false);
      return;
    }

    timeoutRef.current = setTimeout(() => performSearch(value.trim()), 300);
  }

  async function performSearch(q: string) {
    setLoading(true);
    setError(false);
    setVisible(true);

    try {
      const res = await fetch(`/api/search?q=${encodeURIComponent(q)}&limit=10`);
      const data = await res.json();
      setResults(data);
    } catch {
      setError(true);
      setResults(null);
    } finally {
      setLoading(false);
    }
  }

  return (
    <>
      <form
        ref={formRef}
        onSubmit={(e) => {
          e.preventDefault();
          if (query.trim().length >= 2) performSearch(query.trim());
        }}
      >
        <input
          type="text"
          placeholder="search packages..."
          value={query}
          onChange={(e) => handleInput(e.target.value)}
        />
        <button type="submit">search</button>
      </form>

      {visible && (
        <div className="search-results">
          {loading && <div className="search-loading">searching...</div>}
          {error && <div className="search-error">search failed</div>}
          {!loading && !error && results && results.length === 0 && (
            <div className="search-no-results">no packages found</div>
          )}
          {!loading &&
            !error &&
            results &&
            results.map((pkg) => (
              <div key={pkg.name} className="search-result">
                <div>
                  <a href={`/packages/${pkg.name}`}>{pkg.name}</a>
                </div>
                <div>{pkg.description}</div>
                <div>
                  v{pkg.latest} | {pkg.downloads} downloads
                </div>
                <span className="install-command">
                  cupertino install {pkg.name}
                </span>
              </div>
            ))}
        </div>
      )}
    </>
  );
}
