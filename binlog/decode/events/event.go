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

package events

import (
	"fmt"

	"github.com/liipx/go-mysql-binlog/binlog/common"
	"github.com/liipx/go-mysql-binlog/binlog/decode/events/types"
)

// Event is binary log event definition
type Event struct {
	Header       *EventHeader
	ChecksumType byte
	ChecksumVal  []byte

	Body types.EventBody
}

// ValidateData event format
func (e *Event) ValidateData(body []byte, hasChecksum bool) ([]byte, error) {
	if e.Header == nil {
		return body, fmt.Errorf("empty evnet header")
	}

	if l := int64(len(body) + len(e.Header.Data)); l != e.Header.EventSize {
		return body, fmt.Errorf("event size got %d need %d", l, e.Header.EventSize)
	}

	if hasChecksum {
		index := len(body) - common.BinlogChecksumLength - 1
		e.ChecksumType = body[index]
		e.ChecksumVal = body[index+1:]
		body = body[:index+1]

		if !common.ChecksumValidate(e.ChecksumType, e.ChecksumVal, append(e.Header.Data, body...)) || len(e.ChecksumVal) != 4 {
			return body, fmt.Errorf("binlog checksum validation failed")
		}
	}

	return body, nil
}

func NewEvent() *Event {
	return &Event{}
}
