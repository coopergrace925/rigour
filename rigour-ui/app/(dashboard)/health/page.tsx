'use client';

import { useEffect, useState } from 'react';
import { Card, CardContent, CardHeader } from '../../../../components/ui/card';
import { Badge } from '../../../../components/ui/badge';
import Link from 'next/link';
import { Button } from '../../../../components/ui/button';
import { ChevronLeft, Activity, Server, Zap, Database, Network } from 'lucide-react';

interface ScanStats {
  total_hosts_scanned: number;
  total_ports_scanned: number;
  active_scanners: number;
  queue_depth: number;
  scans_per_second: number;
  last_scan_time: string;
  system_status: string;
  data_freshness: string;
}

interface PortSchedule {
  port: number;
  priority: string;
  last_scanned: string;
  next_scan: string;
  scan_interval: string;
}

interface ASNRate {
  asn: number;
  current_rate: number;
  limit: number;
  percent_used: number;
}

interface StreamHealth {
  stream_name: string;
  message_count: number;
  consumer_count: number;
  replicas: number;
  status: string;
}

export default function DashboardPage() {
  const [stats, setStats] = useState<ScanStats | null>(null);
  const [schedules, setSchedules] = useState<PortSchedule[]>([]);
  const [asnRates, setASNRates] = useState<ASNRate[]>([]);
  const [streams, setStreams] = useState<StreamHealth[]>([]);
  const [loading, setLoading] = useState(true);

  const API_BASE = process.env.NEXT_PUBLIC_API_BASE_URL || 'http://localhost:8080';

  useEffect(() => {
    const fetchData = async () => {
      try {
        const [statsRes, schedulesRes, asnRatesRes, streamsRes] = await Promise.all([
          fetch(`${API_BASE}/api/dashboard/stats`),
          fetch(`${API_BASE}/api/dashboard/schedules`),
          fetch(`${API_BASE}/api/dashboard/asn-rates`),
          fetch(`${API_BASE}/api/dashboard/streams`),
        ]);

        const statsData = await statsRes.json();
        const schedulesData = await schedulesRes.json();
        const asnRatesData = await asnRatesRes.json();
        const streamsData = await streamsRes.json();

        setStats(statsData);
        setSchedules(schedulesData.schedules || []);
        setASNRates(asnRatesData.asn_rates || []);
        setStreams(streamsData.streams || []);
      } catch (err) {
        console.error('Failed to fetch dashboard data:', err);
      } finally {
        setLoading(false);
      }
    };

    fetchData();
    const interval = setInterval(fetchData, 30000); // Refresh every 30 seconds

    return () => clearInterval(interval);
  }, [API_BASE]);

  if (loading) {
    return (
      <div className="min-h-screen bg-background dark font-mono text-xs crt-lines scan-sweep flex items-center justify-center">
        <div className="text-primary text-sm uppercase tracking-widest animate-pulse">
          LOADING_TELEMETRY_STREAM...
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
              SCAN_HEALTH_DASHBOARD
            </h1>
            <p className="text-muted-foreground text-[10px] uppercase tracking-widest mt-1">
              // REAL-TIME SYSTEM TELEMETRY MONITOR
            </p>
          </div>
          <Link href="/">
            <Button size="sm" variant="outline" className="gap-2 rounded-none border-border hover:border-primary uppercase tracking-widest text-[10px]">
              <ChevronLeft className="h-4 w-4" />
              BACK_TO_SEARCH
            </Button>
          </Link>
        </div>

        {/* System Status Overview */}
        {stats && (
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            <Card className="bg-card border border-border rounded-none">
              <CardContent className="p-4">
                <div className="flex items-center gap-2 mb-2">
                  <Activity className="h-4 w-4 text-primary" />
                  <span className="text-[10px] text-muted-foreground uppercase tracking-widest font-bold">System Status</span>
                </div>
                <div className="text-2xl font-bold text-primary glow-text">{stats.system_status.toUpperCase()}</div>
                <div className="text-[10px] text-muted-foreground mt-1">Data Freshness: {stats.data_freshness}</div>
              </CardContent>
            </Card>

            <Card className="bg-card border border-border rounded-none">
              <CardContent className="p-4">
                <div className="flex items-center gap-2 mb-2">
                  <Zap className="h-4 w-4 text-primary" />
                  <span className="text-[10px] text-muted-foreground uppercase tracking-widest font-bold">Scan Rate</span>
                </div>
                <div className="text-2xl font-bold text-foreground">{stats.scans_per_second.toFixed(1)}</div>
                <div className="text-[10px] text-muted-foreground mt-1">scans per second</div>
              </CardContent>
            </Card>

            <Card className="bg-card border border-border rounded-none">
              <CardContent className="p-4">
                <div className="flex items-center gap-2 mb-2">
                  <Server className="h-4 w-4 text-primary" />
                  <span className="text-[10px] text-muted-foreground uppercase tracking-widest font-bold">Active Scanners</span>
                </div>
                <div className="text-2xl font-bold text-foreground">{stats.active_scanners}</div>
                <div className="text-[10px] text-muted-foreground mt-1">ZMap/ZGrab2 nodes</div>
              </CardContent>
            </Card>

            <Card className="bg-card border border-border rounded-none">
              <CardContent className="p-4">
                <div className="flex items-center gap-2 mb-2">
                  <Database className="h-4 w-4 text-primary" />
                  <span className="text-[10px] text-muted-foreground uppercase tracking-widest font-bold">Queue Depth</span>
                </div>
                <div className="text-2xl font-bold text-foreground">{stats.queue_depth.toLocaleString()}</div>
                <div className="text-[10px] text-muted-foreground mt-1">pending tasks</div>
              </CardContent>
            </Card>
          </div>
        )}

        {/* Port Schedules */}
        <Card className="bg-card border border-border rounded-none overflow-hidden">
          <div className="bg-primary/5 px-4 py-2 border-b border-border">
            <span className="text-[10px] text-primary uppercase tracking-widest font-bold">
              // PORT_PRIORITY_SCHEDULE
            </span>
          </div>
          <CardContent className="p-4">
            <div className="space-y-3">
              {schedules.map((schedule) => (
                <div key={schedule.port} className="flex items-center justify-between p-3 border border-border bg-black/20">
                  <div className="flex items-center gap-4">
                    <span className="text-lg font-bold text-primary bg-primary/10 border border-primary/30 px-3 py-1">
                      PORT {schedule.port}
                    </span>
                    <Badge variant="outline" className="uppercase text-[10px] font-bold border-border">
                      {schedule.priority}
                    </Badge>
                    <span className="text-[10px] text-muted-foreground">
                      INTERVAL: {schedule.scan_interval}
                    </span>
                  </div>
                  <div className="text-[10px] text-muted-foreground">
                    NEXT: {new Date(schedule.next_scan).toLocaleString()}
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* ASN Rate Limiting */}
        <Card className="bg-card border border-border rounded-none overflow-hidden">
          <div className="bg-primary/5 px-4 py-2 border-b border-border">
            <span className="text-[10px] text-primary uppercase tracking-widest font-bold">
              // ASN_RATE_LIMITING_STATUS
            </span>
          </div>
          <CardContent className="p-4">
            <div className="space-y-3">
              {asnRates.map((rate) => (
                <div key={rate.asn} className="flex items-center justify-between p-3 border border-border bg-black/20">
                  <div className="flex items-center gap-4">
                    <span className="text-sm font-bold text-foreground font-mono">AS{rate.asn}</span>
                    <div className="flex-1 max-w-xs">
                      <div className="h-2 bg-secondary border border-border relative overflow-hidden">
                        <div
                          className={`h-full ${
                            rate.percent_used > 80 ? 'bg-red-500' : rate.percent_used > 50 ? 'bg-yellow-500' : 'bg-emerald-500'
                          }`}
                          style={{ width: `${rate.percent_used}%` }}
                        />
                      </div>
                    </div>
                  </div>
                  <div className="text-[10px] text-muted-foreground font-mono">
                    {rate.current_rate}/{rate.limit} ({rate.percent_used.toFixed(1)}%)
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* NATS Stream Health */}
        <Card className="bg-card border border-border rounded-none overflow-hidden">
          <div className="bg-primary/5 px-4 py-2 border-b border-border">
            <span className="text-[10px] text-primary uppercase tracking-widest font-bold">
              // NATS_JETSTREAM_HEALTH
            </span>
          </div>
          <CardContent className="p-4">
            <div className="space-y-3">
              {streams.map((stream) => (
                <div key={stream.stream_name} className="flex items-center justify-between p-3 border border-border bg-black/20">
                  <div className="flex items-center gap-4">
                    <Network className="h-4 w-4 text-primary" />
                    <span className="text-sm font-bold text-foreground font-mono">{stream.stream_name}</span>
                    <Badge
                      variant="outline"
                      className={`text-[10px] font-bold ${
                        stream.status === 'healthy'
                          ? 'border-emerald-900 text-emerald-500 bg-emerald-950/20'
                          : 'border-red-900 text-red-500 bg-red-950/20'
                      }`}
                    >
                      {stream.status.toUpperCase()}
                    </Badge>
                  </div>
                  <div className="flex items-center gap-4 text-[10px] text-muted-foreground font-mono">
                    <span>MSGS: {stream.message_count.toLocaleString()}</span>
                    <span>CONSUMERS: {stream.consumer_count}</span>
                    <span>REPLICAS: {stream.replicas}</span>
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
