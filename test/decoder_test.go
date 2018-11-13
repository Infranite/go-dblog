/*
Copyright 2018 liipx(lipengxiang)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package test

import (
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/liipx/go-mysql-binlog"
)

func TestDecoder(t *testing.T) {
	memStats := &runtime.MemStats{}
	runtime.ReadMemStats(memStats)
	decoder, err := binlog.NewBinFileDecoder("./testdata/mysql-bin.000004")

	if err != nil {
		t.Error(err)
		return
	}

	f, _ := decoder.BinFile.Stat()
	fmt.Println("Binlog file size:", f.Size()>>10>>10, "MB")
	starTime := time.Now()

	count := 0
	maxCount := 0
	err = decoder.WalkEvent(func(event *binlog.BinEvent) (isContinue bool, err error) {
		fmt.Println(event.Header)
		count++
		return maxCount > count || maxCount == 0, nil
	})

	duration := time.Since(starTime)
	fmt.Println("Time total:", duration.String())

	speed := float64(f.Size()>>10>>10) / duration.Seconds()
	fmt.Printf("Speed: %.2f MB/s\n", speed)

	if err != nil {
		t.Error(err)
	}

	runtime.ReadMemStats(memStats)
	fmt.Println("GC times:", memStats.NumGC)
	pauseTotal := time.Duration(int64(memStats.PauseTotalNs))
	fmt.Println("Pause total:", pauseTotal.String())
}
