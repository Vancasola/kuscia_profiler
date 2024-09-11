package container_stats

import (
	"os"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"bytes"
	"github.com/secretflow/kuscia/pkg/utils/nlog"
	"os/exec"
	"strings"
	"fmt"
)

// ContainerStats holds the stats information for a container
type ContainerStats struct {
	CPUPercentage  string
	Memory         string
	Disk           string
	Inodes         string
}

// GetContainerStats fetches the container stats using crictl stats command
func GetContainerStats() (map[string]ContainerStats, error) {
	// Execute the crictl stats command
	cmd := exec.Command("crictl", "stats")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		nlog.Warn("failed to execute crictl stats", err)
		return nil, err
	}

	// Parse the output
	lines := strings.Split(out.String(), "\n")
	if len(lines) < 2 {
		nlog.Warn("unexpected output format from crictl stats")
		return nil, err
	}

	// Create a map to hold the stats for each container
	statsMap := make(map[string]ContainerStats)

	// Skip the header line and iterate over the output lines
	for _, line := range lines[1:] {
		// Skip empty lines
		if line == "" {
			continue
		}

		// Split the line by whitespace
		fields := strings.Fields(line)
		if len(fields) < 5 {
			nlog.Warn("unexpected output format for line: %s", line)
			return nil, err
		}

		// Extract the stats information
		containerID := fields[0]
		stats := ContainerStats{
			CPUPercentage: fields[1],
			Memory:        fields[2],
			Disk:          fields[3],
			Inodes:        fields[4],
		}

		// Add the stats to the map
		statsMap[containerID] = stats
	}

	return statsMap, nil
}

// GetMemoryUsageStats retrieves memory usage statistics (virtual, physical, swap) for a given container
func GetMaxMemoryUsageStats(pid string, cidPrefix string) (uint64, uint64, error) {
	// Find the full CID based on the prefix
	cid, err := findCIDByPrefix(cidPrefix)
	if err != nil {
		nlog.Warn("failed to find full CID by prefix", err)
		return 0, 0, err
	}

	// Get Virtual Memory (VmPeak)
	virtualMemory, err := getVirtualMemoryUsage(pid)
	if err != nil {
		return 0, 0, err
	}

	// Get Physical and Swap Memory (max usage in bytes)
	physicalMemory, err := getPhysicalMemoryUsage(cid)
	if err != nil {
		return 0, 0, err
	}

	return virtualMemory, physicalMemory, nil
}

// getVirtualMemoryUsage retrieves the peak virtual memory usage for a given PID
func getVirtualMemoryUsage(pid string) (uint64, error) {
	// Read /proc/[pid]/status
	statusFile := filepath.Join("/proc", pid, "status")
	statusData, err := ioutil.ReadFile(statusFile)
	if err != nil {
		nlog.Warn("failed to read /proc/[pid]/status", err)
		return 0, err
	}

	// Parse VmPeak from status
	lines := strings.Split(string(statusData), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "VmPeak:") {
			parts := strings.Fields(line)
			if len(parts) == 3 {
				vmPeak, err := strconv.ParseUint(parts[1], 10, 64)
				if err != nil {
					nlog.Warn("failed to parse VmPeak value", err)
					return 0, err
				}
				return vmPeak * 1024, nil // Convert to bytes
			}
		}
	}

	nlog.Warn("VmPeak not found in /proc/[pid]/status")
	return 0, fmt.Errorf("VmPeak not found in /proc/[pid]/status")
}

// getPhysicalMemoryUsage retrieves the peak physical memory usage for a cgroup
func getPhysicalMemoryUsage(cid string) (uint64, error) {
	// Read /sys/fs/cgroup/memory/k8s.io/[cid]/memory.max_usage_in_bytes
	memMaxUsageFile := filepath.Join("/sys/fs/cgroup/memory/k8s.io", cid, "memory.max_usage_in_bytes")
	memMaxUsageData, err := ioutil.ReadFile(memMaxUsageFile)
	if err != nil {
		nlog.Warn("failed to read memory.max_usage_in_bytes", err)
		return 0, err
	}

	// Parse the max usage in bytes
	physicalMemory, err := strconv.ParseUint(strings.TrimSpace(string(memMaxUsageData)), 10, 64)
	if err != nil {
		nlog.Warn("failed to parse memory.max_usage_in_bytes", err)
		return 0, err
	}

	return physicalMemory, nil
}

// GetCPUUsageStats retrieves the CPU usage statistics for a given CID prefix
func GetTotalCPUUsageStats(cidPrefix string) (uint64, error) {
	//var stats CPUUsageStats

	// Find the full CID based on the prefix
	cid, err := findCIDByPrefix(cidPrefix)
	if err != nil {
		nlog.Warn("failed to find full CID by prefix", err)
		return 0, err
	}

	// Read /sys/fs/cgroup/cpu/k8s.io/[cid]/cpuacct.usage
	cgroupUsageFile := filepath.Join("/sys/fs/cgroup/cpu/k8s.io", cid, "cpuacct.usage")
	cgroupUsageData, err := ioutil.ReadFile(cgroupUsageFile)
	if err != nil {
		nlog.Warn("failed to read cpuacct.usage", err)
		return 0, err
	}

	// Parse cpuacct.usage
	cpuUsage, err := strconv.ParseUint(strings.TrimSpace(string(cgroupUsageData)), 10, 64)
	if err != nil {
		nlog.Warn("failed to parse cpuacct.usage", err)
		return 0, err
	}

	return cpuUsage, nil
}

// findCIDByPrefix searches for a full CID in the cgroup directory based on the given prefix
func findCIDByPrefix(prefix string) (string, error) {
	cgroupDir := "/sys/fs/cgroup/cpu/k8s.io/"
	files, err := ioutil.ReadDir(cgroupDir)
	if err != nil {
		return "", err
	}

	for _, file := range files {
		if strings.HasPrefix(file.Name(), prefix) {
			return file.Name(), nil
		}
	}

	return "", os.ErrNotExist
}

// GetIOStats retrieves I/O statistics (read_bytes, write_bytes) for a given PID
func GetTotalIOStats(pid string) (uint64, uint64, error) {
	// Read /proc/[pid]/io
	ioFile := filepath.Join("/proc", pid, "io")
	ioData, err := ioutil.ReadFile(ioFile)
	if err != nil {
		nlog.Warn("failed to read /proc/[pid]/io", err)
		return 0, 0, err
	}

	var readBytes, writeBytes uint64

	// Parse the file contents to find read_bytes and write_bytes
	lines := strings.Split(string(ioData), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "read_bytes:") {
			parts := strings.Fields(line)
			readBytes, err = strconv.ParseUint(parts[1], 10, 64)
			if err != nil {
				nlog.Warn("failed to parse read_bytes", err)
				return 0, 0, err
			}
		}
		if strings.HasPrefix(line, "write_bytes:") {
			parts := strings.Fields(line)
			writeBytes, err = strconv.ParseUint(parts[1], 10, 64)
			if err != nil {
				nlog.Warn("failed to parse write_bytes", err)
				return 0, 0, err
			}
		}
	}

	return readBytes, writeBytes, nil
}

