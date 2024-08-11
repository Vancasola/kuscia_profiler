# kuscia_profiler

This project aims to profile Kuscia regarding jobs, tasks, pods, containers, etc.

https://github.com/secretflow/kuscia

## Kuscia task stats

### supported metrics

kusciataskstats aims to collect the metrics of system resources used by a Kussia task (runC), including:

sys stats:

- CPU usage
- Memory usage
- Disk IO
- Inode


net stats:

- ReceivedBytes
- SentBytes
- ReceivedBandwidth
- SentBytes

### implementation

A Kuscia task is implemented as a container. 

- sys stats: kusciataskstats leverage a CRI cmd tool (crictl) to obtain the statistics of each container and the container ID.

- net stats: kusciataskstats (1) identify the PID (process ID) of each Kuscia task (implemented as a container), (2) read the network statistics from '/proc/%s/net/dev', where %s is the PID.

- associating a Kuscia task ID with its container ID. 
