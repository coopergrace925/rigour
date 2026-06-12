'use client';

import { useState } from 'react';
import { Host } from '../lib/types';
import { HostCard } from './HostCard';
import { Button } from './ui/button';
import { ChevronLeft, ChevronRight } from 'lucide-react';
import { useRouter } from 'next/navigation';

interface HostResultsProps {
  hosts: Host[];
  totalCount: number;
  isLoading?: boolean;
}

const HOSTS_PER_PAGE = 10;

export function HostResults({ hosts, totalCount, isLoading }: HostResultsProps) {
  const [currentPage, setCurrentPage] = useState(1);
  const router = useRouter();

  const totalPages = Math.ceil(hosts.length / HOSTS_PER_PAGE);
  const startIndex = (currentPage - 1) * HOSTS_PER_PAGE;
  const endIndex = startIndex + HOSTS_PER_PAGE;
  const paginatedHosts = hosts.slice(startIndex, endIndex);

  const handlePageChange = (page: number) => {
    setCurrentPage(page);
    window.scrollTo({ top: 0, behavior: 'smooth' });
  };

  if (currentPage > totalPages && totalPages > 0) {
    setCurrentPage(1);
  }

  const getPageNumbers = () => {
    const pages: (number | string)[] = [];
    const showPages = 5;

    if (totalPages <= showPages) {
      for (let i = 1; i <= totalPages; i++) {
        pages.push(i);
      }
    } else {
      if (currentPage <= 3) {
        for (let i = 1; i <= 4; i++) {
          pages.push(i);
        }
        pages.push('...');
        pages.push(totalPages);
      } else if (currentPage >= totalPages - 2) {
        pages.push(1);
        pages.push('...');
        for (let i = totalPages - 3; i <= totalPages; i++) {
          pages.push(i);
        }
      } else {
        pages.push(1);
        pages.push('...');
        pages.push(currentPage - 1);
        pages.push(currentPage);
        pages.push(currentPage + 1);
        pages.push('...');
        pages.push(totalPages);
      }
    }

    return pages;
  };

  const handleHostClick = (ip: string) => {
    router.push(`/host/${ip}`);
  };

  return (
    <div className="space-y-6 font-mono text-xs">
      <div className="flex items-center justify-between border border-border bg-black/40 px-4 py-3">
        <div className="text-muted-foreground text-[10px] uppercase tracking-widest font-bold">
          INDEX_STREAM: SHOWING <span className="text-primary font-bold">{startIndex + 1}-{Math.min(endIndex, hosts.length)}</span> OF{' '}
          <span className="text-primary font-bold">{hosts.length}</span> QUERY_RESULTS
          {totalCount !== hosts.length && (
            <span className="ml-1 text-[9px] text-muted-foreground">
              ({totalCount} TOTAL SUBNET HOSTS)
            </span>
          )}
        </div>
      </div>

      {hosts.length === 0 ? (
        <div className="text-center py-16 border border-border border-dashed bg-black/10">
          <p className="text-muted-foreground uppercase tracking-widest text-[10px]">// NO_RESPONSIVE_DEVICES_FOUND</p>
        </div>
      ) : (
        <>
          <div className="space-y-4">
            {paginatedHosts.map((host) => (
              <HostCard key={host.id} host={host} onClick={() => handleHostClick(host.ip)} />
            ))}
          </div>

          {totalPages > 1 && (
            <div className="flex items-center justify-center gap-1.5 pt-4">
              <Button
                variant="outline"
                size="sm"
                onClick={() => handlePageChange(currentPage - 1)}
                disabled={currentPage === 1}
                className="border-border hover:border-primary rounded-none font-mono text-[10px] tracking-widest uppercase bg-transparent text-foreground"
              >
                <ChevronLeft className="h-3 w-3 mr-1" />
                PREV
              </Button>

              {getPageNumbers().map((page, index) => (
                typeof page === 'number' ? (
                  <Button
                    key={index}
                    variant={currentPage === page ? 'default' : 'outline'}
                    size="sm"
                    onClick={() => handlePageChange(page)}
                    className={`min-w-[36px] h-8 rounded-none font-mono text-[11px] border-border ${
                      currentPage === page
                        ? 'bg-primary text-black font-bold hover:bg-primary/95'
                        : 'bg-transparent text-foreground hover:border-primary'
                    }`}
                  >
                    {page}
                  </Button>
                ) : (
                  <span key={index} className="px-2 text-muted-foreground self-end pb-1.5">
                    {page}
                  </span>
                )
              ))}

              <Button
                variant="outline"
                size="sm"
                onClick={() => handlePageChange(currentPage + 1)}
                disabled={currentPage === totalPages}
                className="border-border hover:border-primary rounded-none font-mono text-[10px] tracking-widest uppercase bg-transparent text-foreground"
              >
                NEXT
                <ChevronRight className="h-3 w-3 ml-1" />
              </Button>
            </div>
          )}
        </>
      )}
    </div>
  );
}
