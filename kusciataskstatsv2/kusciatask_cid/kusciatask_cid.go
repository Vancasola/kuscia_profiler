package kusciatask_cid

import (
	"bytes"
	"os/exec"
	"strings"

	"github.com/secretflow/kuscia/pkg/utils/nlog"
)

// GetContainerMappings fetches the container info using crictl ps command
func GetTaskIDToContainerID() (map[string]string, error) {
	cmd := exec.Command("crictl", "ps")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		nlog.Error("failed to execute crictl ps.", err)
		return nil, err
	}

	lines := strings.Split(out.String(), "\n")
	if len(lines) < 2 {
		nlog.Error("unexpected output format from crictl ps", err)
		return nil, err
	}

	taskIDToContainerID := make(map[string]string)
	for _, line := range lines[1:] {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 8 {
			nlog.Error("unexpected output format for line: %s", line)
			return nil, err
		}

		containerID := fields[0]
		kusciaTaskID := fields[len(fields)-1]
		taskIDToContainerID[kusciaTaskID] = containerID
	}

	return taskIDToContainerID, nil
}
