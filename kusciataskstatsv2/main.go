package main

import (
	"fmt"
	"github.com/secretflow/kuscia/pkg/utils/nlog"
	"time"

	ctrnetio "kusciataskstats/container_netio"
	ctrstats "kusciataskstats/container_stats"
	ktcid "kusciataskstats/kusciatask_cid"
	ktpid "kusciataskstats/kusciatask_pid"
)

type NetStats struct {
	RecvBytes uint64
	XmitBytes uint64
	RecvBW    float64 // bps
	XmitBW    float64 // bps
}

type KusicaTaskStats struct {
	CtrStats       ctrstats.ContainerStats
	NetIO          NetStats
	CPUUsage       uint64
	VirtualMemory  uint64
	PhysicalMemory uint64
	ReadBytes      uint64
	WriteBytes     uint64
}

// DisplayKusicaTaskStats displays statistics for a specific task
func DisplayKusicaTaskStats(kusciaTaskID string, kusciaTaskStat KusicaTaskStats) {
	netStats := kusciaTaskStat.NetIO
	now := time.Now()
	timeStamp := now.Format(time.RFC3339)
	fmt.Println("Timestamp: ", timeStamp)
	fmt.Printf("KusciaTask ID: %s\n", kusciaTaskID)
	fmt.Printf("  Received Bytes: %d\n", netStats.RecvBytes)
	fmt.Printf("  Transmitted Bytes: %d\n", netStats.XmitBytes)
	fmt.Printf("  Received Bandwidth: %.2f bps\n", netStats.RecvBW)
	fmt.Printf("  Transmitted Bandwidth: %.2f bps\n", netStats.XmitBW)

	ctrStats := kusciaTaskStat.CtrStats
	if ctrStats.CPUPercentage == "" {
		ctrStats.CPUPercentage = "0"
	}
	if ctrStats.Memory == "" {
		ctrStats.Memory = "0MB"
	}
	if ctrStats.Disk == "" {
		ctrStats.Disk = "0B"
	}
	if ctrStats.Inodes == "" {
		ctrStats.Inodes = "0"
	}

	fmt.Printf("  CPU%%: %s\n", ctrStats.CPUPercentage)
	fmt.Printf("  Memory: %s\n", ctrStats.Memory)
	fmt.Printf("  Disk: %s\n", ctrStats.Disk)
	fmt.Printf("  Inodes: %s\n", ctrStats.Inodes)

	// Display additional stats
	fmt.Printf("  Total CPU Usage: %d ns\n", kusciaTaskStat.CPUUsage)
	fmt.Printf("  Total Virtual Memory: %d bytes\n", kusciaTaskStat.VirtualMemory)
	fmt.Printf("  Total Physical Memory: %d bytes\n", kusciaTaskStat.PhysicalMemory)
	fmt.Printf("  Total Received Bytes: %d\n", netStats.RecvBytes)
	fmt.Printf("  Total Transmitted Bytes: %d\n", netStats.XmitBytes)
}

func main() {
	checkInterval := 1 * time.Second
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	var preRecvBytes, preXmitBytes uint64
	timeWindow := 1.0

	for range ticker.C {
		taskToPID, err := ktpid.GetKusciaTaskPID()
		if err != nil {
			nlog.Error("Fail to get container PIDs", err)
			continue
		}

		taskIDToContainerID, err := ktcid.GetTaskIDToContainerID()
		if err != nil {
			nlog.Error("Fail to get container ID", err)
			continue
		}

		for kusciaTaskID, containerPID := range taskToPID {
			containerID, exists := taskIDToContainerID[kusciaTaskID]
			if !exists || containerID == "" {
				// Skip this task if no valid CID is found
				continue
			}

			recvBytes, xmitBytes, err := ctrnetio.GetContainerNetIOFromProc("eth0", containerPID)
			if err != nil {
				nlog.Warn("Fail to get container network IO from proc")
				continue
			}

			recvBW, xmitBW, err := ctrnetio.GetContainerBandwidth(recvBytes, preRecvBytes, xmitBytes, preXmitBytes, timeWindow)
			if err != nil {
				nlog.Warn("Fail to get the network bandwidth of containers")
				continue
			}

			preRecvBytes = recvBytes
			preXmitBytes = xmitBytes

			var kusciaTaskStat KusicaTaskStats
			kusciaTaskStat.NetIO.RecvBytes = recvBytes
			kusciaTaskStat.NetIO.XmitBytes = xmitBytes
			kusciaTaskStat.NetIO.RecvBW = recvBW
			kusciaTaskStat.NetIO.XmitBW = xmitBW

			// Get container stats
			containerStats, err := ctrstats.GetContainerStats()
			if err != nil {
				nlog.Warn("Fail to get the stats of containers")
				continue
			}
			kusciaTaskStat.CtrStats = containerStats[containerID]

			// Get CPU, Memory, and I/O stats
			cpuUsage, err := ctrstats.GetTotalCPUUsageStats(containerID)
			if err != nil {
				nlog.Warn("Fail to get the total CPU usage stats")
				continue
			}
			kusciaTaskStat.CPUUsage = cpuUsage

			virtualMemory, physicalMemory, err := ctrstats.GetMaxMemoryUsageStats(containerPID, containerID)
			if err != nil {
				nlog.Warn("Fail to get the total memory stats")
				continue
			}
			kusciaTaskStat.VirtualMemory = virtualMemory
			kusciaTaskStat.PhysicalMemory = physicalMemory

			readBytes, writeBytes, err := ctrstats.GetTotalIOStats(containerPID)
			if err != nil {
				nlog.Warn("Fail to get the total IO stats")
				continue
			}
			kusciaTaskStat.ReadBytes = readBytes
			kusciaTaskStat.WriteBytes = writeBytes

			// Display the stats for the current task
			DisplayKusicaTaskStats(kusciaTaskID, kusciaTaskStat)
		}
	}
}
