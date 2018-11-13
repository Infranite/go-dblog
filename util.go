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
	"io"
)

// ReadNBytes read n bytes from io.Reader
func ReadNBytes(rd io.Reader, size int64) ([]byte, error) {
	data := make([]byte, size)
	n, err := rd.Read(data)
	total := n
	for int64(total) < size && err == nil {
		n, err = rd.Read(data[total:])
		total += n
	}

	if n == 0 && err != nil {
		return nil, err
	}

	return data, err
}

// FixedLengthInt will turn byte to uint64
// this function is from 'github.com/siddontang/go-mysql/replication/util.go'
func FixedLengthInt(buf []byte) uint64 {
	var num uint64
	for i, b := range buf {
		num |= uint64(b) << (uint(i) * 8)
	}
	return num
}

// LengthEncodedInt will decode byte to uint64
// this function is from 'github.com/siddontang/go-mysql/replication/util.go'
func LengthEncodedInt(b []byte) (num uint64, isNull bool, n int) {
	switch b[0] {

	// 251: NULL
	case 0xfb:
		n = 1
		isNull = true
		return

		// 252: value of following 2
	case 0xfc:
		num = uint64(b[1]) | uint64(b[2])<<8
		n = 3
		return

		// 253: value of following 3
	case 0xfd:
		num = uint64(b[1]) | uint64(b[2])<<8 | uint64(b[3])<<16
		n = 4
		return

		// 254: value of following 8
	case 0xfe:
		num = uint64(b[1]) | uint64(b[2])<<8 | uint64(b[3])<<16 |
			uint64(b[4])<<24 | uint64(b[5])<<32 | uint64(b[6])<<40 |
			uint64(b[7])<<48 | uint64(b[8])<<56
		n = 9
		return
	}

	// 0-250: value of first byte
	num = uint64(b[0])
	n = 1
	return
}

// LengthEnodedString will decode bytes
func LengthEnodedString(b []byte) ([]byte, bool, int, error) {
	// Get length
	num, isNull, n := LengthEncodedInt(b)
	if num < 1 {
		return nil, isNull, n, nil
	}

	n += int(num)

	// Check data length
	if len(b) >= n {
		return b[n-int(num) : n], false, n, nil
	}
	return nil, false, n, io.EOF
}
