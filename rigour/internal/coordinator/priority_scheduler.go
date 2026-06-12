package coordinator

import (
	"context"
	"log"
	"time"
)

// PortSchedule tracks when a port should be scanned next
type PortSchedule struct {
	Port         int
	Tier         PortTier
	LastScanned  time.Time
	NextScan     time.Time
	ScanInterval time.Duration
}

// Scheduler manages production-grade port scanning schedules
// Aligned with Shodan's 3,846 port list industry standard
type Scheduler struct {
	registry  *PortRegistry
	schedules map[int]*PortSchedule
}

// NewScheduler initializes a production scheduler with Shodan's port list
func NewScheduler() *Scheduler {
	registry, err := NewPortRegistry()
	if err != nil {
		log.Fatalf("Failed to initialize port registry: %v", err)
	}

	s := &Scheduler{
		registry:  registry,
		schedules: make(map[int]*PortSchedule),
	}
	s.initializeSchedules()
	return s
}

// initializeSchedules configures scan schedules for all ports
func (s *Scheduler) initializeSchedules() {
	// Define scan intervals per tier
	intervals := map[PortTier]time.Duration{
		TierCritical:   6 * time.Hour,       // Every 6h - remote access, high-risk
		TierHigh:       24 * time.Hour,      // Every 24h - web, email, databases
		TierICS:        24 * time.Hour,      // Every 24h - ICS/SCADA
		TierTop1000:    7 * 24 * time.Hour,  // Every 7 days - Nmap top-1000
		TierShodanFull: 30 * 24 * time.Hour, // Every 30 days - Full Shodan list
	}

	// Schedule all ports from registry
	for _, port := range s.registry.GetAllPorts() {
		def, _ := s.registry.GetPortInfo(port)
		interval := intervals[def.Tier]

		s.schedules[port] = &PortSchedule{
			Port:         port,
			Tier:         def.Tier,
			ScanInterval: interval,
			NextScan:     time.Now(), // All ports due immediately on first run
		}
	}

	log.Printf("Initialized scheduler with %d ports from Shodan list", len(s.schedules))
	stats := s.registry.GetTierStats()
	log.Printf("Port distribution - Critical: %d, High: %d, ICS: %d, Top1000: %d, Shodan: %d",
		stats["critical"], stats["high"], stats["ics"], stats["top1000"], stats["shodan_full"])
}

// GetDuePorts returns ports that need scanning now
func (s *Scheduler) GetDuePorts(ctx context.Context) []int {
	now := time.Now()
	var duePorts []int

	for port, schedule := range s.schedules {
		if now.After(schedule.NextScan) || now.Equal(schedule.NextScan) {
			duePorts = append(duePorts, port)
		}
	}

	return duePorts
}

// GetDuePortsByTier returns ports due for scanning, grouped by tier
// Useful for prioritizing critical ports in scan execution
func (s *Scheduler) GetDuePortsByTier(ctx context.Context) map[PortTier][]int {
	now := time.Now()
	byTier := make(map[PortTier][]int)

	for port, schedule := range s.schedules {
		if now.After(schedule.NextScan) || now.Equal(schedule.NextScan) {
			byTier[schedule.Tier] = append(byTier[schedule.Tier], port)
		}
	}

	return byTier
}

// MarkScanned updates the schedule after a port is scanned
func (s *Scheduler) MarkScanned(port int) {
	if schedule, exists := s.schedules[port]; exists {
		now := time.Now()
		schedule.LastScanned = now
		schedule.NextScan = now.Add(schedule.ScanInterval)

		// Only log critical/high ports to reduce noise
		if schedule.Tier <= TierHigh {
			log.Printf("Port %d marked scanned. Next scan: %s", port, schedule.NextScan.Format(time.RFC3339))
		}
	}
}

// GetNextScanTime returns when the next scan should occur
func (s *Scheduler) GetNextScanTime() time.Time {
	if len(s.schedules) == 0 {
		return time.Now()
	}

	nextScan := time.Now().Add(365 * 24 * time.Hour) // Far future
	for _, schedule := range s.schedules {
		if schedule.NextScan.Before(nextScan) {
			nextScan = schedule.NextScan
		}
	}

	return nextScan
}

// GetScheduleSummary returns a detailed summary of port schedules
func (s *Scheduler) GetScheduleSummary() map[string]interface{} {
	stats := s.registry.GetTierStats()

	return map[string]interface{}{
		"total_ports":       len(s.schedules),
		"critical_ports":    stats["critical"],
		"high_ports":        stats["high"],
		"ics_ports":         stats["ics"],
		"top1000_ports":     stats["top1000"],
		"shodan_full_ports": stats["shodan_full"],
		"next_scan":         s.GetNextScanTime().Format(time.RFC3339),
		"source":            "Shodan official port list (3,846 ports)",
	}
}

// GetPortRegistry returns the underlying port registry
// Useful for accessing port metadata
func (s *Scheduler) GetPortRegistry() *PortRegistry {
	return s.registry
}
