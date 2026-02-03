package sysinfo

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	pb "github.com/paasdeploy/backend/gen/go/flowdeploy/v1"
)

const (
	procMemInfo   = "/proc/meminfo"
	procCpuInfo   = "/proc/cpuinfo"
	procLoadAvg   = "/proc/loadavg"
	procNetDev    = "/proc/net/dev"
	etcOSRelease  = "/etc/os-release"
	defaultRootFS = "/"
)

func GetSystemInfo() (*pb.SystemInfo, error) {
	hostname, _ := os.Hostname()
	osName, osVersion := readOSRelease()
	arch := runtime.GOARCH
	kernel := readKernelVersion()
	cpuCores := countCPUCores()
	memTotal := readMemTotal()
	diskTotal := readDiskTotal()

	return &pb.SystemInfo{
		Hostname:         hostname,
		Os:               osName,
		OsVersion:        osVersion,
		Architecture:     arch,
		CpuCores:         int32(cpuCores),
		MemoryTotalBytes: memTotal,
		DiskTotalBytes:   diskTotal,
		KernelVersion:    kernel,
	}, nil
}

func GetSystemMetrics() (*pb.SystemMetrics, error) {
	cpuPercent := readCPUUsagePercent()
	memUsed, memAvail := readMemUsage()
	diskUsed, diskAvail := readDiskUsage()
	load1, load5, load15 := readLoadAvg()
	netRx, netTx := readNetworkBytes()

	return &pb.SystemMetrics{
		CpuUsagePercent:      cpuPercent,
		MemoryUsedBytes:      memUsed,
		MemoryAvailableBytes: memAvail,
		DiskUsedBytes:        diskUsed,
		DiskAvailableBytes:   diskAvail,
		LoadAverage_1M:       load1,
		LoadAverage_5M:       load5,
		LoadAverage_15M:      load15,
		NetworkRxBytes:       netRx,
		NetworkTxBytes:       netTx,
	}, nil
}

func readMemTotal() int64 {
	return readMemInfoKey("MemTotal")
}

func readMemUsage() (used, available int64) {
	total := readMemInfoKey("MemTotal")
	avail := readMemInfoKey("MemAvailable")
	if avail == 0 {
		avail = readMemInfoKey("MemFree")
	}
	return total - avail, avail
}

func readMemInfoKey(key string) int64 {
	f, err := os.Open(procMemInfo)
	if err != nil {
		return 0
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, key+":") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				val, _ := strconv.ParseInt(fields[1], 10, 64)
				if len(fields) >= 3 && fields[2] == "kB" {
					val *= 1024
				}
				return val
			}
		}
	}
	return 0
}

func countCPUCores() int {
	f, err := os.Open(procCpuInfo)
	if err != nil {
		return runtime.NumCPU()
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "processor") {
			count++
		}
	}
	if count > 0 {
		return count
	}
	return runtime.NumCPU()
}

func readOSRelease() (name, version string) {
	f, err := os.Open(etcOSRelease)
	if err != nil {
		return "linux", ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			s := strings.TrimPrefix(line, "PRETTY_NAME=")
			s = strings.Trim(s, "\"")
			parts := strings.SplitN(s, " ", 2)
			if len(parts) >= 1 {
				name = parts[0]
			}
			if len(parts) >= 2 {
				version = strings.Trim(parts[1], " ()")
			}
			return name, version
		}
		if strings.HasPrefix(line, "ID=") {
			name = strings.Trim(strings.TrimPrefix(line, "ID="), "\"")
		}
		if strings.HasPrefix(line, "VERSION_ID=") {
			version = strings.Trim(strings.TrimPrefix(line, "VERSION_ID="), "\"")
		}
	}
	return "linux", version
}

func readKernelVersion() string {
	var u syscall.Utsname
	if err := syscall.Uname(&u); err != nil {
		return ""
	}
	b := make([]byte, 0, len(u.Release))
	for _, c := range u.Release {
		if c == 0 {
			break
		}
		b = append(b, byte(c))
	}
	return string(b)
}

func readDiskTotal() int64 {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(defaultRootFS, &stat); err != nil {
		return 0
	}
	return int64(stat.Blocks) * int64(stat.Bsize)
}

func readDiskUsage() (used, available int64) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(defaultRootFS, &stat); err != nil {
		return 0, 0
	}
	blockSize := int64(stat.Bsize)
	total := int64(stat.Blocks) * blockSize
	avail := int64(stat.Bavail) * blockSize
	return total - avail, avail
}

func readLoadAvg() (load1, load5, load15 float64) {
	data, err := os.ReadFile(procLoadAvg)
	if err != nil {
		return 0, 0, 0
	}
	fields := strings.Fields(string(data))
	if len(fields) >= 3 {
		load1, _ = strconv.ParseFloat(fields[0], 64)
		load5, _ = strconv.ParseFloat(fields[1], 64)
		load15, _ = strconv.ParseFloat(fields[2], 64)
	}
	return load1, load5, load15
}

func readCPUUsagePercent() float64 {
	vals, err := readProcStatCPU()
	if err != nil || len(vals) < 4 {
		return 0
	}
	total := vals[0] + vals[1] + vals[2] + vals[3]
	idle := vals[3]
	if total == 0 {
		return 0
	}
	return 100 * (1 - float64(idle)/float64(total))
}

func readProcStatCPU() ([]uint64, error) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "cpu ") {
			fields := strings.Fields(line)[1:]
			vals := make([]uint64, 0, len(fields))
			for _, f := range fields {
				v, _ := strconv.ParseUint(f, 10, 64)
				vals = append(vals, v)
			}
			return vals, nil
		}
	}
	return nil, fmt.Errorf("cpu line not found")
}

func readNetworkBytes() (rx, tx int64) {
	f, err := os.Open(procNetDev)
	if err != nil {
		return 0, 0
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				continue
			}
			iface := strings.TrimSpace(parts[0])
			if iface == "lo" {
				continue
			}
			fields := strings.Fields(parts[1])
			if len(fields) >= 10 {
				r, _ := strconv.ParseInt(fields[0], 10, 64)
				t, _ := strconv.ParseInt(fields[8], 10, 64)
				rx += r
				tx += t
			}
		}
	}
	return rx, tx
}
