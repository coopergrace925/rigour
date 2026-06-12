'use client';

import { Search, Activity } from 'lucide-react';
import { Input } from './ui/input';
import { Button } from './ui/button';
import { useRouter } from 'next/navigation';
import { useState } from 'react';
import Link from 'next/link';

interface SearchHeaderProps {
  initialQuery?: string;
}

export function SearchHeader({ initialQuery = '' }: SearchHeaderProps) {
  const [query, setQuery] = useState(initialQuery);
  const router = useRouter();

  // Parse query syntax like "field: value" into filter parameters
  const parseQuery = (queryString: string): Record<string, string> | null => {
    // Look for pattern like "field: value" or "field:value"
    const match = queryString.match(/(\w+(?:\.\w+)*)\s*:\s*(.+)/);

    if (match) {
      const [, field, value] = match;
      return { [field]: value.trim() };
    }

    return null;
  };

  const handleSearch = () => {
    const params = new URLSearchParams();

    if (query) {
      // Pass the raw query as 'q' parameter (Shodan-style)
      params.append('q', query);
    }

    router.push(`?${params.toString()}`);
  };

  const handleKeyPress = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      handleSearch();
    }
  };

  return (
    <header className="space-y-6">
      <div className="flex flex-col md:flex-row md:items-center justify-between border-b border-border pb-6 gap-4">
        <div>
          <h1 className="tracking-tighter text-foreground glow-text text-5xl font-extrabold font-mono">
            RIGOUR_
          </h1>
          <p className="text-xs text-muted-foreground uppercase tracking-widest mt-1">
            // TELEMETRY RECONNAISSANCE SYSTEM
          </p>
        </div>
        <div className="flex flex-col gap-2">
          <div className="flex flex-wrap gap-2 text-[10px] font-mono border border-border p-2 bg-black/40">
            <div className="flex items-center gap-1.5 px-2 py-0.5 border-r border-border">
              <span className="h-2 w-2 rounded-full bg-emerald-500 animate-pulse"></span>
              <span className="text-muted-foreground">SYS_STATUS:</span>
              <span className="text-emerald-500 font-bold">ACTIVE</span>
            </div>
            <div className="flex items-center gap-1.5 px-2 py-0.5 border-r border-border">
              <span className="text-muted-foreground">ZMAP_SWEEP:</span>
              <span className="text-foreground">10.4M PPS</span>
            </div>
            <div className="flex items-center gap-1.5 px-2 py-0.5 border-r border-border">
              <span className="text-muted-foreground">BLOCKLIST:</span>
              <span className="text-foreground">RFC1918 + OPT-OUT</span>
            </div>
            <div className="flex items-center gap-1.5 px-2 py-0.5">
              <span className="text-muted-foreground">PORT_COVERAGE:</span>
              <span className="text-foreground">65,535 (FULL)</span>
            </div>
          </div>
          <Link href="/health">
            <Button size="sm" variant="outline" className="w-full gap-2 rounded-none border-border hover:border-primary text-foreground uppercase tracking-widest text-[10px]">
              <Activity className="h-3 w-3" />
              SYSTEM_HEALTH_DASHBOARD
            </Button>
          </Link>
        </div>
      </div>

      <div className="space-y-2">
        <p className="text-xs text-muted-foreground">
          Enter Shodan-style queries (e.g., <code className="bg-secondary px-1 py-0.5 border border-border rounded text-foreground font-mono">port:22 country:US</code> or <code className="bg-secondary px-1 py-0.5 border border-border rounded text-foreground font-mono">cve:CVE-2023-44487 org:Google</code>)
        </p>
      </div>

      <div className="flex gap-2">
        <div className="relative flex-1 scan-sweep border border-border">
          <Search className="absolute left-4 top-1/2 -translate-y-1/2 h-5 w-5 text-muted-foreground" />
          <Input
            type="text"
            placeholder="e.g. port:22 country:US apache"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyPress={handleKeyPress}
            className="pl-12 h-14 bg-black/60 border-0 text-white font-mono placeholder:text-muted-foreground/60 focus:ring-0 focus:outline-none"
          />
        </div>
        <Button
          onClick={handleSearch}
          className="h-14 px-8 bg-primary hover:bg-primary/80 font-mono text-black font-bold uppercase tracking-wider"
        >
          EXECUTE_SEARCH
        </Button>
      </div>
    </header>
  );
}
