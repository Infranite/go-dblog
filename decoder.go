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
			return err
		}

		// TODO: break function by BinFileDecoder.Options condition

		isContinue, err := f(event)
		if !isContinue || err != nil {
			return err
		}
	}
}

// DecodeEvent will decode a single event from binary log
func (decoder *BinFileDecoder) DecodeEvent(rd io.Reader) (*BinEvent, error) {
	if rd == nil {
		rd = decoder.BinFile
	}

	header, err := decodeEventHeader(rd, decoder.Description)
	if err != nil {
		return nil, err
	}

	event := &BinEvent{
		Header: header,
	}

	if _, ok := EventType2Str[event.Header.EventType]; !ok {
		event.Header.EventType = UnknownEvent
	}

	var eventBody BinEventBody
	// decode event body
	switch event.Header.EventType {
	case FormatDescriptionEvent:
		// FORMAT_DESCRIPTION_EVENT
		decoder.Description, err = decodeFmtDescEvent(rd, header)
		eventBody = decoder.Description
	case QueryEvent:
		// QUERY_EVENT
		eventBody, err = decodeQueryEvent(rd, header, decoder.Description)
	case XIDEvent:
		// XID_EVENT
		eventBody, err = decodeXIDEvent(rd, header, decoder.Description)
	case IntvarEvent:
		// INTVAR_EVENT
		eventBody, err = decodeIntvarEvent(rd, header, decoder.Description)
	case RotateEvent:
		// ROTATE_EVENT
		eventBody, err = decodeRotateEvent(rd, decoder.Description)
	case PreviousGTIDEvent, AnonymousGTIDEvent:
		// decode ignore event.
		// TODO: decode AnonymousGTIDEvent
		eventBody, err = decodeUnSupportEvent(rd, header, decoder.Description)
	case UnknownEvent:
		return nil, fmt.Errorf("got unknown event")
	default:
		// TODO more decoders for more events
		eventBody, err = decodeUnSupportEvent(rd, header, decoder.Description)
	}

	if err != nil {
		return nil, err
	}

	// set event body
	event.Body = eventBody

	return event, nil
}
