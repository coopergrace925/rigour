import { SearchHeader } from '../../components/SearchHeader';
import { FacetFilters } from '../../components/FacetFilters';
import { HostResults } from '../../components/HostResults';
import WorldMap from '../../components/ui/world-map';
import { searchHosts, getFacets, FacetCounts, API_BASE_URL } from '../../lib/api';
import { Host } from '../../lib/types';

interface PageProps {
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}

export default async function Home({ searchParams: searchParamsPromise }: PageProps) {
  const searchParams = await searchParamsPromise;

  let hosts: Host[] = [];
  let facets: FacetCounts | undefined = undefined;
  let error: string | null = null;

  try {
    const filter: Record<string, unknown> = {};

    const selectedCountries = searchParams.countries
      ? Array.isArray(searchParams.countries)
        ? searchParams.countries
        : searchParams.countries.split(',')
      : [];
    const selectedASNs = searchParams.asns
      ? Array.isArray(searchParams.asns)
        ? searchParams.asns
        : searchParams.asns.split(',')
      : [];
    const selectedServices = searchParams.services
      ? Array.isArray(searchParams.services)
        ? searchParams.services
        : searchParams.services.split(',')
      : [];

    if (selectedCountries.length > 0) {
      filter['location.country_code'] = { $in: selectedCountries };
    }

    if (selectedASNs.length > 0) {
      const asnNumbers = selectedASNs.map(asn => parseInt(asn.replace('AS', '')));
      filter['asn.number'] = { $in: asnNumbers };
    }

    if (selectedServices.length > 0) {
      filter['services.protocol'] = { $in: selectedServices };
    }

    if (searchParams.filter) {
      const filterParams = Array.isArray(searchParams.filter)
        ? searchParams.filter
        : [searchParams.filter];

      for (const filterParam of filterParams) {
        try {
          const parsedFilter = JSON.parse(filterParam);
          Object.assign(filter, parsedFilter);
        } catch (e) {
          console.error('Failed to parse filter parameter:', e);
        }
      }
    }

    const searchResult = await searchHosts(filter, 50);
    hosts = searchResult.hosts || [];

    const facetsResult = await getFacets(filter);
    facets = Object.keys(facetsResult.facets).length === 0
      ? { services: {}, countries: [], asns: [] }
      : facetsResult.facets;
  } catch (err) {
    console.error('Failed to fetch data:', err);
    error = err instanceof Error ? err.message : 'Failed to fetch data';
  }

  const mapDots = hosts
    .filter(host => host.location && host.location.city !== 'Unknown' && host.location.coordinates)
    .map(host => ({
      start: {
        lat: host.location.coordinates[1],
        lng: host.location.coordinates[0],
        label: host.ip,
      },
      end: {
        lat: host.location.coordinates[1],
        lng: host.location.coordinates[0],
        label: host.ip,
      },
    }));

  if (error) {
    return (
      <div className="dark min-h-screen bg-background font-mono text-xs crt-lines scan-sweep flex items-center justify-center p-6">
        <div className="max-w-md w-full border border-red-900 bg-red-950/20 p-6 text-center">
          <div className="text-red-500 font-bold uppercase tracking-wider text-sm mb-2">
            [CONNECTION_ERROR_DETECTED]
          </div>
          <p className="text-muted-foreground mb-4">
            Could not fetch scanner telemetry from node API repository.
          </p>
          <div className="text-left font-mono text-[10px] bg-black/40 border border-border/60 p-3 text-muted-foreground break-all mb-4">
            ERR_CONN_REFUSED: {error}
          </div>
          <p className="text-[10px] text-muted-foreground">
            Please verify API server is active on {API_BASE_URL}
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="dark min-h-screen bg-background font-mono text-xs crt-lines scan-sweep">
      <div className="container mx-auto px-4 py-8 max-w-7xl space-y-8">
        <SearchHeader
          initialQuery={
            typeof searchParams.query === 'string' ? searchParams.query : ''
          }
        />

        <div className="grid grid-cols-1 lg:grid-cols-5 gap-6">
          <aside className="lg:col-span-1">
            {facets && (
              <FacetFilters
                facets={facets}
                selectedCountries={
                  Array.isArray(searchParams.countries)
                    ? searchParams.countries
                    : searchParams.countries?.split(',') || []
                }
                selectedASNs={
                  Array.isArray(searchParams.asns)
                    ? searchParams.asns
                    : searchParams.asns?.split(',') || []
                }
                selectedServices={
                  Array.isArray(searchParams.services)
                    ? searchParams.services
                    : searchParams.services?.split(',') || []
                }
              />
            )}
          </aside>

          <main className="lg:col-span-4 space-y-6">
            {hosts.length > 0 && (
              <div className="border border-border p-2 bg-black/35 rounded-none">
                <div className="px-4 py-2 border-b border-border/40 text-[10px] text-muted-foreground uppercase tracking-widest font-bold">
                  // RADIR_COORDS_SCAN_SWEEP
                </div>
                <div className="p-2">
                  <WorldMap dots={mapDots} lineColor="#f97316" />
                </div>
              </div>
            )}
            <HostResults hosts={hosts} totalCount={hosts.length} />
          </main>
        </div>
      </div>
    </div>
  );
}
