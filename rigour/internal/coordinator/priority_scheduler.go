package coordinator

import (
	"context"
	"log"
	"time"
)

// PortPriority defines scan frequency tiers
type PortPriority int

const (
	PriorityCritical PortPriority = iota // Every 6 hours (SSH, Telnet, RDP)
	PriorityHigh                         // Every 24 hours (HTTP, HTTPS)
	PriorityICS                          // Every 24 hours (ICS/SCADA)
	PriorityTop1000                      // Every 7 days (Top 1000 ports)
	PriorityFull                         // Every 30 days (All 65535 ports)
)

type PortSchedule struct {
	Port         int
	Priority     PortPriority
	LastScanned  time.Time
	NextScan     time.Time
	ScanInterval time.Duration
}

type Scheduler struct {
	schedules map[int]*PortSchedule
}

func NewScheduler() *Scheduler {
	s := &Scheduler{
		schedules: make(map[int]*PortSchedule),
	}
	s.initializeSchedules()
	return s
}

func (s *Scheduler) initializeSchedules() {
	// Critical ports - every 6h
	criticalPorts := []int{22, 23, 3389, 5900, 5901}
	for _, port := range criticalPorts {
		s.schedules[port] = &PortSchedule{
			Port:         port,
			Priority:     PriorityCritical,
			ScanInterval: 6 * time.Hour,
			NextScan:     time.Now(),
		}
	}

	// High priority - every 24h
	highPorts := []int{80, 443, 8080, 8443, 8000, 8888}
	for _, port := range highPorts {
		s.schedules[port] = &PortSchedule{
			Port:         port,
			Priority:     PriorityHigh,
			ScanInterval: 24 * time.Hour,
			NextScan:     time.Now(),
		}
	}

	// ICS/SCADA - every 24h
	icsPorts := []int{102, 502, 1089, 1911, 2222, 4000, 4840, 20000, 44818, 47808, 55000}
	for _, port := range icsPorts {
		s.schedules[port] = &PortSchedule{
			Port:         port,
			Priority:     PriorityICS,
			ScanInterval: 24 * time.Hour,
			NextScan:     time.Now(),
		}
	}

	// Top 1000 - every 7 days
	// (In production, load from nmap top-1000 ports list)
	top1000Ports := []int{21, 25, 110, 143, 445, 3306, 5432, 6379, 27017}
	for _, port := range top1000Ports {
		s.schedules[port] = &PortSchedule{
			Port:         port,
			Priority:     PriorityTop1000,
			ScanInterval: 7 * 24 * time.Hour,
			NextScan:     time.Now(),
		}
	}
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

// MarkScanned updates the schedule after a port is scanned
func (s *Scheduler) MarkScanned(port int) {
	if schedule, exists := s.schedules[port]; exists {
		now := time.Now()
		schedule.LastScanned = now
		schedule.NextScan = now.Add(schedule.ScanInterval)
		log.Printf("Port %d marked scanned. Next scan: %s", port, schedule.NextScan.Format(time.RFC3339))
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

// GetScheduleSummary returns a summary of all port schedules
func (s *Scheduler) GetScheduleSummary() map[string]interface{} {
	criticalCount := 0
	highCount := 0
	icsCount := 0
	top1000Count := 0

	for _, schedule := range s.schedules {
		switch schedule.Priority {
		case PriorityCritical:
			criticalCount++
		case PriorityHigh:
			highCount++
		case PriorityICS:
			icsCount++
		case PriorityTop1000:
			top1000Count++
		}
	}

	return map[string]interface{}{
		"total_ports":      len(s.schedules),
		"critical_ports":   criticalCount,
		"high_ports":       highCount,
		"ics_ports":        icsCount,
		"top1000_ports":    top1000Count,
		"next_scan":        s.GetNextScanTime().Format(time.RFC3339),
	}
}
