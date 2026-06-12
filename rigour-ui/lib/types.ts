export interface Service {
  ip: string;
  port: number;
  protocol: string;
  tls: boolean;
  transport: string;
  last_scan: string;
  cpe?: string;
  product?: string;
  banner?: string;
  https?: {
    status: string;
    statusCode: number;
    responseHeaders: Record<string, string[]>;
  };
  http?: {
    status: string;
    statusCode: number;
    responseHeaders: Record<string, string[]>;
  };
  ssh?: {
    banner: string;
  };
}

export interface Host {
  id: string;
  ip: string;
  ip_int: number;
  rdns?: string;
  cves?: string[];
  asn: {
    number: number;
    organization: string;
    country: string;
  };
  location: {
    coordinates: [number, number];
    city: string;
    timezone: string;
    country_code: string;
    country_name: string;
  };
  first_seen: string;
  last_seen: string;
  services: Service[];
}
