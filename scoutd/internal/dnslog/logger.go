package dnslog

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
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
	logDir     string
	devMode    bool
	lookup     DeviceLookup
	mu         sync.RWMutex
	entries    []Entry
	head       int
	count      int
	running    bool
}

// NewLogger creates a DNS query logger.
func NewLogger(logDir string, devMode bool, lookup DeviceLookup) *Logger {
	return &Logger{
		logDir:  logDir,
		devMode: devMode,
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

func (l *Logger) addEntry(e Entry) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.entries[l.head] = e
	l.head = (l.head + 1) % len(l.entries)
	if l.count < len(l.entries) {
		l.count++
	}
}

// Unbound log-queries format:
// [timestamp] unbound[pid]: [thread] info: IP query_name TYPE CLASS
var unboundQueryRe = regexp.MustCompile(
	`(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}).*info:\s+(\S+)\s+(\S+)\s+(\S+)`,
)

// ParseUnboundLine parses a single Unbound log line into an Entry.
// Returns ok=false if the line is not a query log entry.
func (l *Logger) ParseUnboundLine(line string) (Entry, bool) {
	m := unboundQueryRe.FindStringSubmatch(line)
	if m == nil {
		return Entry{}, false
	}

	timestamp := m[1]
	clientIP := m[2]
	domain := strings.TrimSuffix(m[3], ".")

	// Look up device info.
	var deviceMAC, deviceName string
	if l.lookup != nil {
		if mac, name, ok := l.lookup.LookupByIP(clientIP); ok {
			deviceMAC = mac
			deviceName = name
		}
	}

	action := "allow"
	if strings.Contains(line, "nxdomain") || strings.Contains(line, "NXDOMAIN") {
		action = "block"
	}

	return Entry{
		Timestamp:  timestamp,
		DeviceMAC:  deviceMAC,
		DeviceIP:   clientIP,
		DeviceName: deviceName,
		Domain:     domain,
		Action:     action,
	}, true
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
// Blocks until the file exists or ctx is cancelled. In dev-mode, returns immediately.
func (l *Logger) TailFile(logPath string, done <-chan struct{}) {
	if l.devMode {
		<-done
		return
	}

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

		scanner := bufio.NewScanner(f)
		for {
			select {
			case <-done:
				f.Close()
				return
			default:
			}

			if scanner.Scan() {
				line := scanner.Text()
				if entry, ok := l.ParseUnboundLine(line); ok {
					l.addEntry(entry)
				}
			} else {
				// No more lines — wait briefly and continue.
				time.Sleep(500 * time.Millisecond)
				// Check if file was rotated (inode changed).
				newInfo, err := os.Stat(logPath)
				if err != nil {
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
