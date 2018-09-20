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

import "io"

// read n bytes from io.Reader
func ReadNBytes(rd io.Reader, size int64) ([]byte, error) {
	data := make([]byte, size)
	n, err := rd.Read(data)
	if n == 0 && err != nil {
		return nil, err
	}

	if n != 0 && int64(n) < size {
		return data, io.EOF
	}

	return data, nil
}
