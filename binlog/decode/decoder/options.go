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

package decoder

import (
	"time"

	"github.com/liipx/go-mysql-binlog/binlog/decode/events"
)

type BinFileDecodeOptFunc func(o *BinFileDecodeOption)

type BinFileDecodeOption struct {
	StartPos  int64
	EndPos    int64
	StartTime time.Time
	EndTime   time.Time
}

// NeedStart return bool of if start decoding
func (o *BinFileDecodeOption) NeedStart(header *events.EventHeader) bool {
	if o == nil {
		return true
	} else if o.StartPos != 0 && o.StartPos <= header.LogPos-header.EventSize {
		return true
	} else if o.StartTime.Unix() <= time.Unix(header.Timestamp, 0).Unix() {
		return true
	}
	return false
}

// NeedStop return bool of if stop decoding
func (o *BinFileDecodeOption) NeedStop(header *events.EventHeader) bool {
	if o == nil {
		return false
	} else if o.EndPos != 0 && o.EndPos < header.LogPos {
		return true
	} else if !o.EndTime.IsZero() && o.EndTime.Unix() <= time.Unix(header.Timestamp, 0).Unix() {
		return true
	}
	return false
}

func WithStartPos(startPos int64) BinFileDecodeOptFunc {
	return func(o *BinFileDecodeOption) {
		o.StartPos = startPos
	}
}

func WithEndPos(endPos int64) BinFileDecodeOptFunc {
	return func(o *BinFileDecodeOption) {
		o.EndPos = endPos
	}
}

func WithStartTime(startTime time.Time) BinFileDecodeOptFunc {
	return func(o *BinFileDecodeOption) {
		o.StartTime = startTime
	}
}

func WithEndTime(endTime time.Time) BinFileDecodeOptFunc {
	return func(o *BinFileDecodeOption) {
		o.EndTime = endTime
	}
}
