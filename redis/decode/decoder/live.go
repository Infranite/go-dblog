package decoder

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/Infranite/go-dblog"
	"github.com/Infranite/go-dblog/redis/decode/events/types"
)

type liveConn interface {
	io.ReadWriteCloser
}

// LiveDecoder streams Redis replication commands.
type LiveDecoder struct {
	ctx           context.Context
	source        dblog.Source
	conn          liveConn
	reader        *bufio.Reader
	plugins       []types.CommandPlugin
	startPosition int
	position      int
	handshook     bool
}

// NewLiveDecoder creates a decoder over a live Redis replication stream.
func NewLiveDecoder(ctx context.Context, source dblog.Source, dsn string, opts ...Option) (*LiveDecoder, error) {
	address, username, password, err := redisAddress(dsn)
	if err != nil {
		return nil, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("redis live connect: %w", err)
	}
	if source.Name == "" {
		source.Name = address
	}
	decoder := newLiveDecoder(ctx, source, conn, opts...)
	if username != "" || password != "" {
		var err error
		if username == "" {
			err = decoder.send("AUTH", password)
		} else {
			err = decoder.send("AUTH", username, password)
		}
		if err == nil {
			err = decoder.readOK()
		}
		if err != nil {
			_ = conn.Close()
			return nil, err
		}
	}
	return decoder, nil
}

func newLiveDecoder(ctx context.Context, source dblog.Source, conn liveConn, opts ...Option) *LiveDecoder {
	if ctx == nil {
		ctx = context.Background()
	}
	if source.Driver == "" {
		source.Driver = types.Driver
	}
	cfg := options{}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	return &LiveDecoder{
		ctx:           ctx,
		source:        source,
		conn:          conn,
		reader:        bufio.NewReader(conn),
		plugins:       cfg.commandPlugins,
		startPosition: cfg.startPosition,
	}
}

func (d *LiveDecoder) Events() iter.Seq2[dblog.Event, error] {
	return func(yield func(dblog.Event, error) bool) {
		if d == nil || d.reader == nil || d.conn == nil {
			return
		}
		done := make(chan struct{})
		go func() {
			select {
			case <-d.ctx.Done():
				_ = d.Close()
			case <-done:
			}
		}()
		defer close(done)

		if !d.handshook {
			if err := d.handshake(); err != nil {
				if d.ctx.Err() == nil {
					yield(nil, err)
				}
				return
			}
			d.handshook = true
		}

		for d.ctx.Err() == nil {
			command, raw, err := parseCommand(d.reader)
			if err != nil {
				if d.ctx.Err() == nil && !errors.Is(err, io.EOF) && !errors.Is(err, net.ErrClosed) {
					yield(nil, err)
				}
				return
			}
			d.position++
			if d.position <= d.startPosition {
				continue
			}
			if err := applyCommandPlugins(&command, d.plugins); err != nil {
				yield(nil, err)
				return
			}
			if !yield(types.NewEvent(d.source, d.position, raw, command), nil) {
				return
			}
		}
	}
}

func (d *LiveDecoder) Close() error {
	if d == nil || d.conn == nil {
		return nil
	}
	if err := d.conn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
		return err
	}
	return nil
}

func (d *LiveDecoder) handshake() error {
	if err := d.send("REPLCONF", "listening-port", "0"); err != nil {
		return err
	}
	if err := d.readOK(); err != nil {
		return err
	}
	if err := d.send("REPLCONF", "capa", "psync2"); err != nil {
		return err
	}
	if err := d.readOK(); err != nil {
		return err
	}
	if err := d.send("PSYNC", "?", "-1"); err != nil {
		return err
	}
	line, err := readLine(d.reader, new(bytes.Buffer))
	if err != nil {
		return fmt.Errorf("redis live psync: %w", err)
	}
	switch {
	case strings.HasPrefix(line, "+FULLRESYNC"):
		return skipRDB(d.reader)
	case strings.HasPrefix(line, "+CONTINUE"):
		return nil
	case strings.HasPrefix(line, "-"):
		return fmt.Errorf("redis live psync: %s", strings.TrimPrefix(line, "-"))
	default:
		return fmt.Errorf("%w: psync response %q", types.ErrInvalidRESP, line)
	}
}

func (d *LiveDecoder) send(parts ...string) error {
	var raw bytes.Buffer
	raw.WriteString("*" + strconv.Itoa(len(parts)) + respLineEnd)
	for _, part := range parts {
		raw.WriteString("$" + strconv.Itoa(len(part)) + respLineEnd)
		raw.WriteString(part)
		raw.WriteString(respLineEnd)
	}
	_, err := d.conn.Write(raw.Bytes())
	if err != nil {
		return fmt.Errorf("redis live write command: %w", err)
	}
	return nil
}

func (d *LiveDecoder) readOK() error {
	line, err := readLine(d.reader, new(bytes.Buffer))
	if err != nil {
		return fmt.Errorf("redis live read reply: %w", err)
	}
	if strings.HasPrefix(line, "+") {
		return nil
	}
	if strings.HasPrefix(line, "-") {
		return fmt.Errorf("redis live reply: %s", strings.TrimPrefix(line, "-"))
	}
	return fmt.Errorf("%w: reply %q", types.ErrInvalidRESP, line)
}

func skipRDB(reader *bufio.Reader) error {
	line, err := readLine(reader, new(bytes.Buffer))
	if err != nil {
		return fmt.Errorf("redis live read rdb header: %w", err)
	}
	if !strings.HasPrefix(line, "$") {
		return fmt.Errorf("%w: rdb header %q", types.ErrInvalidRESP, line)
	}
	if marker, ok := strings.CutPrefix(line, "$EOF:"); ok {
		return discardUntil(reader, []byte(marker))
	}
	size, err := strconv.Atoi(strings.TrimPrefix(line, "$"))
	if err != nil || size < 0 {
		return fmt.Errorf("%w: rdb length %q", types.ErrInvalidRESP, line)
	}
	if _, err := io.CopyN(io.Discard, reader, int64(size)); err != nil {
		return fmt.Errorf("redis live read rdb: %w", err)
	}
	return nil
}

func discardUntil(reader *bufio.Reader, marker []byte) error {
	if len(marker) == 0 {
		return fmt.Errorf("%w: empty rdb eof marker", types.ErrInvalidRESP)
	}
	window := make([]byte, 0, len(marker))
	for {
		b, err := reader.ReadByte()
		if err != nil {
			return fmt.Errorf("redis live read diskless rdb: %w", err)
		}
		window = append(window, b)
		if len(window) > len(marker) {
			copy(window, window[1:])
			window = window[:len(marker)]
		}
		if bytes.Equal(window, marker) {
			return nil
		}
	}
}

func redisAddress(dsn string) (string, string, string, error) {
	parsed, err := url.Parse(strings.TrimSpace(dsn))
	if err != nil || parsed.Scheme != "redis" || parsed.Host == "" {
		return "", "", "", types.ErrReaderRequired
	}
	username := parsed.User.Username()
	password, _ := parsed.User.Password()
	return ensurePort(parsed.Hostname(), parsed.Port()), username, password, nil
}

func ensurePort(host, port string) string {
	if port == "" {
		port = "6379"
	}
	return net.JoinHostPort(host, port)
}

var _ dblog.Decoder[dblog.Event] = (*LiveDecoder)(nil)
