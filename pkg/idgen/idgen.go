package idgen

// Generator ID生成器接口
type Generator interface {
	// NextID 生成下一个唯一ID
	NextID() (int64, error)
}
