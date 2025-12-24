// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package pool

import (
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/hashicorp/go-msgpack/v2/codec"
)

// ClientCodec is the interface for RPC client codec
type ClientCodec interface {
	WriteRequest(*Request, interface{}) error
	ReadResponseHeader(*Response) error
	ReadResponseBody(interface{}) error
	Close() error
}

// Request is the RPC request header
type Request struct {
	ServiceMethod string
	Seq           uint64
}

// Response is the RPC response header
type Response struct {
	ServiceMethod string
	Seq           uint64
	Error         string
}

// MsgpackClientCodec implements ClientCodec using msgpack encoding
type MsgpackClientCodec struct {
	conn    io.ReadWriteCloser
	enc     *codec.Encoder
	dec     *codec.Decoder
	pending map[uint64]string
	mu      sync.Mutex
}

// msgpackHandle is the shared handle for encoding/decoding msgpack payloads
var msgpackHandle = &codec.MsgpackHandle{
	WriteExt: true,
}

// NewMsgpackClientCodec creates a new msgpack client codec
func NewMsgpackClientCodec(conn io.ReadWriteCloser) *MsgpackClientCodec {
	return &MsgpackClientCodec{
		conn:    conn,
		enc:     codec.NewEncoder(conn, msgpackHandle),
		dec:     codec.NewDecoder(conn, msgpackHandle),
		pending: make(map[uint64]string),
	}
}

func (c *MsgpackClientCodec) WriteRequest(r *Request, body interface{}) error {
	c.mu.Lock()
	c.pending[r.Seq] = r.ServiceMethod
	c.mu.Unlock()

	// Encode the request header and body together
	if err := c.enc.Encode(r); err != nil {
		return err
	}
	return c.enc.Encode(body)
}

func (c *MsgpackClientCodec) ReadResponseHeader(r *Response) error {
	if err := c.dec.Decode(r); err != nil {
		return err
	}
	c.mu.Lock()
	r.ServiceMethod = c.pending[r.Seq]
	delete(c.pending, r.Seq)
	c.mu.Unlock()
	return nil
}

func (c *MsgpackClientCodec) ReadResponseBody(body interface{}) error {
	return c.dec.Decode(body)
}

func (c *MsgpackClientCodec) Close() error {
	return c.conn.Close()
}

// CallWithCodec makes an RPC call using the given codec
func CallWithCodec(codec ClientCodec, method string, args interface{}, reply interface{}) error {
	req := &Request{ServiceMethod: method, Seq: 1}
	if err := codec.WriteRequest(req, args); err != nil {
		return err
	}

	var resp Response
	if err := codec.ReadResponseHeader(&resp); err != nil {
		return err
	}

	if resp.Error != "" {
		return fmt.Errorf("rpc error: %s", resp.Error)
	}

	return codec.ReadResponseBody(reply)
}

// IsErrEOF returns true if we have an EOF-related error.
func IsErrEOF(err error) bool {
	if err == nil {
		return false
	}
	if err == io.EOF {
		return true
	}
	if err == io.ErrUnexpectedEOF {
		return true
	}
	errStr := err.Error()
	if strings.Contains(errStr, "EOF") {
		return true
	}
	if strings.Contains(errStr, "connection reset by peer") {
		return true
	}
	if strings.Contains(errStr, "stream closed") {
		return true
	}
	if strings.Contains(errStr, "session shutdown") {
		return true
	}
	return false
}
