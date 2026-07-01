package decoder

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/Infranite/go-dblog/redis/decode/events/types"
)

// ParseCommand parses one Redis RESP array command.
func ParseCommand(reader io.Reader) (types.Command, []byte, error) {
	return parseCommand(bufio.NewReader(reader))
}

func parseCommand(reader *bufio.Reader) (types.Command, []byte, error) {
	var raw bytes.Buffer
	line, err := readLine(reader, &raw)
	if err != nil {
		return types.Command{}, raw.Bytes(), err
	}
	if !strings.HasPrefix(line, string(respArrayPrefix)) {
		return types.Command{}, raw.Bytes(), fmt.Errorf("%w: array header %q", types.ErrInvalidRESP, line)
	}
	count, err := strconv.Atoi(line[1:])
	if err != nil || count <= 0 {
		return types.Command{}, raw.Bytes(), fmt.Errorf("%w: array length %q", types.ErrInvalidRESP, line)
	}
	parts := make([]string, 0, count)
	for range count {
		part, err := readBulkString(reader, &raw)
		if err != nil {
			return types.Command{}, raw.Bytes(), err
		}
		parts = append(parts, part)
	}
	return types.Command{
		Name: strings.ToLower(parts[0]),
		Args: append([]string(nil), parts[1:]...),
	}, raw.Bytes(), nil
}

func readBulkString(reader *bufio.Reader, raw *bytes.Buffer) (string, error) {
	header, err := readLine(reader, raw)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(header, string(respBulkPrefix)) {
		return "", fmt.Errorf("%w: bulk header %q", types.ErrInvalidRESP, header)
	}
	n, err := strconv.Atoi(header[1:])
	if err != nil || n < 0 {
		return "", fmt.Errorf("%w: bulk length %q", types.ErrInvalidRESP, header)
	}
	data := make([]byte, n+len(respLineEnd))
	if _, err := io.ReadFull(reader, data); err != nil {
		return "", fmt.Errorf("%w: bulk payload: %v", types.ErrInvalidRESP, err)
	}
	raw.Write(data)
	if data[n] != respCR || data[n+1] != respLF {
		return "", fmt.Errorf("%w: bulk terminator", types.ErrInvalidRESP)
	}
	return string(data[:n]), nil
}

func readLine(reader *bufio.Reader, raw *bytes.Buffer) (string, error) {
	line, err := reader.ReadString(respLF)
	if err != nil {
		return "", err
	}
	raw.WriteString(line)
	line = strings.TrimSuffix(line, "\n")
	line = strings.TrimSuffix(line, "\r")
	return line, nil
}
