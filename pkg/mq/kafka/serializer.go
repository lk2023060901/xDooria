package kafka

import "github.com/lk2023060901/xdooria/pkg/serializer"

// Serializer 序列化器接口（复用公共包）
type Serializer = serializer.Serializer

// 预定义序列化器
var (
	JSONSerializer  = serializer.NewJSON()
	ProtoSerializer = serializer.NewProto()
	RawSerializer   = serializer.NewRaw()
)
