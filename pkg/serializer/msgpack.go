// pkg/serializer/msgpack.go
package serializer

import (
	"bytes"
	"io"
	"reflect"

	"github.com/hashicorp/go-msgpack/v2/codec"
	"github.com/lk2023060901/xdooria/pkg/pool/bytebuff"
)

// msgpackHandle 是 msgpack 编解码的配置
// 参考 Consul 的配置: RawToString=true, MapType=map[string]interface{}
var msgpackHandle = &codec.MsgpackHandle{}

func init() {
	msgpackHandle.MapType = reflect.TypeOf(map[string]interface{}{})
	msgpackHandle.RawToString = true
}

// defaultSizeHint 默认的 buffer 容量提示
const defaultSizeHint = 256

// Encode 使用 msgpack 编码数据
func Encode(v interface{}) ([]byte, error) {
	return EncodeWithSizeHint(v, defaultSizeHint)
}

// EncodeWithSizeHint 使用 msgpack 编码数据，并指定预期大小
func EncodeWithSizeHint(v interface{}, sizeHint int) ([]byte, error) {
	buf := bytebuff.Get(sizeHint)
	defer bytebuff.Put(buf)

	enc := codec.NewEncoder(buf, msgpackHandle)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}

	// 复制数据，因为 buf 会被回收复用
	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, nil
}

// Decode 使用 msgpack 解码数据
func Decode(data []byte, v interface{}) error {
	dec := codec.NewDecoder(bytes.NewReader(data), msgpackHandle)
	return dec.Decode(v)
}

// NewEncoder 创建一个新的 msgpack 编码器
func NewEncoder(w io.Writer) *codec.Encoder {
	return codec.NewEncoder(w, msgpackHandle)
}

// NewDecoder 创建一个新的 msgpack 解码器
func NewDecoder(r io.Reader) *codec.Decoder {
	return codec.NewDecoder(r, msgpackHandle)
}
