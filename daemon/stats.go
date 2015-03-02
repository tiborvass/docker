package daemon

import (
	"encoding/json"

	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/daemon/execdriver"
	"github.com/tiborvass/docker/engine"
	"github.com/docker/libcontainer"
	"github.com/docker/libcontainer/cgroups"
)

func (daemon *Daemon) ContainerStats(job *engine.Job) engine.Status {
	updates, err := daemon.SubscribeToContainerStats(job.Args[0])
	if err != nil {
		return job.Error(err)
	}
	enc := json.NewEncoder(job.Stdout)
	for v := range updates {
		update := v.(*execdriver.ResourceStats)
		ss := convertToAPITypes(update.ContainerStats)
		ss.MemoryStats.Limit = uint64(update.MemoryLimit)
		ss.Read = update.Read
		ss.CpuStats.SystemUsage = update.SystemUsage
		if err := enc.Encode(ss); err != nil {
			// TODO: handle the specific broken pipe
			daemon.UnsubscribeToContainerStats(job.Args[0], updates)
			return job.Error(err)
		}
	}
	return engine.StatusOK
}

// convertToAPITypes converts the libcontainer.ContainerStats to the api specific
// structs.  This is done to preserve API compatibility and versioning.
func convertToAPITypes(ls *libcontainer.ContainerStats) *types.Stats {
	s := &types.Stats{}
	if ls.NetworkStats != nil {
		s.Network = types.Network{
			RxBytes:   ls.NetworkStats.RxBytes,
			RxPackets: ls.NetworkStats.RxPackets,
			RxErrors:  ls.NetworkStats.RxErrors,
			RxDropped: ls.NetworkStats.RxDropped,
			TxBytes:   ls.NetworkStats.TxBytes,
			TxPackets: ls.NetworkStats.TxPackets,
			TxErrors:  ls.NetworkStats.TxErrors,
			TxDropped: ls.NetworkStats.TxDropped,
		}
	}
	cs := ls.CgroupStats
	if cs != nil {
		s.BlkioStats = types.BlkioStats{
			IoServiceBytesRecursive: copyBlkioEntry(cs.BlkioStats.IoServiceBytesRecursive),
			IoServicedRecursive:     copyBlkioEntry(cs.BlkioStats.IoServicedRecursive),
			IoQueuedRecursive:       copyBlkioEntry(cs.BlkioStats.IoQueuedRecursive),
			IoServiceTimeRecursive:  copyBlkioEntry(cs.BlkioStats.IoServiceTimeRecursive),
			IoWaitTimeRecursive:     copyBlkioEntry(cs.BlkioStats.IoWaitTimeRecursive),
			IoMergedRecursive:       copyBlkioEntry(cs.BlkioStats.IoMergedRecursive),
			IoTimeRecursive:         copyBlkioEntry(cs.BlkioStats.IoTimeRecursive),
			SectorsRecursive:        copyBlkioEntry(cs.BlkioStats.SectorsRecursive),
		}
		cpu := cs.CpuStats
		s.CpuStats = types.CpuStats{
			CpuUsage: types.CpuUsage{
				TotalUsage:        cpu.CpuUsage.TotalUsage,
				PercpuUsage:       cpu.CpuUsage.PercpuUsage,
				UsageInKernelmode: cpu.CpuUsage.UsageInKernelmode,
				UsageInUsermode:   cpu.CpuUsage.UsageInUsermode,
			},
			ThrottlingData: types.ThrottlingData{
				Periods:          cpu.ThrottlingData.Periods,
				ThrottledPeriods: cpu.ThrottlingData.ThrottledPeriods,
				ThrottledTime:    cpu.ThrottlingData.ThrottledTime,
			},
		}
		mem := cs.MemoryStats
		s.MemoryStats = types.MemoryStats{
			Usage:    mem.Usage,
			MaxUsage: mem.MaxUsage,
			Stats:    mem.Stats,
			Failcnt:  mem.Failcnt,
		}
	}
	return s
}

func copyBlkioEntry(entries []cgroups.BlkioStatEntry) []types.BlkioStatEntry {
	out := make([]types.BlkioStatEntry, len(entries))
	for i, re := range entries {
		out[i] = types.BlkioStatEntry{
			Major: re.Major,
			Minor: re.Minor,
			Op:    re.Op,
			Value: re.Value,
		}
	}
	return out
}
