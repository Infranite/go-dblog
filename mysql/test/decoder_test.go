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
	"errors"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/Infranite/go-dblog/mysql/decode/decoder"
	"github.com/Infranite/go-dblog/mysql/decode/events"
)

func TestDecoder(t *testing.T) {
	const binlogPath = "./testdata/mysql-bin.000004"
	if _, err := os.Stat(binlogPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Skipf("test binlog %s not found; run testdata/generate_mysql_binlog.sh", binlogPath)
		}
		t.Fatal(err)
	}

	memStats := &runtime.MemStats{}
	runtime.ReadMemStats(memStats)
	fileDecoder, err := decoder.NewBinFileDecoder(binlogPath)

	if err != nil {
		t.Error(err)
		return
	}
	t.Cleanup(func() {
		if err := fileDecoder.Close(); err != nil {
			t.Fatal(err)
		}
	})

	f, err := fileDecoder.BinFile.Stat()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Binlog file size: %d MB", f.Size()>>10>>10)
	startTime := time.Now()

	count := 0
	maxCount := 0
	err = fileDecoder.WalkEvent(func(event *events.Event) (isContinue bool, err error) {
		t.Log(event.Header)
		count++
		return maxCount > count || maxCount == 0, nil
	})

	duration := time.Since(startTime)
	t.Logf("Time total: %s", duration)

	speed := float64(f.Size()>>10>>10) / duration.Seconds()
	t.Logf("Speed: %.2f MB/s", speed)

	if err != nil {
		t.Error(err)
	}

	runtime.ReadMemStats(memStats)
	t.Logf("GC times: %d", memStats.NumGC)
	pauseTotal := time.Duration(int64(memStats.PauseTotalNs))
	t.Logf("Pause total: %s", pauseTotal)
}

func BenchmarkDecoder(b *testing.B) {
	const binlogPath = "./testdata/mysql-bin.000004"
	if _, err := os.Stat(binlogPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			b.Skipf("test binlog %s not found; run testdata/generate_mysql_binlog.sh", binlogPath)
		}
		b.Fatal(err)
	}

	info, err := os.Stat(binlogPath)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.SetBytes(info.Size())
	for i := 0; i < b.N; i++ {
		fileDecoder, err := decoder.NewBinFileDecoder(binlogPath)
		if err != nil {
			b.Fatal(err)
		}
		err = fileDecoder.WalkEvent(func(*events.Event) (bool, error) {
			return true, nil
		})
		if closeErr := fileDecoder.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
		if err != nil {
			b.Fatal(err)
		}
	}
}
