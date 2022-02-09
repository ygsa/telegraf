/*
The MIT License (MIT)

procgather - gather process info, include cpu, mem, io, rlimit and common.
arstercz<arstercz@gmail.com>

*/

package main

import (
	"flag"
	"fmt"
	"time"
	"runtime"
	"sort"
	"os"
	"github.com/shirou/gopsutil/process"
)

func main() {
	// options
	var cpu, mem, io, limit, comm, all bool
	var pid int
	var prefix string

	flag.BoolVar(&cpu, "cpu", false, "whether gather process cpu usage")
	flag.BoolVar(&mem, "mem", false, "whether gather process mem usage")
	flag.BoolVar(&io, "io", false, "whether gather process io usage")
	flag.BoolVar(&limit, "limit", false, "whether gather process limit usage")
	flag.BoolVar(&comm, "comm", false, "whether gather process common usage")
	flag.BoolVar(&all, "all", true, "whether gather process all usage")
	flag.IntVar(&pid, "pid", 0, "gather process info with process ID")
	flag.StringVar(&prefix, "prefix", "", "add prefix string to the result fileds")

	flag.Parse()

	if pid <= 0 {
		fmt.Println("Invailid pid!")
		os.Exit(1)
	}

	isExist, err := process.PidExists(int32(pid))
	if err != nil {
		fmt.Printf("check pid error: %v\n", err)
		os.Exit(2)
	}

	if !isExist {
		fmt.Printf("Non-exist pid: %d\n", pid)
		os.Exit(3)
	}

	if cpu || mem || io || limit || comm {
		all = false
	}

	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		fmt.Printf("new process error: %v\n", err)
		os.Exit(4)
	}

	if prefix != "" {
		prefix = prefix + "_"
	}

	fields := map[string]interface{}{}

	if all || comm {
		ppid, err := proc.Ppid()
		if err == nil {
			fields[prefix+"ppid"] = ppid
		}

		numThreads, err := proc.NumThreads()
		if err == nil {
			fields[prefix + "num_threads"] = numThreads
		}

		fds, err := proc.NumFDs()
		if err == nil {
			fields[prefix + "num_fds"] = fds
		}

		ctx, err := proc.NumCtxSwitches()
		if err == nil {
			fields[prefix+"voluntary_context_switches"] = ctx.Voluntary
			fields[prefix+"involuntary_context_switches"] = ctx.Involuntary
		}

		faults, err := proc.PageFaults()
		if err == nil {
			fields[prefix+"minor_faults"] = faults.MinorFaults
			fields[prefix+"major_faults"] = faults.MajorFaults
			fields[prefix+"child_minor_faults"] = faults.ChildMinorFaults
			fields[prefix+"child_major_faults"] = faults.ChildMajorFaults
		}

		createdAt, err := proc.CreateTime() //Returns epoch in ms
		if err == nil {
			fields[prefix+"created_at"] = createdAt * 1000000 //Convert ms to ns
		}
	}

	if all || cpu {
		cpuTime, err := proc.Times()
		if err == nil {
			fields[prefix+"cpu_time_user"] = cpuTime.User
			fields[prefix+"cpu_time_system"] = cpuTime.System
			fields[prefix+"cpu_time_idle"] = cpuTime.Idle
			fields[prefix+"cpu_time_nice"] = cpuTime.Nice
			fields[prefix+"cpu_time_iowait"] = cpuTime.Iowait
			fields[prefix+"cpu_time_irq"] = cpuTime.Irq
			fields[prefix+"cpu_time_soft_irq"] = cpuTime.Softirq
			fields[prefix+"cpu_time_steal"] = cpuTime.Steal
			fields[prefix+"cpu_time_guest"] = cpuTime.Guest
			fields[prefix+"cpu_time_guest_nice"] = cpuTime.GuestNice
		}

		cpuPerc, err := proc.Percent(time.Duration(0))
		if err == nil {
			solarisMode := isSolaris()
			if solarisMode {
				fields[prefix+"cpu_usage"] = cpuPerc / float64(runtime.NumCPU())
			} else {
				fields[prefix+"cpu_usage"] = cpuPerc
			}
		}
	}

	if all || mem {
		memInfo, err := proc.MemoryInfo()
		if err == nil {
			fields[prefix+"memory_rss"] = memInfo.RSS
			fields[prefix+"memory_vms"] = memInfo.VMS
			fields[prefix+"memory_swap"] = memInfo.Swap
			fields[prefix+"memory_data"] = memInfo.Data
			fields[prefix+"memory_stack"] = memInfo.Stack
			fields[prefix+"memory_locked"] = memInfo.Locked
		}

		memPerc, err := proc.MemoryPercent()
		if err == nil {
			fields[prefix+"memory_usage"] = memPerc
		}
	}

	if all || io {
		ioStat, err := proc.IOCounters()
		if err == nil {
			fields[prefix+"read_count"] = ioStat.ReadCount
			fields[prefix+"write_count"] = ioStat.WriteCount
			fields[prefix+"read_bytes"] = ioStat.ReadBytes
			fields[prefix+"write_bytes"] = ioStat.WriteBytes
		}
	}

	if all || limit {
		rlims, err := proc.RlimitUsage(true)
		if err == nil {
			for _, rlim := range rlims {
				var name string
				switch rlim.Resource {
				case process.RLIMIT_CPU:
					name = "cpu_time"
				case process.RLIMIT_DATA:
					name = "memory_data"
				case process.RLIMIT_STACK:
					name = "memory_stack"
				case process.RLIMIT_RSS:
					name = "memory_rss"
				case process.RLIMIT_NOFILE:
					name = "num_fds"
				case process.RLIMIT_MEMLOCK:
					name = "memory_locked"
				case process.RLIMIT_AS:
					name = "memory_vms"
				case process.RLIMIT_LOCKS:
					name = "file_locks"
				case process.RLIMIT_SIGPENDING:
					name = "signals_pending"
				case process.RLIMIT_NICE:
					name = "nice_priority"
				case process.RLIMIT_RTPRIO:
					name = "realtime_priority"
				default:
					continue
				}

				fields[prefix+"rlimit_"+name+"_soft"] = rlim.Soft
				fields[prefix+"rlimit_"+name+"_hard"] = rlim.Hard
				if name != "file_locks" { // gopsutil doesn't currently track the used file locks count
					fields[prefix+name] = rlim.Used
				}
			}
		}
	}

	dumpFields(fields)
}

func isSolaris() bool {
	os := runtime.GOOS
	if os == "solaris" {
		return true
	}
	return false
}

func dumpFields(fields map[string]interface{}) {
	if len(fields) > 0 {
		keys := make([]string, 0, len(fields))
		for k := range fields {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			fmt.Printf("%s %v\n", k, fields[k])
		}
	} else {
		fmt.Println("  [note] empty result.")
	}
}
