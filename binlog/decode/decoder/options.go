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
	"github.com/liipx/go-mysql-binlog/binlog/decode/events/types"
)

// BinFileDecodeOptFunc configures binlog file decoder option.
type BinFileDecodeOptFunc func(o *BinFileDecodeOption)

// EventCompatibilityMode controls how unknown event types are handled.
type EventCompatibilityMode uint8

const (
	// EventCompatibilityAuto checks event type with FORMAT_DESCRIPTION_EVENT metadata.
	EventCompatibilityAuto EventCompatibilityMode = iota
	// EventCompatibilityStrict rejects event types not built into this package.
	EventCompatibilityStrict
	// EventCompatibilityLoose keeps unknown event types as metadata events.
	EventCompatibilityLoose
)

// BinFileDecodeOption is the option of binlog file decoder.
type BinFileDecodeOption struct {
	StartPos          int64
	EndPos            int64
	StartTime         time.Time
	EndTime           time.Time
	CompatibilityMode EventCompatibilityMode
	EventPlugins      []types.EventPlugin
}

// NeedStart return bool of if start decoding
func (o *BinFileDecodeOption) NeedStart(header *events.EventHeader) bool {
	if o == nil {
		return true
	}
	if o.StartPos != 0 && header.LogPos-header.EventSize < o.StartPos {
		return false
	}
	if !o.StartTime.IsZero() && time.Unix(header.Timestamp, 0).Before(o.StartTime) {
		return false
	}
	return true
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

// WithStartPos set start position option
func WithStartPos(startPos int64) BinFileDecodeOptFunc {
	return func(o *BinFileDecodeOption) {
		o.StartPos = startPos
	}
}

// WithEndPos set end position option
func WithEndPos(endPos int64) BinFileDecodeOptFunc {
	return func(o *BinFileDecodeOption) {
		o.EndPos = endPos
	}
}

// WithStartTime set start time option
func WithStartTime(startTime time.Time) BinFileDecodeOptFunc {
	return func(o *BinFileDecodeOption) {
		o.StartTime = startTime
	}
}

// WithEndTime set end time option
func WithEndTime(endTime time.Time) BinFileDecodeOptFunc {
	return func(o *BinFileDecodeOption) {
		o.EndTime = endTime
	}
}

// WithEventCompatibilityMode set event compatibility mode option
func WithEventCompatibilityMode(mode EventCompatibilityMode) BinFileDecodeOptFunc {
	return func(o *BinFileDecodeOption) {
		o.CompatibilityMode = mode
	}
}

// WithEventPlugins set event plugin option
func WithEventPlugins(plugins ...types.EventPlugin) BinFileDecodeOptFunc {
	return func(o *BinFileDecodeOption) {
		o.EventPlugins = append(o.EventPlugins, plugins...)
	}
}
