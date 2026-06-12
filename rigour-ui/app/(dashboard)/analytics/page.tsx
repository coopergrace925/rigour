'use client';

import { useEffect, useState } from 'react';
import { Card, CardContent } from '../../../../components/ui/card';
import Link from 'next/link';
import { Button } from '../../../../components/ui/button';
import { ChevronLeft, TrendingUp, Shield, Network, Database } from 'lucide-react';

interface AnalyticsOverview {
  total_scans: number;
  unique_hosts: number;
  top_services: ServiceStat[];
  recent_cves: CVETrend[];
  top_asns: ASNStat[];
}

interface ServiceStat {
  service: string;
  port: number;
  host_count: number;
  total_scans: number;
}

interface CVETrend {
  date: string;
  cve_id: string;
  affected_hosts: number;
}

interface ASNStat {
  asn: number;
  country: string;
  scan_count: number;
  unique_ips: number;
}

export default function AnalyticsPage() {
  const [overview, setOverview] = useState<AnalyticsOverview | null>(null);
  const [loading, setLoading] = useState(true);

  const API_BASE = process.env.NEXT_PUBLIC_API_BASE_URL || 'http://localhost:8080';

  useEffect(() => {
    const fetchData = async () => {
      try {
        const res = await fetch(`${API_BASE}/api/analytics/overview`);
        const data = await res.json();
        setOverview(data);
      } catch (err) {
        console.error('Failed to fetch analytics:', err);
      } finally {
        setLoading(false);
      }
    };

    fetchData();
    const interval = setInterval(fetchData, 60000);

    return () => clearInterval(interval);
  }, [API_BASE]);

  if (loading) {
    return (
      <div className="min-h-screen bg-background dark font-mono text-xs crt-lines flex items-center justify-center">
        <div className="text-primary text-sm uppercase tracking-widest animate-pulse">
          LOADING_ANALYTICS_ENGINE...
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-background dark font-mono text-xs crt-lines scan-sweep p-6">
      <div className="max-w-7xl mx-auto space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-border pb-4">
          <div>
            <h1 className="text-4xl font-extrabold text-primary glow-text tracking-tighter">
              ANALYTICS_DASHBOARD
            </h1>
            <p className="text-muted-foreground text-[10px] uppercase tracking-widest mt-1">
              // TIME-SERIES ANALYTICS & TREND ANALYSIS
            </p>
          </div>
          <div className="flex gap-2">
            <Link href="/health">
              <Button size="sm" variant="outline" className="rounded-none border-border text-[10px] uppercase tracking-widest">
                HEALTH_MONITOR
              </Button>
            </Link>
            <Link href="/">
              <Button size="sm" variant="outline" className="gap-2 rounded-none border-border text-[10px] uppercase tracking-widest">
                <ChevronLeft className="h-4 w-4" />
                SEARCH
              </Button>
            </Link>
          </div>
        </div>

        {/* Overview Stats */}
        {overview && (
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            <Card className="bg-card border border-border rounded-none">
              <CardContent className="p-4">
                <div className="flex items-center gap-2 mb-2">
                  <Database className="h-4 w-4 text-primary" />
                  <span className="text-[10px] text-muted-foreground uppercase tracking-widest font-bold">Total Scans</span>
                </div>
                <div className="text-2xl font-bold text-foreground">{overview.total_scans.toLocaleString()}</div>
              </CardContent>
            </Card>

            <Card className="bg-card border border-border rounded-none">
              <CardContent className="p-4">
                <div className="flex items-center gap-2 mb-2">
                  <TrendingUp className="h-4 w-4 text-primary" />
                  <span className="text-[10px] text-muted-foreground uppercase tracking-widest font-bold">Unique Hosts</span>
                </div>
                <div className="text-2xl font-bold text-foreground">{overview.unique_hosts.toLocaleString()}</div>
              </CardContent>
            </Card>

            <Card className="bg-card border border-border rounded-none">
              <CardContent className="p-4">
                <div className="flex items-center gap-2 mb-2">
                  <Shield className="h-4 w-4 text-primary" />
                  <span className="text-[10px] text-muted-foreground uppercase tracking-widest font-bold">CVE Detections</span>
                </div>
                <div className="text-2xl font-bold text-foreground">{overview.recent_cves?.length || 0}</div>
              </CardContent>
            </Card>

            <Card className="bg-card border border-border rounded-none">
              <CardContent className="p-4">
                <div className="flex items-center gap-2 mb-2">
                  <Network className="h-4 w-4 text-primary" />
                  <span className="text-[10px] text-muted-foreground uppercase tracking-widest font-bold">Top ASNs</span>
                </div>
                <div className="text-2xl font-bold text-foreground">{overview.top_asns?.length || 0}</div>
              </CardContent>
            </Card>
          </div>
        )}

        {/* Top Services */}
        <Card className="bg-card border border-border rounded-none overflow-hidden">
          <div className="bg-primary/5 px-4 py-2 border-b border-border">
            <span className="text-[10px] text-primary uppercase tracking-widest font-bold">
              // TOP_SERVICES_BY_HOST_COUNT
            </span>
          </div>
          <CardContent className="p-4">
            <div className="space-y-3">
              {overview?.top_services?.map((service, i) => (
                <div key={i} className="flex items-center justify-between p-3 border border-border bg-black/20">
                  <div className="flex items-center gap-4">
                    <span className="text-sm font-bold text-primary">{service.service}</span>
                    <span className="text-[10px] text-muted-foreground font-mono">PORT {service.port}</span>
                  </div>
                  <div className="text-[10px] text-muted-foreground font-mono">
                    {service.host_count.toLocaleString()} hosts | {service.total_scans.toLocaleString()} scans
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* Recent CVE Detections */}
        <Card className="bg-card border border-border rounded-none overflow-hidden">
          <div className="bg-red-950/20 px-4 py-2 border-b border-red-950">
            <span className="text-[10px] text-red-500 uppercase tracking-widest font-bold">
              // RECENT_CVE_DETECTIONS
            </span>
          </div>
          <CardContent className="p-4">
            <div className="space-y-3">
              {overview?.recent_cves?.slice(0, 20).map((cve, i) => (
                <div key={i} className="flex items-center justify-between p-3 border border-border bg-black/20">
                  <a
                    href={`https://nvd.nist.gov/vuln/detail/${cve.cve_id}`}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-sm font-bold text-red-400 hover:text-red-300 transition-colors"
                  >
                    {cve.cve_id}
                  </a>
                  <div className="text-[10px] text-muted-foreground font-mono">
                    {cve.affected_hosts.toLocaleString()} affected hosts
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* Top ASNs by Scan Volume */}
        <Card className="bg-card border border-border rounded-none overflow-hidden">
          <div className="bg-primary/5 px-4 py-2 border-b border-border">
            <span className="text-[10px] text-primary uppercase tracking-widest font-bold">
              // TOP_ASNS_BY_SCAN_VOLUME
            </span>
          </div>
          <CardContent className="p-4">
            <div className="space-y-3">
              {overview?.top_asns?.map((asn, i) => (
                <div key={i} className="flex items-center justify-between p-3 border border-border bg-black/20">
                  <div className="flex items-center gap-4">
                    <span className="text-sm font-bold text-foreground font-mono">AS{asn.asn}</span>
                    <span className="text-[10px] text-muted-foreground">{asn.country}</span>
                  </div>
                  <div className="text-[10px] text-muted-foreground font-mono">
                    {asn.scan_count.toLocaleString()} scans | {asn.unique_ips.toLocaleString()} IPs
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
