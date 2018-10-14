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

package binlog

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/kr/pretty"
	"github.com/pkg/errors"
)

// binFileHeader : A binlog file starts with a Binlog File Header [ fe 'bin' ]
// https://dev.mysql.com/doc/internals/en/binlog-file-header.html
var binFileHeader = []byte{254, 98, 105, 110}

// BinReaderOption will describe the details to tell decoders when it should start and when stop.
type BinReaderOption struct {
	StartPos  int
	UntilPos  int
	StartTime time.Time
	UntilTime time.Time
}

// BinFileDecoder will mapping a binary log file, decode binary log event
type BinFileDecoder struct {
	Path string          // binary log path
	prev *BinFileDecoder // prev binary log
	next *BinFileDecoder // next binary log

	// binary log reading options
	Option *BinReaderOption

	// file object
	BinFile *os.File

	// cause different version mapping different payload
	// every binary log event analysis depend on descriptions
	Description *BinFmtDescEvent
}

// NewBinFileDecoder return a BinFileDecoder with binary log file path
func NewBinFileDecoder(path string, options ...*BinReaderOption) (*BinFileDecoder, error) {
	decoder := &BinFileDecoder{
		Path: path,
	}
	// set options
	if len(options) > 0 {
		decoder.Option = options[0]
	}

	// decoder init
	return decoder, decoder.init()
}

// Init BinFileDecoder, binary log file validate
func (decoder *BinFileDecoder) init() error {
	// open binary log
	if decoder.BinFile == nil {
		binFile, err := os.Open(decoder.Path)
		if err != nil {
			return err
		}
		decoder.BinFile = binFile
	}

	// binary log header validate
	header := make([]byte, 4)
	if _, err := decoder.BinFile.Read(header); err != nil {
		return err
	}

	if !bytes.Equal(header, binFileHeader) {
		return fmt.Errorf("invalid binary log header {%x}", header)
	}

	return nil
}

// DecodeEvent will decode a single event from binary log
func (decoder *BinFileDecoder) DecodeEvent(rd io.Reader) (*BinEvent, error) {
	if rd == nil {
		rd = decoder.BinFile
	}

	event := &BinEvent{}

	eventHeaderLength := defaultEventHeaderSize
	if decoder.Description != nil {
		eventHeaderLength = decoder.Description.EventHeaderLength
	}

	// read binlog event header
	headerData, err := ReadNBytes(rd, eventHeaderLength)
	if err != nil {
		return nil, err
	}

	// decode binlog event header
	event.Header, err = decodeEventHeader(headerData, eventHeaderLength)
	if err != nil {
		return nil, err
	}

	if _, ok := EventType2Str[event.Header.EventType]; !ok {
		return nil, fmt.Errorf("got unknown event type {%x}", event.Header.EventType)
	}

	// read binlog event body
	var data []byte
	data, err = ReadNBytes(rd, event.Header.EventSize-eventHeaderLength)
	if err != nil {
		return nil, err
	}

	if l := len(data); int64(l)+eventHeaderLength != event.Header.EventSize {
		return event, errors.Errorf("event size need %d got %d", l, event.Header.EventSize)
	}

	// checksum
	if decoder.Description != nil && decoder.Description.hasCheckSum {
		index := len(data) - binlogChecksumLength - 1
		event.ChecksumType = data[index]
		event.ChecksumVal = data[index+1:]
		data = data[:index+1]

		if !ChecksumValidate(event.ChecksumType, event.ChecksumVal, append(headerData, data...)) || len(event.ChecksumVal) != 4 {
			return event, errors.Errorf("binlog checksum validation failed")
		}
	}

	// decode binlog event body
	var eventBody BinEventBody
	switch event.Header.EventType {
	case FormatDescriptionEvent:
		// FORMAT_DESCRIPTION_EVENT
		decoder.Description, err = decodeFmtDescEvent(data)
		eventBody = decoder.Description
	case QueryEvent:
		// QUERY_EVENT
		eventBody, err = decodeQueryEvent(data, decoder.Description.BinlogVersion)
	case XIDEvent:
		// XID_EVENT
		eventBody, err = decodeXIDEvent(data)
	case IntvarEvent:
		// INTVAR_EVENT
		eventBody, err = decodeIntvarEvent(data)
	case RotateEvent:
		// ROTATE_EVENT
		eventBody, err = decodeRotateEvent(data, decoder.Description.BinlogVersion)
	case PreviousGTIDEvent, AnonymousGTIDEvent:
		// decode ignore event.
		// TODO: decode AnonymousGTIDEvent
		eventBody, err = decodeUnSupportEvent(data)
	case UnknownEvent:
		return nil, fmt.Errorf("got unknown event")
	default:
		// TODO more decoders for more events
		eventBody, err = decodeUnSupportEvent(data)
	}

	if err != nil {
		return nil, err
	}

	// set event body
	event.Body = eventBody

	return event, nil
}

// WalkEvent will walk all events for binary log which in io.Reader
// This function will return isFinish bool and err error.
func (decoder *BinFileDecoder) WalkEvent(f func(event *BinEvent) (isContinue bool, err error), rd io.Reader) error {
	for {
		// if rd is nil, BinFileDecoder.DecodeEvent() will set rd to BinFileDecoder.BinFile
		event, err := decoder.DecodeEvent(rd)
		if err == io.EOF {
			return nil
		}

		if err != nil {
			pretty.Println(event)
			return err
		}

		// TODO: break function by BinFileDecoder.Options condition

		isContinue, err := f(event)
		if !isContinue || err != nil {
			return err
		}
	}
}
