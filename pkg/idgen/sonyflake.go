package idgen

import (
	"time"

	"github.com/cockroachdb/errors"
	"github.com/sony/sonyflake"
)

type sonyflakeGenerator struct {
	sf *sonyflake.Sonyflake
}

// NewSonyflake 创建基于 Sonyflake 的ID生成器
// machineID: 机器ID (0-65535)
func NewSonyflake(machineID uint16) (Generator, error) {
	settings := sonyflake.Settings{
		StartTime: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		MachineID: func() (uint16, error) {
			return machineID, nil
		},
	}

	sf := sonyflake.NewSonyflake(settings)
	if sf == nil {
		return nil, errors.New("failed to create sonyflake generator")
	}

	return &sonyflakeGenerator{sf: sf}, nil
}

func (g *sonyflakeGenerator) NextID() (int64, error) {
	id, err := g.sf.NextID()
	if err != nil {
		return 0, errors.Wrap(err, "failed to generate id")
	}
	return int64(id), nil
}
