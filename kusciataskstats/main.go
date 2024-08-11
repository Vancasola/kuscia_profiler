package main

import (
	"fmt"
	"time"

	//"github.com/secretflow/kuscia/pkg/utils/nlog"

	ctrnetio "kusciataskstats/container_netio"
	ctrstats "kusciataskstats/container_stats"
	ktcid "kusciataskstats/kusciatask_cid"
	ktpid "kusciataskstats/kusciatask_pid"
)

type NetStats struct {
	RecvBytes uint64  // Bytes
	XmitBytes uint64  // Bytes
	RecvBW    float64 // bps
	XmitBW    float64 // bps
}
type KusicaTaskStats struct {
	CtrStats ctrstats.ContainerStats
	NetIO    NetStats
}

func DisplayKusicaTaskStats(kusciaTaskStats map[string]KusicaTaskStats) {
	for kusciaTaskID, kusciaTaskStat := range kusciaTaskStats {
		netStats := kusciaTaskStat.NetIO
		now := time.Now()
		timeStamp := now.Format(time.RFC3339)
		fmt.Println("Timestamp: ", timeStamp)
		fmt.Printf("KusciaTask ID: %s\n", kusciaTaskID)
		fmt.Printf("  Received Bytes: %d\n", netStats.RecvBytes)
		fmt.Printf("  Transmitted Bytes: %d\n", netStats.XmitBytes)
		fmt.Printf("  Received Bandwidth: %.2f\n", netStats.RecvBW)
		fmt.Printf("  Transmitted Bandwidth: %.2f\n", netStats.XmitBW)

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
	}
}
func main() {
	checkInterval := 1 * time.Second
	timer := time.NewTimer(checkInterval)
	var preRecvBytes, preXmitBytes uint64
	timeWindow := 1.0
	kusciaTaskStats := make(map[string]KusicaTaskStats)
	for {
		select {
		case <-timer.C:
			taskToPID, err := ktpid.GetKusciaTaskPID()
			if err != nil {
				//nlog.Error("Fail to get container pids", err)
			}
			taskIDToContainerID, err := ktcid.GetTaskIDToContainerID()
			if err != nil {
				//nlog.Error("Fail to get container ID", err)
			}

			for kusciaTaskID, containerPID := range taskToPID {
				recvBytes, xmitBytes, err := ctrnetio.GetContainerNetIOFromProc("eth0", containerPID)
				var kusciaTaskStat KusicaTaskStats
				kusciaTaskStat.NetIO.RecvBytes = recvBytes
				kusciaTaskStat.NetIO.XmitBytes = xmitBytes
				if err != nil {
					nlog.Warn("Fail to get container network IO from proc")
				}
				recvBW, xmitBW, err := ctrnetio.GetContainerBandwidth(recvBytes, preRecvBytes, xmitBytes, preXmitBytes, timeWindow)
				if err != nil {
					nlog.Warn("Fail to get the network bandwidth of containers")
				}
				preRecvBytes = recvBytes
				preXmitBytes = xmitBytes
				kusciaTaskStat.NetIO.RecvBW = recvBW
				kusciaTaskStat.NetIO.XmitBW = xmitBW
				containerStats, err := ctrstats.GetContainerStats()
				containerID := taskIDToContainerID[kusciaTaskID]
				kusciaTaskStat.CtrStats = containerStats[containerID]
				kusciaTaskStats[kusciaTaskID] = kusciaTaskStat

				DisplayKusicaTaskStats(kusciaTaskStats)
			}
			// Reset the timer for the next interval
			timer.Reset(checkInterval)
		}
	}
}
