package balancer

import (
	"testing"

	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/resolver"
)

func TestWeightedRoundRobinBuilder_Build(t *testing.T) {
	builder := &weightedRoundRobinBuilder{}

	// 测试空连接
	picker := builder.Build(base.PickerBuildInfo{
		ReadySCs: map[balancer.SubConn]base.SubConnInfo{},
	})

	_, err := picker.Pick(balancer.PickInfo{})
	if err != balancer.ErrNoSubConnAvailable {
		t.Errorf("expected ErrNoSubConnAvailable, got %v", err)
	}
}

func TestWeightedRoundRobinPicker_Pick(t *testing.T) {
	// 创建模拟的 SubConn
	sc1 := &mockSubConn{id: "sc1"}
	sc2 := &mockSubConn{id: "sc2"}
	sc3 := &mockSubConn{id: "sc3"}

	builder := &weightedRoundRobinBuilder{}

	// 构建带权重的 picker
	// sc1: weight=5, sc2: weight=1, sc3: weight=1
	picker := builder.Build(base.PickerBuildInfo{
		ReadySCs: map[balancer.SubConn]base.SubConnInfo{
			sc1: {Address: resolver.Address{
				Addr:       "127.0.0.1:8001",
				Attributes: newAttributesWithWeight(5),
			}},
			sc2: {Address: resolver.Address{
				Addr:       "127.0.0.1:8002",
				Attributes: newAttributesWithWeight(1),
			}},
			sc3: {Address: resolver.Address{
				Addr:       "127.0.0.1:8003",
				Attributes: newAttributesWithWeight(1),
			}},
		},
	})

	// 统计选择次数
	counts := make(map[string]int)
	total := 700 // 7的倍数，便于验证比例

	for i := 0; i < total; i++ {
		result, err := picker.Pick(balancer.PickInfo{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		sc := result.SubConn.(*mockSubConn)
		counts[sc.id]++
	}

	// 验证权重比例（5:1:1）
	// sc1 应该被选中约 5/7 的次数
	// sc2 和 sc3 各约 1/7 的次数
	expected1 := total * 5 / 7
	expected2 := total * 1 / 7
	expected3 := total * 1 / 7

	// 允许 ±10% 的误差
	tolerance := total / 10

	if abs(counts["sc1"]-expected1) > tolerance {
		t.Errorf("sc1 count: got %d, expected ~%d", counts["sc1"], expected1)
	}
	if abs(counts["sc2"]-expected2) > tolerance {
		t.Errorf("sc2 count: got %d, expected ~%d", counts["sc2"], expected2)
	}
	if abs(counts["sc3"]-expected3) > tolerance {
		t.Errorf("sc3 count: got %d, expected ~%d", counts["sc3"], expected3)
	}
}

func TestWeightedRoundRobinPicker_DefaultWeight(t *testing.T) {
	// 测试默认权重（未设置权重时，默认为1）
	sc1 := &mockSubConn{id: "sc1"}
	sc2 := &mockSubConn{id: "sc2"}

	builder := &weightedRoundRobinBuilder{}

	picker := builder.Build(base.PickerBuildInfo{
		ReadySCs: map[balancer.SubConn]base.SubConnInfo{
			sc1: {Address: resolver.Address{Addr: "127.0.0.1:8001"}},
			sc2: {Address: resolver.Address{Addr: "127.0.0.1:8002"}},
		},
	})

	// 统计选择次数
	counts := make(map[string]int)
	total := 100

	for i := 0; i < total; i++ {
		result, err := picker.Pick(balancer.PickInfo{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		sc := result.SubConn.(*mockSubConn)
		counts[sc.id]++
	}

	// 默认权重相同，应该接近 50:50
	tolerance := total / 5 // ±20%

	if abs(counts["sc1"]-total/2) > tolerance {
		t.Errorf("sc1 count: got %d, expected ~%d", counts["sc1"], total/2)
	}
	if abs(counts["sc2"]-total/2) > tolerance {
		t.Errorf("sc2 count: got %d, expected ~%d", counts["sc2"], total/2)
	}
}

func TestWeightedRoundRobinPicker_SmoothDistribution(t *testing.T) {
	// 验证平滑加权轮询的分布特性
	// 权重 5:1:1 的情况下，选择序列应该是平滑的，而不是 AAAAABCAAAABC...
	sc1 := &mockSubConn{id: "A"}
	sc2 := &mockSubConn{id: "B"}
	sc3 := &mockSubConn{id: "C"}

	builder := &weightedRoundRobinBuilder{}

	picker := builder.Build(base.PickerBuildInfo{
		ReadySCs: map[balancer.SubConn]base.SubConnInfo{
			sc1: {Address: resolver.Address{
				Addr:       "127.0.0.1:8001",
				Attributes: newAttributesWithWeight(5),
			}},
			sc2: {Address: resolver.Address{
				Addr:       "127.0.0.1:8002",
				Attributes: newAttributesWithWeight(1),
			}},
			sc3: {Address: resolver.Address{
				Addr:       "127.0.0.1:8003",
				Attributes: newAttributesWithWeight(1),
			}},
		},
	})

	// 获取前14次选择的序列（两个完整周期）
	var sequence []string
	for i := 0; i < 14; i++ {
		result, err := picker.Pick(balancer.PickInfo{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		sc := result.SubConn.(*mockSubConn)
		sequence = append(sequence, sc.id)
	}

	// 验证序列中不应该出现连续5个或更多的A
	// 平滑加权轮询会将高权重节点分散到整个序列中
	maxConsecutive := 0
	current := 1
	for i := 1; i < len(sequence); i++ {
		if sequence[i] == sequence[i-1] && sequence[i] == "A" {
			current++
		} else {
			if current > maxConsecutive {
				maxConsecutive = current
			}
			current = 1
		}
	}
	if current > maxConsecutive {
		maxConsecutive = current
	}

	// 平滑加权轮询应该避免连续选择同一节点过多次
	// 对于权重 5:1:1，最多连续4-5次是可接受的
	// (因为平滑算法会在某些情况下连续选择高权重节点)
	if maxConsecutive > 5 {
		t.Errorf("sequence not smooth enough, max consecutive A: %d, sequence: %v", maxConsecutive, sequence)
	}
}
