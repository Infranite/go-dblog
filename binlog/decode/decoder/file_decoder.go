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
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/liipx/go-mysql-binlog/binlog/common"
	"github.com/liipx/go-mysql-binlog/binlog/decode/events"
	"github.com/liipx/go-mysql-binlog/binlog/decode/events/types"
)

// binFileHeader : A binlog file starts with a Binlog File Header [ fe 'bin' ]
// https://dev.mysql.com/doc/internals/en/binlog-file-header.html
var binFileHeader = []byte{254, 98, 105, 110}

// BinFileDecoder will mapping a binary log file, decode binary log event
type BinFileDecoder struct {
	Path string          // binary log path
	prev *BinFileDecoder // prev binary log
	next *BinFileDecoder // next binary log

	// binary log reading options
	Option *BinFileDecodeOption

	// file object
	BinFile *os.File

	// buffer
	buf *bufio.Reader

	// context
	*types.EventContext
}

// init BinFileDecoder, binary log file validate
func (decoder *BinFileDecoder) init() error {
	// open binary log
	if decoder.BinFile == nil {
		binFile, err := os.Open(decoder.Path)
		if err != nil {
			return err
		}
		decoder.BinFile = binFile
		decoder.buf = bufio.NewReader(decoder.BinFile)
	}

	// binary log header validate
	header := make([]byte, 4)
	if _, err := decoder.BinFile.Read(header); err != nil {
		return err
	}

	if !bytes.Equal(header, binFileHeader) {
		return fmt.Errorf("invalid binary log header {%x}", header)
	}

	decoder.EventContext = types.NewEventContext()
	return nil
}

// decodeEventHeader
func (decoder *BinFileDecoder) decodeEventHeader() (*events.EventHeader, error) {
	headerLength := decoder.GetEventHeaderLength()
	// read from binary log file
	headerData, err := common.ReadNBytes(decoder.buf, headerLength)
	if err != nil {
		return nil, err
	}

	// decode event header
	header, err := events.DecodeEventHeader(headerData, headerLength)
	if err != nil {
		return nil, err
	}

	// header event validate
	if _, ok := common.EventType2Str[header.EventType]; !ok {
		return nil, fmt.Errorf("got unknown event type {%x}", header.EventType)
	}

	return header, nil
}

// readEventData
func (decoder *BinFileDecoder) readEventData(event *events.Event) ([]byte, error) {
	readDataLength := event.Header.EventSize - decoder.GetEventHeaderLength()
	data, err := common.ReadNBytes(decoder.buf, readDataLength)
	if err != nil {
		return nil, err
	}
	data, err = event.ValidateData(data, decoder.HasCheckSum())
	if err != nil {
		return nil, err
	}
	return data, nil
}

// checkSkip to check if decoding data needs to be skipped
func (decoder *BinFileDecoder) checkSkip(header *events.EventHeader) bool {
	// FMTEvent contains global information and cannot be skipped
	if header.EventType == common.FormatDescriptionEvent {
		return false
	}
	return !decoder.Option.NeedStart(header)
}

// DecodeEvent will decode a single event from binary log
func (decoder *BinFileDecoder) DecodeEvent() (*events.Event, error) {
	var err error
	event := events.NewEvent()

	// read & decode binlog event header
	event.Header, err = decoder.decodeEventHeader()
	if err != nil {
		return nil, err
	}
	// check if event detail needs to skip decode
	if decoder.checkSkip(event.Header) {
		return nil, nil
	}

	// read & validate event data
	data, err := decoder.readEventData(event)
	if err != nil {
		return event, err
	}

	bodyDecoder := types.GetEventBodyDecoder(event.Header.EventType)
	if bodyDecoder == nil {
		typeStr := ""
		if _, has := common.EventType2Str[event.Header.EventType]; has {
			typeStr = common.EventType2Str[event.Header.EventType]
		}
		return nil, fmt.Errorf("can't find decoder for event type %s[%x], may not suppoted event", typeStr, event.Header.EventType)
	}

	fmt.Println("start to decode ", common.EventType2Str[event.Header.EventType])
	event.Body, err = bodyDecoder.Decode(types.WithData(data), types.WithContext(decoder.EventContext))
	decoder.Description = decoder.EventContext.Description

	return event, nil
}

// WalkEvent will walk all events for binary log which in io.Reader
// This function will return isFinish bool and err error.
func (decoder *BinFileDecoder) WalkEvent(f func(event *events.Event) (isContinue bool, err error)) error {
	for {
		// if rd is nil, BinFileDecoder.DecodeEvent() will set rd to BinFileDecoder.BinFile
		event, err := decoder.DecodeEvent()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		// will receive a nil event if decoding not start yet
		if event == nil {
			continue
		}

		// if stop decoding
		if decoder.Option.NeedStop(event.Header) {
			return nil
		}

		isContinue, err := f(event)
		if !isContinue || err != nil {
			return err
		}
	}
}

// NewBinFileDecoder return a BinFileDecoder with binary log file path
func NewBinFileDecoder(path string, opts ...BinFileDecodeOptFunc) (*BinFileDecoder, error) {
	decoder := &BinFileDecoder{
		Path: path,
	}

	// set options
	for _, o := range opts {
		o(decoder.Option)
	}

	// decoder init
	return decoder, decoder.init()
}
