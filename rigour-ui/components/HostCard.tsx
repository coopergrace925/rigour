import { Host } from '../lib/types';
import { Card, CardContent } from './ui/card';
import { Badge } from './ui/badge';
import { Globe, Network, Server, Clock, MapPin, AlertTriangle, ShieldCheck } from 'lucide-react';

interface HostCardProps {
  host: Host;
  onClick?: () => void;
}

export function HostCard({ host, onClick }: HostCardProps) {
  const formatDate = (dateString: string) => {
    return new Date(dateString).toISOString().split('T')[0];
  };

  return (
    <Card
      className="bg-card border border-border hover:border-primary/60 transition-all duration-200 cursor-pointer overflow-hidden font-mono text-xs rounded-none crt-lines"
      onClick={onClick}
    >
      <div className="bg-primary/5 px-4 py-2 border-b border-border flex items-center justify-between">
        <span className="text-[10px] text-muted-foreground uppercase tracking-widest font-bold">
          [TELEMETRY_RECORD_0x{host.ip.split('.').map(x => parseInt(x).toString(16).padStart(2, '0')).join('').toUpperCase()}]
        </span>
        <span className="text-[10px] text-muted-foreground">
          LAST_SEEN: {formatDate(host.last_seen)}
        </span>
      </div>
      <CardContent className="p-4 space-y-4">
        <div className="flex flex-col md:flex-row md:items-start justify-between gap-4">
          <div className="space-y-1.5 min-w-0">
            <div className="flex items-baseline gap-3">
              <h3 className="text-2xl font-bold tracking-tight text-primary glow-text font-mono">
                {host.ip}
              </h3>
              {host.rdns && (
                <span className="text-muted-foreground text-xs truncate max-w-[200px] md:max-w-[300px]">
                  ({host.rdns})
                </span>
              )}
            </div>
            
            <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-muted-foreground text-[11px]">
              <div className="flex items-center gap-1">
                <Globe className="h-3 w-3 text-primary/70" />
                <span>{host.asn.country} ({host.location.country_code})</span>
              </div>
              <div className="flex items-center gap-1">
                <MapPin className="h-3 w-3 text-primary/70" />
                <span>{host.location.city || 'Unknown Location'}</span>
              </div>
              <div className="flex items-center gap-1">
                <Network className="h-3 w-3 text-primary/70" />
                <span>AS{host.asn.number} - {host.asn.organization}</span>
              </div>
            </div>
          </div>

          {/* Vulnerability Count (CVE Badge) */}
          {host.cves && host.cves.length > 0 ? (
            <div className="flex-shrink-0 flex items-center gap-1.5 bg-red-950/40 border border-red-900 px-2.5 py-1 text-red-500 rounded-none animate-pulse">
              <AlertTriangle className="h-3.5 w-3.5" />
              <span className="font-bold tracking-wider uppercase text-[10px]">
                {host.cves.length} VULNERABILITIES DETECTED
              </span>
            </div>
          ) : (
            <div className="flex-shrink-0 flex items-center gap-1.5 bg-emerald-950/20 border border-emerald-900/40 px-2.5 py-1 text-emerald-500 rounded-none">
              <ShieldCheck className="h-3.5 w-3.5" />
              <span className="font-bold tracking-wider uppercase text-[10px]">
                NO KNOWN THREATS
              </span>
            </div>
          )}
        </div>

        {/* Services discovered */}
        <div className="space-y-2 pt-2 border-t border-border/40">
          <div className="flex items-center gap-2">
            <Server className="h-3 w-3 text-muted-foreground" />
            <span className="text-[10px] text-muted-foreground uppercase tracking-widest font-bold">DISCOVERED_SERVICES</span>
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
            {host.services.map((service, idx) => (
              <div
                key={idx}
                className="flex items-center gap-2 px-2.5 py-1.5 border border-border/60 bg-black/20"
              >
                <span className="text-primary font-bold text-[11px] bg-primary/10 px-1.5 py-0.5 border border-primary/20">
                  {service.port}
                </span>
                <span className="uppercase text-[10px] font-bold text-muted-foreground border border-border/80 px-1 py-0.5">
                  {service.protocol}
                </span>
                <span className="text-[10px] uppercase text-muted-foreground">{service.transport}</span>
                {service.tls && (
                  <span className="text-[9px] bg-emerald-950/30 text-emerald-500 border border-emerald-900/60 px-1 py-0.5 uppercase font-bold">
                    TLS
                  </span>
                )}
                {service.cpe && (
                  <span className="text-[9px] bg-secondary border border-border text-foreground px-1 truncate max-w-[120px]" title={service.cpe}>
                    {service.cpe.replace('cpe:/a:', '')}
                  </span>
                )}
              </div>
            ))}
          </div>
        </div>

        {/* Banners & Details preview */}
        {host.services.some(s => s.https || s.http || s.ssh || s.banner) && (
          <div className="pt-2 border-t border-border/40 space-y-1.5">
            <div className="text-[10px] text-muted-foreground uppercase tracking-widest font-bold">TELEMETRY_LOG</div>
            <div className="space-y-1 font-mono text-[11px] bg-black/40 border border-border/60 p-2 text-muted-foreground overflow-hidden max-h-[60px] truncate">
              {host.services.map((service, idx) => {
                if (service.https) {
                  return (
                    <div key={idx} className="truncate">
                      PORT {service.port}: HTTPS/1.1 {service.https.statusCode} {service.https.status}
                    </div>
                  );
                }
                if (service.http) {
                  return (
                    <div key={idx} className="truncate">
                      PORT {service.port}: HTTP/1.1 {service.http.statusCode} {service.http.status}
                    </div>
                  );
                }
                if (service.ssh) {
                  return (
                    <div key={idx} className="truncate">
                      PORT {service.port}: SSH-2.0 banner =&gt; {service.ssh.banner}
                    </div>
                  );
                }
                if (service.banner) {
                  return (
                    <div key={idx} className="truncate">
                      PORT {service.port}: {service.banner}
                    </div>
                  );
                }
                return null;
              })}
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
