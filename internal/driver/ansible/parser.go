package ansible

import (
	"bufio"
	"regexp"
	"strings"
)

// HostStatus holds the parsed system inspection metrics.
type HostStatus struct {
	IP          string
	Status      string // SUCCESS, CHANGED, FAILED, UNREACHABLE
	Uptime      string
	MemTotal    string
	MemUsed     string
	DiskSize    string
	DiskUsed    string
	DiskUsePct  string
	DockerState string
	NginxState  string
	MinIOState  string
	RawOutput   string
}

var hostHeaderRegexp = regexp.MustCompile(`^([a-zA-Z0-9\.\-_]+) \| (CHANGED|SUCCESS|FAILED|UNREACHABLE)`)

// ParseDoctorOutput parses the raw terminal stdout of the inspection ad-hoc command.
func ParseDoctorOutput(raw string) []HostStatus {
	var results []HostStatus
	scanner := bufio.NewScanner(strings.NewReader(raw))

	var currentHost *HostStatus
	var currentLines []string

	flushCurrent := func() {
		if currentHost != nil {
			currentHost.RawOutput = strings.Join(currentLines, "\n")
			parseHostMetrics(currentHost)
			results = append(results, *currentHost)
		}
	}

	for scanner.Scan() {
		line := scanner.Text()
		matches := hostHeaderRegexp.FindStringSubmatch(line)
		if len(matches) > 0 {
			flushCurrent()
			currentHost = &HostStatus{
				IP:     matches[1],
				Status: matches[2],
			}
			currentLines = []string{}
		} else {
			if currentHost != nil {
				currentLines = append(currentLines, line)
			}
		}
	}
	flushCurrent()

	return results
}

func parseHostMetrics(h *HostStatus) {
	if h.Status == "UNREACHABLE" || h.Status == "FAILED" {
		return
	}

	lines := strings.Split(h.RawOutput, "\n")
	var section string

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// Detect section headers
		if strings.Contains(line, "===UPTIME===") {
			section = "uptime"
			continue
		} else if strings.Contains(line, "===FREE===") {
			section = "free"
			continue
		} else if strings.Contains(line, "===DF===") {
			section = "df"
			continue
		} else if strings.Contains(line, "===SERVICES===") {
			section = "services"
			continue
		}

		switch section {
		case "uptime":
			if strings.Contains(line, "load average") {
				parts := strings.Split(line, "up")
				if len(parts) > 1 {
					h.Uptime = "up " + strings.TrimSpace(strings.Split(parts[1], ",")[0])
				} else {
					h.Uptime = line
				}
			}
		case "free":
			if strings.HasPrefix(line, "Mem:") {
				fields := strings.Fields(line)
				if len(fields) >= 3 {
					h.MemTotal = fields[1] + "MB"
					h.MemUsed = fields[2] + "MB"
				}
			}
		case "df":
			if strings.HasSuffix(line, "/") || strings.Contains(line, " / ") || strings.Contains(line, " /") {
				fields := strings.Fields(line)
				// Format: Filesystem Size Used Avail Use% Mounted
				// Or: Name Size Used Avail Use% /
				if len(fields) >= 5 {
					// Usually the fields are: [Filesystem, Size, Used, Avail, Use%, Mounted]
					// We find the field with percentage, and the size/used fields around it.
					for idx, f := range fields {
						if strings.HasSuffix(f, "%") {
							h.DiskUsePct = f
							if idx >= 3 {
								h.DiskSize = fields[idx-3]
								h.DiskUsed = fields[idx-2]
							}
							break
						}
					}
				}
			}
		case "services":
			// First line of services is docker, second is nginx, third is minio
			if h.DockerState == "" {
				h.DockerState = line
			} else if h.NginxState == "" {
				h.NginxState = line
			} else if h.MinIOState == "" {
				h.MinIOState = line
			}
		}
	}
}
