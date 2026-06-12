'use client';

import { useState, useTransition } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from './ui/card';
import { Checkbox } from './ui/checkbox';
import { Label } from './ui/label';
import { ScrollArea } from './ui/scroll-area';
import { Button } from './ui/button';
import { ChevronDown, ChevronRight, Loader2 } from 'lucide-react';
import { useRouter } from 'next/navigation';
import { FacetCounts } from '../lib/api';

interface FacetFiltersProps {
  facets: FacetCounts;
  selectedCountries: string[];
  selectedASNs: string[];
  selectedServices: string[];
}

export function FacetFilters({
  facets,
  selectedCountries: initialCountries,
  selectedASNs: initialASNs,
  selectedServices: initialServices,
}: FacetFiltersProps) {
  const [expandedSections, setExpandedSections] = useState({
    countries: true,
    asns: true,
    services: true,
  });
  const [tempCountries, setTempCountries] = useState(initialCountries);
  const [tempASNs, setTempASNs] = useState(initialASNs);
  const [tempServices, setTempServices] = useState(initialServices);

  const [isPending, startTransition] = useTransition();
  const router = useRouter();

  const toggleSection = (section: keyof typeof expandedSections) => {
    setExpandedSections(prev => ({
      ...prev,
      [section]: !prev[section],
    }));
  };

  const applyFilters = () => {
    startTransition(() => {
      const params = new URLSearchParams();

      if (tempCountries.length > 0) {
        params.append('countries', tempCountries.join(','));
      }

      if (tempASNs.length > 0) {
        params.append('asns', tempASNs.join(','));
      }

      if (tempServices.length > 0) {
        params.append('services', tempServices.join(','));
      }

      router.push(`?${params.toString()}`);
    });
  };

  const handleCountryToggle = (country: string) => {
    setTempCountries(prev =>
      prev.includes(country)
        ? prev.filter(c => c !== country)
        : [...prev, country]
    );
  };

  const handleASNToggle = (asn: string) => {
    setTempASNs(prev =>
      prev.includes(asn)
        ? prev.filter(a => a !== asn)
        : [...prev, asn]
    );
  };

  const handleServiceToggle = (service: string) => {
    setTempServices(prev =>
      prev.includes(service)
        ? prev.filter(s => s !== service)
        : [...prev, service]
    );
  };

  const hasChanges =
    JSON.stringify(tempCountries) !== JSON.stringify(initialCountries) ||
    JSON.stringify(tempASNs) !== JSON.stringify(initialASNs) ||
    JSON.stringify(tempServices) !== JSON.stringify(initialServices);

  return (
    <div className="space-y-4 relative font-mono text-xs">
      {isPending && (
        <div className="absolute inset-0 bg-black/60 rounded-none flex items-center justify-center z-10 pointer-events-none border border-border">
          <div className="flex flex-col items-center gap-2">
            <Loader2 className="h-5 w-5 animate-spin text-primary" />
            <span className="text-[10px] text-primary uppercase tracking-wider font-bold">REFILTERING_STREAM...</span>
          </div>
        </div>
      )}

      {/* FILTER CONTROL MODULE HEADER */}
      <div className="border border-border bg-primary/5 px-4 py-3 flex items-center justify-between">
        <span className="text-[10px] font-bold tracking-widest text-primary uppercase">// FILTER_CONTROLS</span>
      </div>

      <Card className="bg-card border border-border rounded-none crt-lines">
        <CardHeader className="pb-2 pt-3 px-4">
          <CardTitle
            className="flex items-center justify-between cursor-pointer uppercase tracking-widest text-[10px] font-bold text-muted-foreground hover:text-primary transition-colors"
            onClick={() => toggleSection('countries')}
          >
            <span>LOCATION_COUNTRY</span>
            {expandedSections.countries ? (
              <ChevronDown className="h-3 w-3" />
            ) : (
              <ChevronRight className="h-3 w-3" />
            )}
          </CardTitle>
        </CardHeader>
        {expandedSections.countries && (
          <CardContent className="p-0">
            <ScrollArea className="h-44 border-t border-border/40">
              <div className="space-y-2.5 px-4 py-3">
                {facets.countries.map((country) => (
                  <div key={country.code} className="flex items-center justify-between gap-2">
                    <div className="flex items-center gap-2 min-w-0 flex-1">
                      <Checkbox
                        id={`country-${country.code}`}
                        checked={tempCountries.includes(country.code)}
                        onCheckedChange={() => handleCountryToggle(country.code)}
                        className="h-3.5 w-3.5 border-border rounded-none data-[state=checked]:bg-primary data-[state=checked]:text-black"
                      />
                      <Label
                        htmlFor={`country-${country.code}`}
                        className="cursor-pointer text-[11px] font-mono font-normal truncate text-foreground hover:text-primary transition-colors"
                        title={country.name}
                      >
                        {country.name}
                      </Label>
                    </div>
                    <span className="text-[10px] text-muted-foreground font-mono">
                      {country.count}
                    </span>
                  </div>
                ))}
              </div>
            </ScrollArea>
          </CardContent>
        )}
      </Card>

      <Card className="bg-card border border-border rounded-none crt-lines">
        <CardHeader className="pb-2 pt-3 px-4">
          <CardTitle
            className="flex items-center justify-between cursor-pointer uppercase tracking-widest text-[10px] font-bold text-muted-foreground hover:text-primary transition-colors"
            onClick={() => toggleSection('asns')}
          >
            <span>ROUTING_ASN</span>
            {expandedSections.asns ? (
              <ChevronDown className="h-3 w-3" />
            ) : (
              <ChevronRight className="h-3 w-3" />
            )}
          </CardTitle>
        </CardHeader>
        {expandedSections.asns && (
          <CardContent className="p-0">
            <ScrollArea className="h-44 border-t border-border/40">
              <div className="space-y-2.5 px-4 py-3">
                {facets.asns.map((asn) => {
                  const asnString = `AS${asn.code}`;
                  return (
                    <div key={asn.code} className="flex items-center justify-between gap-2">
                      <div className="flex items-center gap-2 min-w-0 flex-1">
                        <Checkbox
                          id={`asn-${asnString}`}
                          checked={tempASNs.includes(asnString)}
                          onCheckedChange={() => handleASNToggle(asnString)}
                          className="h-3.5 w-3.5 border-border rounded-none data-[state=checked]:bg-primary data-[state=checked]:text-black"
                        />
                        <Label
                          htmlFor={`asn-${asnString}`}
                          className="cursor-pointer text-[11px] font-mono font-normal truncate text-foreground hover:text-primary transition-colors"
                          title={asn.name}
                        >
                          AS{asn.code} - {asn.name}
                        </Label>
                      </div>
                      <span className="text-[10px] text-muted-foreground font-mono">
                        {asn.count}
                      </span>
                    </div>
                  );
                })}
              </div>
            </ScrollArea>
          </CardContent>
        )}
      </Card>

      <Card className="bg-card border border-border rounded-none crt-lines">
        <CardHeader className="pb-2 pt-3 px-4">
          <CardTitle
            className="flex items-center justify-between cursor-pointer uppercase tracking-widest text-[10px] font-bold text-muted-foreground hover:text-primary transition-colors"
            onClick={() => toggleSection('services')}
          >
            <span>SERVICE_PROTOCOL</span>
            {expandedSections.services ? (
              <ChevronDown className="h-3 w-3" />
            ) : (
              <ChevronRight className="h-3 w-3" />
            )}
          </CardTitle>
        </CardHeader>
        {expandedSections.services && (
          <CardContent className="p-0">
            <ScrollArea className="h-44 border-t border-border/40">
              <div className="space-y-2.5 px-4 py-3">
                {Object.entries(facets.services || {}).map(([service, count]) => (
                  <div key={service} className="flex items-center justify-between gap-2">
                    <div className="flex items-center gap-2 min-w-0 flex-1">
                      <Checkbox
                        id={`service-${service}`}
                        checked={tempServices.includes(service)}
                        onCheckedChange={() => handleServiceToggle(service)}
                        className="h-3.5 w-3.5 border-border rounded-none data-[state=checked]:bg-primary data-[state=checked]:text-black"
                      />
                      <Label
                        htmlFor={`service-${service}`}
                        className="cursor-pointer text-[11px] font-mono font-normal uppercase text-foreground hover:text-primary transition-colors"
                        title={service}
                      >
                        {service}
                      </Label>
                    </div>
                    <span className="text-[10px] text-muted-foreground font-mono">
                      {count}
                    </span>
                  </div>
                ))}
              </div>
            </ScrollArea>
          </CardContent>
        )}
      </Card>

      {hasChanges && (
        <Button
          onClick={applyFilters}
          disabled={isPending}
          className="w-full bg-primary hover:bg-primary/80 font-mono text-black font-bold uppercase tracking-wider rounded-none py-4"
        >
          {isPending ? 'APPLYING...' : 'APPLY_FILTERS'}
        </Button>
      )}
    </div>
  );
}
