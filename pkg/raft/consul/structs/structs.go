// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package structs

import (
	"reflect"
	"time"

	"github.com/hashicorp/go-msgpack/v2/codec"
)

const (
	// JitterFraction is a the limit to the amount of jitter we apply
	// to a user specified MaxQueryTime. We divide the specified time by
	// the fraction. So 16 == 6.25% limit of jitter. This jitter is also
	// applied to determine the wait time for RPCs with blocking queries.
	JitterFraction = 16
)

// MsgpackHandle is a shared handle for encoding/decoding msgpack payloads
var MsgpackHandle = &codec.MsgpackHandle{
	WriteExt: true,
	BasicHandle: codec.BasicHandle{
		DecodeOptions: codec.DecodeOptions{
			MapType: reflect.TypeOf(map[string]interface{}{}),
		},
	},
}

// QueryOptions is used to specify various flags for read queries
type QueryOptions struct {
	// Token is the ACL token ID. If not provided, the 'anonymous'
	// token is assumed for backwards compatibility.
	Token string

	// If set, wait until query exceeds given index. Must be provided
	// with MaxQueryTime.
	MinQueryIndex uint64

	// Provided with MinQueryIndex to wait for change.
	MaxQueryTime time.Duration

	// If set, any follower can service the request. Results
	// may be arbitrarily stale.
	AllowStale bool

	// If set, the leader must verify leadership prior to
	// servicing the request. Prevents a stale read.
	RequireConsistent bool

	// If set, the local agent may respond with an arbitrarily stale locally
	// cached response.
	UseCache bool

	// If set and AllowStale is true, will try first a stale
	// read, and then will perform a consistent read if stale
	// read is older than value.
	MaxStaleDuration time.Duration

	// MaxAge limits how old a cached value will be returned if UseCache is true.
	MaxAge time.Duration

	// MustRevalidate forces the agent to fetch a fresh version of a cached
	// resource or at least validate that the cached version is still fresh.
	MustRevalidate bool

	// StaleIfError specifies how stale the client will accept a cached response
	// if the servers are unavailable to fetch a fresh one.
	StaleIfError time.Duration

	// Filter specifies the go-bexpr filter expression to be used for
	// filtering the data prior to returning a response
	Filter string

	// AllowNotModifiedResponse indicates that if the MinIndex matches the
	// QueryMeta.Index, the response can be left empty and QueryMeta.NotModified
	// will be set to true to indicate the result of the query has not changed.
	AllowNotModifiedResponse bool
}

// IsRead is always true for QueryOption.
func (q QueryOptions) IsRead() bool {
	return true
}

// ConsistencyLevel display the consistency required by a request
func (q QueryOptions) ConsistencyLevel() string {
	if q.RequireConsistent {
		return "consistent"
	} else if q.AllowStale {
		return "stale"
	} else {
		return "leader"
	}
}

func (q QueryOptions) AllowStaleRead() bool {
	return q.AllowStale
}

func (q QueryOptions) TokenSecret() string {
	return q.Token
}

func (q *QueryOptions) SetTokenSecret(s string) {
	q.Token = s
}

// BlockingTimeout implements pool.BlockableQuery
func (q QueryOptions) BlockingTimeout(maxQueryTime, defaultQueryTime time.Duration) time.Duration {
	// Match logic in Server.blockingQuery.
	if q.MinQueryIndex > 0 {
		if q.MaxQueryTime > maxQueryTime {
			q.MaxQueryTime = maxQueryTime
		} else if q.MaxQueryTime <= 0 {
			q.MaxQueryTime = defaultQueryTime
		}
		// Timeout after maximum jitter has elapsed.
		q.MaxQueryTime += q.MaxQueryTime / JitterFraction

		return q.MaxQueryTime
	}
	return 0
}

func (q QueryOptions) HasTimedOut(start time.Time, rpcHoldTimeout, maxQueryTime, defaultQueryTime time.Duration) (bool, error) {
	// In addition to BlockingTimeout, allow for an additional rpcHoldTimeout buffer
	// in case we need to wait for a leader election.
	return time.Since(start) > rpcHoldTimeout+q.BlockingTimeout(maxQueryTime, defaultQueryTime), nil
}

// Decode is used to decode a MsgPack encoded object
func Decode(buf []byte, out interface{}) error {
	return codec.NewDecoderBytes(buf, MsgpackHandle).Decode(out)
}

// Encode is used to encode a MsgPack object
func Encode(msg interface{}) ([]byte, error) {
	var buf []byte
	err := codec.NewEncoderBytes(&buf, MsgpackHandle).Encode(msg)
	return buf, err
}
