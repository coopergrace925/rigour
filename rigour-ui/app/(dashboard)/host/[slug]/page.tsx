import Link from 'next/link';
import { getHostByIP } from '../../../../lib/api';
import { formatDate, formatDateShort } from '../../../../lib/utils';
import { Host } from '../../../../lib/types';
import { Card, CardContent, CardHeader } from '../../../../components/ui/card';
import { Badge } from '../../../../components/ui/badge';
import { Button } from '../../../../components/ui/button';
import {
  Globe,
  Network,
  Server,
  Clock,
  MapPin,
  ChevronLeft,
  ExternalLink,
  AlertTriangle,
  ShieldAlert,
  Wifi,
  Shield,
  FileCode,
  Terminal,
} from 'lucide-react';

interface Params {
  slug: string;
}

export default async function HostDetailsPage({ 
  params 
}: { 
  params: Promise<Params> 
}) {
  const { slug } = await params;
  let host: Host | null = null;
  let error: string | null = null;

  try {
    host = await getHostByIP(slug);
  } catch (err) {
    console.error('Failed to fetch host:', err);
    error = err instanceof Error ? err.message : 'Failed to fetch host details';
  }

  if (error || !host) {
    return (
      <div className="min-h-screen bg-background p-6 dark font-mono text-xs crt-lines scan-sweep">
        <div className="max-w-6xl mx-auto space-y-6">
          <Link href="/">
            <Button size="sm" variant="outline" className="gap-2 rounded-none border-border hover:border-primary">
              <ChevronLeft className="h-4 w-4" />
              BACK_TO_SEARCH
            </Button>
          </Link>
          <div className="text-center py-16 border border-red-900 bg-red-950/20 max-w-md mx-auto">
            <ShieldAlert className="h-12 w-12 text-destructive mx-auto mb-4 animate-pulse" />
            <p className="text-destructive font-bold uppercase tracking-wider text-sm">
              [ERROR: HOST_NOT_FOUND]
            </p>
            <p className="text-muted-foreground mt-2 text-[10px]">
              {error || 'IP address is not in scan records.'}
            </p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-background p-6 dark font-mono text-xs crt-lines scan-sweep">
      <div className="max-w-6xl mx-auto space-y-6">
        
        {/* Navigation Bar */}
        <div className="flex items-center justify-between border-b border-border pb-4">
          <Link href="/">
            <Button size="sm" variant="outline" className="gap-2 rounded-none border-border hover:border-primary text-foreground uppercase tracking-widest text-[10px] h-9">
              <ChevronLeft className="h-4 w-4" />
              BACK_TO_SEARCH
            </Button>
          </Link>
          <div className="text-[10px] text-muted-foreground uppercase">
            HOST_ID: <code className="bg-secondary px-2 py-1 border border-border text-foreground font-mono text-xs">{host.id}</code>
          </div>
        </div>

        {/* Diagnostic Printout Summary */}
        <Card className="bg-card border border-border rounded-none overflow-hidden">
          <div className="bg-primary/5 px-4 py-2 border-b border-border flex items-center justify-between">
            <span className="text-[10px] text-primary uppercase tracking-widest font-bold">
              // HOST_DIAGNOSTICS_SUMMARY
            </span>
            <span className="text-[10px] text-muted-foreground flex items-center gap-1.5">
              <span className="h-2 w-2 rounded-full bg-emerald-500 animate-pulse"></span>
              TELEMETRY_STREAM: LIVE
            </span>
          </div>
          <CardContent className="p-6 space-y-6">
            <div className="flex flex-col md:flex-row md:items-start justify-between gap-6">
              <div className="space-y-2">
                <h1 className="text-4xl md:text-5xl font-extrabold tracking-tighter text-primary glow-text font-mono">
                  {host.ip}
                </h1>
                {host.rdns ? (
                  <p className="text-sm text-muted-foreground border-l-2 border-primary/50 pl-3 italic">
                    Resolved Hostname (rDNS): <span className="text-foreground font-bold">{host.rdns}</span>
                  </p>
                ) : (
                  <p className="text-xs text-muted-foreground italic pl-3 border-l-2 border-border">
                    No reverse DNS hostname found for this address.
                  </p>
                )}
              </div>

              {host.cves && host.cves.length > 0 && (
                <div className="flex items-center gap-3 bg-red-950/40 border border-red-900 px-4 py-3 text-red-500 rounded-none max-w-sm animate-pulse">
                  <AlertTriangle className="h-6 w-6 flex-shrink-0" />
                  <div>
                    <div className="font-bold tracking-widest text-[10px] uppercase">SECURITY WARNING</div>
                    <div className="text-[11px] text-red-400 mt-0.5">{host.cves.length} KNOWN VULNERABILITIES DETECTED ON PORT HANDSHAKES</div>
                  </div>
                </div>
              )}
            </div>

            {/* Core Info Grid */}
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4 pt-6 border-t border-border/60">
              <div className="space-y-1">
                <div className="flex items-center gap-1.5 text-[10px] text-muted-foreground uppercase tracking-widest font-bold">
                  <Globe className="h-3.5 w-3.5 text-primary/70" />
                  Country
                </div>
                <p className="text-sm text-foreground font-bold">{host.asn.country} ({host.location.country_code})</p>
              </div>

              <div className="space-y-1">
                <div className="flex items-center gap-1.5 text-[10px] text-muted-foreground uppercase tracking-widest font-bold">
                  <MapPin className="h-3.5 w-3.5 text-primary/70" />
                  City
                </div>
                <p className="text-sm text-foreground font-bold">{host.location.city || 'Unknown'}</p>
              </div>

              <div className="space-y-1">
                <div className="flex items-center gap-1.5 text-[10px] text-muted-foreground uppercase tracking-widest font-bold">
                  <Network className="h-3.5 w-3.5 text-primary/70" />
                  Network ASN
                </div>
                <p className="text-sm text-foreground font-bold font-mono">AS{host.asn.number}</p>
              </div>

              <div className="space-y-1">
                <div className="flex items-center gap-1.5 text-[10px] text-muted-foreground uppercase tracking-widest font-bold">
                  <Clock className="h-3.5 w-3.5 text-primary/70" />
                  Timezone
                </div>
                <p className="text-sm text-foreground font-bold">{host.location.timezone || 'Unknown'}</p>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Security / Vulnerabilities section */}
        {host.cves && host.cves.length > 0 && (
          <Card className="bg-card border border-red-950 rounded-none overflow-hidden">
            <div className="bg-red-950/20 px-4 py-2.5 border-b border-red-950 flex items-center gap-2">
              <Shield className="h-4 w-4 text-red-500" />
              <span className="text-[10px] text-red-500 uppercase tracking-widest font-bold">
                // ACTIVE_VULNERABILITY_INDEX ({host.cves.length})
              </span>
            </div>
            <CardContent className="p-4 space-y-3">
              <div className="text-muted-foreground text-[11px]">
                The following CVE matches have been detected based on banner CPE mappings:
              </div>
              <div className="flex flex-wrap gap-2 pt-1">
                {host.cves.map((cve) => (
                  <a
                    key={cve}
                    href={`https://nvd.nist.gov/vuln/detail/${cve}`}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="flex items-center gap-1.5 px-3 py-1.5 bg-red-950/30 hover:bg-red-950/50 border border-red-900 text-red-400 font-bold transition-colors cursor-pointer group"
                  >
                    <span>{cve}</span>
                    <ExternalLink className="h-3.5 w-3.5 text-red-500 group-hover:translate-x-0.5 transition-transform" />
                  </a>
                ))}
              </div>
            </CardContent>
          </Card>
        )}

        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          {/* Timeline Information */}
          <Card className="bg-card border border-border rounded-none overflow-hidden">
            <div className="bg-primary/5 px-4 py-2 border-b border-border">
              <span className="text-[10px] text-primary uppercase tracking-widest font-bold">// TIMELINE_LOG</span>
            </div>
            <CardContent className="p-4 space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-1">
                  <div className="text-[10px] text-muted-foreground uppercase tracking-widest font-bold">First Seen</div>
                  <p className="text-sm font-bold">{formatDate(host.first_seen)}</p>
                  <p className="text-[10px] text-muted-foreground">{formatDateShort(host.first_seen)}</p>
                </div>
                <div className="space-y-1">
                  <div className="text-[10px] text-muted-foreground uppercase tracking-widest font-bold">Last Seen</div>
                  <p className="text-sm font-bold text-primary glow-text">{formatDate(host.last_seen)}</p>
                  <p className="text-[10px] text-muted-foreground">{formatDateShort(host.last_seen)}</p>
                </div>
              </div>
            </CardContent>
          </Card>

          {/* Network details */}
          <Card className="bg-card border border-border rounded-none overflow-hidden">
            <div className="bg-primary/5 px-4 py-2 border-b border-border">
              <span className="text-[10px] text-primary uppercase tracking-widest font-bold">// NETWORK_ISP_IDENT</span>
            </div>
            <CardContent className="p-4 space-y-3">
              <div className="space-y-1">
                <p className="text-sm">
                  <span className="text-muted-foreground">ISP:</span>{' '}
                  <span className="font-bold text-foreground">{host.asn.organization}</span>
                </p>
                <p className="text-sm">
                  <span className="text-muted-foreground">Geo-coordinates:</span>{' '}
                  <span className="font-bold font-mono">
                    {host.location.coordinates[1].toFixed(5)}, {host.location.coordinates[0].toFixed(5)}
                  </span>
                </p>
                <div className="pt-2">
                  <a
                    href={`https://maps.google.com/?q=${host.location.coordinates[1]},${host.location.coordinates[0]}`}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="inline-flex items-center gap-1 bg-secondary border border-border px-3 py-1.5 hover:border-primary text-foreground text-[10px] uppercase font-bold tracking-widest transition-colors"
                  >
                    View on Google Maps
                    <ExternalLink className="h-3 w-3 text-primary" />
                  </a>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Services List */}
        <Card className="bg-card border border-border rounded-none overflow-hidden">
          <div className="bg-primary/5 px-4 py-2 border-b border-border">
            <span className="text-[10px] text-primary uppercase tracking-widest font-bold">
              // DISCOVERED_PORTS_LISTING ({host.services.length})
            </span>
          </div>
          <CardContent className="p-6 space-y-6">
            {host.services.length === 0 ? (
              <p className="text-muted-foreground italic text-center py-6 font-mono uppercase text-[10px]">// NO_ACTIVE_SERVICE_PORTS</p>
            ) : (
              <div className="space-y-6">
                {host.services.map((service, idx) => (
                  <div
                    key={idx}
                    className="border border-border p-4 bg-black/20 space-y-4 rounded-none"
                  >
                    {/* Header: Port/Proto & TLS */}
                    <div className="flex items-center justify-between gap-4 flex-wrap border-b border-border/40 pb-3">
                      <div className="flex items-center gap-3">
                        <span className="font-mono text-lg font-bold bg-primary/10 border border-primary/30 px-3 py-1 text-primary glow-text">
                          PORT {service.port}
                        </span>
                        <span className="uppercase text-xs font-bold border border-border px-2 py-1">
                          {service.protocol}
                        </span>
                        <span className="text-[10px] text-muted-foreground uppercase">{service.transport}</span>
                        {service.tls && (
                          <span className="text-[10px] bg-emerald-950/20 border border-emerald-900 text-emerald-500 font-bold px-2 py-0.5 flex items-center gap-1.5">
                            <Shield className="h-3 w-3" />
                            TLS_HANDSHAKE
                          </span>
                        )}
                      </div>
                      <div className="text-[10px] text-muted-foreground uppercase flex items-center gap-1">
                        <Wifi className="h-3 w-3" />
                        SCANNED: {formatDateShort(service.last_scan)}
                      </div>
                    </div>

                    {/* CPE telemetry if available */}
                    {service.cpe && (
                      <div className="flex flex-col sm:flex-row sm:items-center gap-2 bg-secondary/40 border border-border/60 p-2.5">
                        <span className="text-[10px] font-bold text-primary uppercase tracking-wider">// DETECTED_CPE:</span>
                        <code className="text-xs font-mono text-foreground font-bold select-all">{service.cpe}</code>
                        {service.product && (
                          <span className="text-[10px] text-muted-foreground">
                            (Identified product: <strong className="text-foreground">{service.product}</strong>)
                          </span>
                        )}
                      </div>
                    )}

                    {/* Service banner or TLS handshake dumps */}
                    {(service.https || service.http || service.ssh || service.banner) && (
                      <div className="space-y-3 font-mono">
                        {service.https && (
                          <div className="space-y-1 bg-black/40 border border-border p-3">
                            <div className="flex items-center gap-1.5 text-[10px] text-emerald-500 font-bold uppercase mb-2">
                              <Terminal className="h-3.5 w-3.5" />
                              HTTPS_RESPONSE_HEADER
                            </div>
                            <div className="space-y-1 text-xs text-muted-foreground pl-1">
                              <p>HTTP/1.1 <span className="text-foreground font-bold">{service.https.statusCode}</span> {service.https.status}</p>
                              {Object.entries(service.https.responseHeaders).map(([key, values]) => (
                                <p key={key}>
                                  <span className="text-primary font-medium">{key}:</span> {values.join(', ')}
                                </p>
                              ))}
                            </div>
                          </div>
                        )}

                        {service.http && (
                          <div className="space-y-1 bg-black/40 border border-border p-3">
                            <div className="flex items-center gap-1.5 text-[10px] text-emerald-500 font-bold uppercase mb-2">
                              <Terminal className="h-3.5 w-3.5" />
                              HTTP_RESPONSE_HEADER
                            </div>
                            <div className="space-y-1 text-xs text-muted-foreground pl-1">
                              <p>HTTP/1.1 <span className="text-foreground font-bold">{service.http.statusCode}</span> {service.http.status}</p>
                              {Object.entries(service.http.responseHeaders).map(([key, values]) => (
                                <p key={key}>
                                  <span className="text-primary font-medium">{key}:</span> {values.join(', ')}
                                </p>
                              ))}
                            </div>
                          </div>
                        )}

                        {service.ssh && (
                          <div className="space-y-1 bg-black/40 border border-border p-3">
                            <div className="flex items-center gap-1.5 text-[10px] text-primary font-bold uppercase mb-2">
                              <Terminal className="h-3.5 w-3.5" />
                              SSH_HANDSHAKE_TELEMETRY
                            </div>
                            <pre className="text-xs text-muted-foreground overflow-auto p-1 font-mono break-all leading-relaxed bg-black/20">
                              {service.ssh.banner}
                            </pre>
                          </div>
                        )}

                        {!service.https && !service.http && !service.ssh && service.banner && (
                          <div className="space-y-1 bg-black/40 border border-border p-3">
                            <div className="flex items-center gap-1.5 text-[10px] text-primary font-bold uppercase mb-2">
                              <Terminal className="h-3.5 w-3.5" />
                              RAW_BANNER_PAYLOAD
                            </div>
                            <pre className="text-xs text-muted-foreground overflow-auto p-1 font-mono break-all leading-relaxed bg-black/20">
                              {service.banner}
                            </pre>
                          </div>
                        )}
                      </div>
                    )}
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        {/* Diagnostic Memory Dump (Raw JSON) */}
        <Card className="bg-card border border-border rounded-none overflow-hidden">
          <div className="bg-primary/5 px-4 py-2 border-b border-border">
            <span className="text-[10px] text-primary uppercase tracking-widest font-bold">
              // DIAGNOSTIC_MEMORY_DUMP (RAW_JSON)
            </span>
          </div>
          <CardContent className="p-4">
            <div className="bg-black/60 p-4 border border-border overflow-auto max-h-96">
              <pre className="text-[11px] font-mono text-muted-foreground leading-normal">
                {JSON.stringify(host, null, 2)}
              </pre>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
