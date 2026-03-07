package dnslog

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	DefaultLogDir  = "/var/lib/scout/logs"
	DefaultLogFile = "dns-query.log"
	MaxEntries     = 1000 // Ring buffer size for recent queries.
)

// Entry represents a single DNS query log entry.
type Entry struct {
	Timestamp  string `json:"timestamp"`
	DeviceMAC  string `json:"device_mac"`
	DeviceIP   string `json:"device_ip"`
	DeviceName string `json:"device_name"`
	Domain     string `json:"domain"`
	Action     string `json:"action"`
}

// DeviceLookup maps IP addresses to device info.
type DeviceLookup interface {
	LookupByIP(ip string) (mac, name string, ok bool)
}

// Logger tails Unbound's query log and produces structured per-device entries.
type Logger struct {
	logDir  string
	lookup  DeviceLookup
	mu      sync.RWMutex
	entries []Entry
	head    int
	count   int
	running bool
}

// NewLogger creates a DNS query logger.
func NewLogger(logDir string, lookup DeviceLookup) *Logger {
	return &Logger{
		logDir:  logDir,
		lookup:  lookup,
		entries: make([]Entry, MaxEntries),
	}
}

// RecentEntries returns the most recent n log entries.
func (l *Logger) RecentEntries(n int) []Entry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if n > l.count {
		n = l.count
	}
	if n == 0 {
		return nil
	}

	result := make([]Entry, n)
	start := (l.head - l.count + len(l.entries)) % len(l.entries)
	offset := l.count - n
	readFrom := (start + offset) % len(l.entries)

	for i := 0; i < n; i++ {
		result[i] = l.entries[(readFrom+i)%len(l.entries)]
	}
	return result
}

// AddEntry appends an entry to the ring buffer.
func (l *Logger) AddEntry(e Entry) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.entries[l.head] = e
	l.head = (l.head + 1) % len(l.entries)
	if l.count < len(l.entries) {
		l.count++
	}
}

// Unbound reply log format (log-replies: yes):
//   [1772738547] unbound[3220:0] info: 192.168.1.162 www.google.com. A IN NOERROR 0.012345 1 55
// The response code (NOERROR, NXDOMAIN, SERVFAIL, etc.) appears after the class field.
//
// Unbound query log format (log-queries: yes):
//   [1772738547] unbound[3220:0] info: 192.168.1.162 example.com. A IN
// Query lines have no response code — only 4 fields after "info:".
var unboundReplyRe = regexp.MustCompile(
	`\[(\d+)\].*info:\s+(\S+)\s+(\S+)\s+(\S+)\s+IN\s+(\S+)`,
)

// Fallback for query-only lines (no reply code).
var unboundQueryRe = regexp.MustCompile(
	`\[(\d+)\].*info:\s+(\S+)\s+(\S+)\s+(\S+)\s+IN\s*$`,
)

// ParseUnboundLine parses a single Unbound log line into an Entry.
// Only parses reply lines (which include the response code like NOERROR,
// NXDOMAIN). Query-only lines are skipped to avoid duplicates when both
// log-queries and log-replies are enabled. Falls back to query-only
// format if log-replies is not enabled.
func (l *Logger) ParseUnboundLine(line string) (Entry, bool) {
	// Try reply format first (has response code after IN).
	if m := unboundReplyRe.FindStringSubmatch(line); m != nil {
		return l.buildEntry(m[1], m[2], m[3], m[5]), true
	}

	// Fall back to query-only format (no response code).
	if m := unboundQueryRe.FindStringSubmatch(line); m != nil {
		return l.buildEntry(m[1], m[2], m[3], ""), true
	}

	return Entry{}, false
}

// buildEntry creates a log entry from parsed fields.
func (l *Logger) buildEntry(epochStr, clientIP, domain, rcode string) Entry {
	epoch, _ := strconv.ParseInt(epochStr, 10, 64)
	timestamp := time.Unix(epoch, 0).UTC().Format("2006-01-02 15:04:05")
	domain = strings.TrimSuffix(domain, ".")

	var deviceMAC, deviceName string
	if l.lookup != nil {
		if mac, name, ok := l.lookup.LookupByIP(clientIP); ok {
			deviceMAC = mac
			deviceName = name
		}
	}

	action := "allow"
	if rcode == "NXDOMAIN" || rcode == "REFUSED" {
		action = "block"
	}

	return Entry{
		Timestamp:  timestamp,
		DeviceMAC:  deviceMAC,
		DeviceIP:   clientIP,
		DeviceName: deviceName,
		Domain:     domain,
		Action:     action,
	}
}

// RotateLog rotates the DNS query log file if it exceeds maxSize bytes.
// Keeps maxFiles rotated copies.
func RotateLog(logDir string, maxSize int64, maxDays int) error {
	logPath := filepath.Join(logDir, DefaultLogFile)

	info, err := os.Stat(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if info.Size() < maxSize {
		return nil
	}

	// Rotate: rename current to .1, delete old ones.
	rotated := logPath + ".1"
	if err := os.Rename(logPath, rotated); err != nil {
		return fmt.Errorf("rotating log: %w", err)
	}

	// Clean up old rotated files beyond retention.
	cleanOldLogs(logDir, maxDays)

	return nil
}

func cleanOldLogs(logDir string, maxDays int) {
	entries, err := os.ReadDir(logDir)
	if err != nil {
		return
	}
	cutoff := time.Now().AddDate(0, 0, -maxDays)
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), DefaultLogFile+".") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			os.Remove(filepath.Join(logDir, e.Name()))
		}
	}
}

// TailFile tails a log file and processes new lines through the parser.
// Blocks until the file exists or done is closed.
func (l *Logger) TailFile(logPath string, done <-chan struct{}) {
	for {
		select {
		case <-done:
			return
		default:
		}

		f, err := os.Open(logPath)
		if err != nil {
			// Wait and retry.
			select {
			case <-done:
				return
			case <-time.After(5 * time.Second):
				continue
			}
		}

		// Seek to end — only tail new entries.
		f.Seek(0, 2)

		reader := bufio.NewReader(f)
		var partial string
		for {
			select {
			case <-done:
				f.Close()
				return
			default:
			}

			line, err := reader.ReadString('\n')
			if len(line) > 0 {
				partial += line
				if strings.HasSuffix(partial, "\n") {
					if entry, ok := l.ParseUnboundLine(strings.TrimRight(partial, "\n")); ok {
						l.AddEntry(entry)
					}
					partial = ""
				}
			}
			if err != nil {
				// No more data — wait briefly and retry.
				time.Sleep(500 * time.Millisecond)
				// Check if file was rotated (inode changed).
				newInfo, statErr := os.Stat(logPath)
				if statErr != nil {
					break // File gone, reopen.
				}
				oldInfo, _ := f.Stat()
				if oldInfo != nil && newInfo.Size() < oldInfo.Size() {
					break // File was truncated/rotated, reopen.
				}
			}
		}
		f.Close()

		log.Printf("dnslog: log file rotated, reopening %s", logPath)
	}
}
