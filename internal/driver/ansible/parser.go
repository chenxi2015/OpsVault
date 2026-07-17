package ansible

import (
	"bufio"
	"encoding/json"
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

var pingHeaderRegexp = regexp.MustCompile(`^([a-zA-Z0-9\.\-_]+) \| (SUCCESS|CHANGED|FAILED|UNREACHABLE)(?:!)?\s*=>`)

// PingStatus holds the parsed system ping metrics.
type PingStatus struct {
	IP      string
	Status  string
	Message string
}

// ParsePingOutput parses the raw terminal stdout of the ping ad-hoc command.
func ParsePingOutput(raw string) []PingStatus {
	var results []PingStatus
	scanner := bufio.NewScanner(strings.NewReader(raw))

	type hostRaw struct {
		IP     string
		Status string
		Lines  []string
	}
	var current *hostRaw
	var hosts []hostRaw

	for scanner.Scan() {
		line := scanner.Text()
		matches := pingHeaderRegexp.FindStringSubmatch(line)
		if len(matches) > 0 {
			if current != nil {
				hosts = append(hosts, *current)
			}
			current = &hostRaw{
				IP:     matches[1],
				Status: matches[2],
			}
			if strings.Contains(line, "{") {
				current.Lines = append(current.Lines, "{")
			}
		} else {
			if current != nil {
				current.Lines = append(current.Lines, line)
			}
		}
	}
	if current != nil {
		hosts = append(hosts, *current)
	}

	for _, h := range hosts {
		status := h.Status
		var msg string
		
		// Attempt to extract JSON from lines
		jsonStr := extractJSONFromLines(h.Lines)
		
		cleanedJSON := strings.TrimSpace(jsonStr)
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(cleanedJSON), &data); err == nil {
			if pingVal, ok := data["ping"].(string); ok {
				msg = pingVal
			} else if msgVal, ok := data["msg"].(string); ok {
				msg = msgVal
			} else if stderrVal, ok := data["module_stderr"].(string); ok && stderrVal != "" {
				msg = strings.TrimSpace(stderrVal)
			}
		} else {
			// Fallback: join all lines if JSON extraction/parsing failed
			msg = strings.TrimSpace(strings.Join(h.Lines, "\n"))
		}

		if msg == "" {
			if status == "SUCCESS" || status == "CHANGED" {
				msg = "pong"
			} else {
				msg = "unknown error"
			}
		}

		msg = strings.ReplaceAll(msg, "\n", " ")
		msg = strings.ReplaceAll(msg, "\r", "")
		if len(msg) > 60 {
			msg = msg[:57] + "..."
		}

		results = append(results, PingStatus{
			IP:      h.IP,
			Status:  status,
			Message: msg,
		})
	}

	return results
}

func extractJSONFromLines(lines []string) string {
	var jsonLines []string
	inJSON := false
	braceCount := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "{") {
			inJSON = true
			braceCount += strings.Count(trimmed, "{")
		}
		if inJSON {
			jsonLines = append(jsonLines, line)
		}
		if strings.Contains(trimmed, "}") {
			braceCount -= strings.Count(trimmed, "}")
			if braceCount <= 0 && inJSON {
				break
			}
		}
	}
	return strings.Join(jsonLines, "\n")
}


