package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/process"
)

type collect struct {
	IntervalSec int
	fp          *os.File
}

func NewCollect(intervalSec int, dir string, serviceName string) (*collect, error) {
	if err := os.MkdirAll(dir, 0666); err != nil {
		return nil, err
	}
	filename := fmt.Sprintf("%s_%d_%s.dump",
		serviceName, os.Getpid(), time.Now().Format("20060102150405"))
	filename = path.Join(dir, filename)

	fp, err := os.Create(filename)
	if err != nil {
		return nil, err
	}
	return &collect{
		IntervalSec: intervalSec, fp: fp,
	}, nil
}

func (c *collect) doAsync() chan Info {
	ret := make(chan Info)
	go func() {
		p := process.Process{
			Pid: int32(os.Getpid()),
		}

		ret <- c.do(p)

		t := time.Tick(time.Second * time.Duration(c.IntervalSec))
		for {
			select {
			case <-t:
				ret <- c.do(p)
			}
		}
	}()
	return ret
}

func (c *collect) do(p process.Process) error {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	mis, _ := p.MemoryInfo()

	info := Info{
		Timestamp:    time.Now().Unix(),
		Sys:          ms.Sys,
		HeapSys:      ms.HeapSys,
		HeapAlloc:    ms.HeapAlloc,
		HeapInuse:    ms.HeapInuse,
		HeapReleased: ms.HeapReleased,
		HeapIdle:     ms.HeapIdle,
		VMS:          mis.VMS,
		RSS:          mis.RSS,
	}

	return c.save(info)
}

func (c *collect) save(info Info) error {
	raw, _ := json.Marshal(&info)
	_, err := c.fp.Write(raw)
	if err != nil {
		return err
	}
	_, err = c.fp.Write([]byte{'\n'})
	if err != nil {
		return err
	}
	return c.fp.Sync()
}