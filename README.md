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

## resource limit
CPU

- docker update --cpus 8 root-kuscia-autonomy-alice5

- docker update --cpu-period=100000 --cpu-quota=50000 root-kuscia-autonomy-alice5

Memory

- docker update root-kuscia-autonomy-alice5 --memory=20GiB --memory-swap=20GiB

Bandwidth/latency #100Mbps 20ms

- tc qdisc add dev eth0 root handle 1: tbf rate 100mbit burst 256kb latency 800ms 
- tc qdisc add dev eth0 parent 1:1 handle 10: netem delay 20msec limit 8000

清除限制

- tc qdisc del dev eth0 root

查看已有配置

- tc qdisc show dev eth0

## scale up filesystem

fdisk -l # check the existence of a filesystem
mkfs.ext4 /dev/sdb # create a filesystem
systemctl stop docker
cp -r /var/lib/docker /home/docker.bak # backup
du -sh /home/docker.bak # check size
mount /dev/sdb /var/lib/docker # mount to new disk
\cp -rf /home/docker.bak/* /var/lib/docker # copy data to new dir
systemctl daemon-reload && systemctl restart docker 
