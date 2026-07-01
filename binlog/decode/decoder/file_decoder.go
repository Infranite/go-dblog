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
	"iter"
	"os"

	"github.com/Infranite/go-mysql-binlog/binlog/common"
	"github.com/Infranite/go-mysql-binlog/binlog/decode/events"
	"github.com/Infranite/go-mysql-binlog/binlog/decode/events/types"
	"github.com/Infranite/go-mysql-binlog/binlog/plugin/mariadb"
)

// binFileHeader : A binlog file starts with a Binlog File Header [ fe 'bin' ]
// https://dev.mysql.com/doc/internals/en/binlog-file-header.html
var binFileHeader = []byte{254, 98, 105, 110}

// BinFileDecoder will mapping a binary log file, decode binary log event
type BinFileDecoder struct {
	Path string // binary log path

	// binary log reading options
	Option *BinFileDecodeOption

	// file object
	BinFile *os.File

	// buffer
	buf *bufio.Reader

	// context
	*types.EventContext

	pluginsApplied bool
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

	if !decoder.knowsEventType(header.EventType) {
		return nil, fmt.Errorf("got unknown event type {%x}", header.EventType)
	}

	return header, nil
}

func (decoder *BinFileDecoder) knowsEventType(eventType uint8) bool {
	if decoder.Option != nil && decoder.Option.CompatibilityMode == EventCompatibilityLoose {
		return true
	}
	if decoder.EventContext.Registry.KnowsEventType(eventType) {
		return true
	}
	if decoder.Option != nil && decoder.Option.CompatibilityMode == EventCompatibilityStrict {
		return false
	}
	return decoder.EventContext.KnowsEventType(eventType)
}

func (decoder *BinFileDecoder) applyEventPlugins(fde *types.FmtDescEvent) {
	if decoder.pluginsApplied || decoder.Option == nil || len(decoder.Option.EventPlugins) == 0 {
		return
	}
	for _, plugin := range decoder.Option.EventPlugins {
		if plugin.Match(fde) {
			plugin.Register(decoder.EventContext.Registry)
		}
	}
	decoder.pluginsApplied = true
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
		if _, err := decoder.readEventData(event); err != nil {
			return event, err
		}
		return nil, nil
	}

	// read & validate event data
	data, err := decoder.readEventData(event)
	if err != nil {
		return event, err
	}

	bodyDecoder := decoder.EventContext.Registry.GetEventBodyDecoder(event.Header.EventType)
	if bodyDecoder == nil {
		return nil, fmt.Errorf("can't find decoder for event type %s[%x], may not suppoted event",
			decoder.EventContext.Registry.EventTypeName(event.Header.EventType), event.Header.EventType)
	}

	event.Body, err = bodyDecoder.Decode(
		types.WithData(data),
		types.WithContext(decoder.EventContext),
		types.WithEventType(event.Header.EventType),
	)
	if err != nil {
		return event, err
	}
	if fde, ok := event.Body.(*types.FmtDescEvent); ok {
		decoder.applyEventPlugins(fde)
	}

	return event, nil
}

// Events returns decoded binlog events as a Go iterator.
func (decoder *BinFileDecoder) Events() iter.Seq2[*events.Event, error] {
	return func(yield func(*events.Event, error) bool) {
		for {
			event, err := decoder.DecodeEvent()
			if err != nil {
				if err != io.EOF {
					yield(nil, err)
				}
				return
			}

			if event == nil {
				continue
			}
			if decoder.Option.NeedStop(event.Header) {
				return
			}
			if !yield(event, nil) {
				return
			}
		}
	}
}

// EventBodies filters an event iterator by decoded body type.
func EventBodies[T types.EventBody](seq iter.Seq2[*events.Event, error]) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		var zero T
		for event, err := range seq {
			if err != nil {
				yield(zero, err)
				return
			}
			if event == nil || event.Body == nil {
				continue
			}
			body, ok := event.Body.(T)
			if !ok {
				continue
			}
			if !yield(body, nil) {
				return
			}
		}
	}
}

// WalkEvent will walk all events for binary log which in io.Reader
// This function will return isFinish bool and err error.
func (decoder *BinFileDecoder) WalkEvent(f func(event *events.Event) (isContinue bool, err error)) error {
	for event, err := range decoder.Events() {
		if err != nil {
			return err
		}
		isContinue, err := f(event)
		if !isContinue || err != nil {
			return err
		}
	}
	return nil
}

// Close closes the underlying binlog file.
func (decoder *BinFileDecoder) Close() error {
	if decoder == nil || decoder.BinFile == nil {
		return nil
	}
	return decoder.BinFile.Close()
}

// NewBinFileDecoder return a BinFileDecoder with binary log file path
func NewBinFileDecoder(path string, opts ...BinFileDecodeOptFunc) (*BinFileDecoder, error) {
	decoder := &BinFileDecoder{
		Path: path,
		Option: &BinFileDecodeOption{
			EventPlugins: []types.EventPlugin{
				mariadb.Plugin(),
			},
		},
	}

	// set options
	for _, o := range opts {
		o(decoder.Option)
	}

	// decoder init
	return decoder, decoder.init()
}
