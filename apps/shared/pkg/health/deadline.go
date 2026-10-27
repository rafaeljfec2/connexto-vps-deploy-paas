package health

import "time"

// ProbeDeadline is the moment a deploy stops probing. startPeriod is a grace
// window during which failures are expected, not a sleep before the first probe.
// Grace is at least minGrace so a missing or tiny startPeriod still leaves room
// for the process to bind its port.
func ProbeDeadline(now time.Time, startPeriod string, minGrace, probeTimeout time.Duration) time.Time {
	grace := minGrace
	if startPeriod != "" {
		parsed, err := time.ParseDuration(startPeriod)
		if err == nil && parsed > grace {
			grace = parsed
		}
	}
	return now.Add(grace + probeTimeout)
}
