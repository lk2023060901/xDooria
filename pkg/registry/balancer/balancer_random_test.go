package balancer

import (
	"testing"

	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/resolver"
)

func TestRandomBuilder_Build(t *testing.T) {
	builder := &randomBuilder{}

	// 测试空连接
	picker := builder.Build(base.PickerBuildInfo{
		ReadySCs: map[balancer.SubConn]base.SubConnInfo{},
	})

	_, err := picker.Pick(balancer.PickInfo{})
	if err != balancer.ErrNoSubConnAvailable {
		t.Errorf("expected ErrNoSubConnAvailable, got %v", err)
	}
}

func TestRandomPicker_Pick(t *testing.T) {
	// 创建模拟的 SubConn
	sc1 := &mockSubConn{id: "sc1"}
	sc2 := &mockSubConn{id: "sc2"}
	sc3 := &mockSubConn{id: "sc3"}

	builder := &randomBuilder{}

	picker := builder.Build(base.PickerBuildInfo{
		ReadySCs: map[balancer.SubConn]base.SubConnInfo{
			sc1: {Address: resolver.Address{Addr: "127.0.0.1:8001"}},
			sc2: {Address: resolver.Address{Addr: "127.0.0.1:8002"}},
			sc3: {Address: resolver.Address{Addr: "127.0.0.1:8003"}},
		},
	})

	// 统计选择次数
	counts := make(map[string]int)
	total := 1000

	for i := 0; i < total; i++ {
		result, err := picker.Pick(balancer.PickInfo{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		sc := result.SubConn.(*mockSubConn)
		counts[sc.id]++
	}

	// 验证每个节点都被选中过
	if counts["sc1"] == 0 {
		t.Error("sc1 was never selected")
	}
	if counts["sc2"] == 0 {
		t.Error("sc2 was never selected")
	}
	if counts["sc3"] == 0 {
		t.Error("sc3 was never selected")
	}

	// 验证随机分布（每个节点应该接近 1/3）
	expected := total / 3
	tolerance := total / 5 // ±20%

	if abs(counts["sc1"]-expected) > tolerance {
		t.Errorf("sc1 count: got %d, expected ~%d (±%d)", counts["sc1"], expected, tolerance)
	}
	if abs(counts["sc2"]-expected) > tolerance {
		t.Errorf("sc2 count: got %d, expected ~%d (±%d)", counts["sc2"], expected, tolerance)
	}
	if abs(counts["sc3"]-expected) > tolerance {
		t.Errorf("sc3 count: got %d, expected ~%d (±%d)", counts["sc3"], expected, tolerance)
	}
}

func TestRandomPicker_SingleSubConn(t *testing.T) {
	// 测试只有一个 SubConn 的情况
	sc1 := &mockSubConn{id: "sc1"}

	builder := &randomBuilder{}

	picker := builder.Build(base.PickerBuildInfo{
		ReadySCs: map[balancer.SubConn]base.SubConnInfo{
			sc1: {Address: resolver.Address{Addr: "127.0.0.1:8001"}},
		},
	})

	// 多次选择，应该总是返回同一个 SubConn
	for i := 0; i < 10; i++ {
		result, err := picker.Pick(balancer.PickInfo{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		sc := result.SubConn.(*mockSubConn)
		if sc.id != "sc1" {
			t.Errorf("expected sc1, got %s", sc.id)
		}
	}
}

func TestRandomPicker_Randomness(t *testing.T) {
	// 验证真的是随机的，而不是轮询
	sc1 := &mockSubConn{id: "sc1"}
	sc2 := &mockSubConn{id: "sc2"}

	builder := &randomBuilder{}

	picker := builder.Build(base.PickerBuildInfo{
		ReadySCs: map[balancer.SubConn]base.SubConnInfo{
			sc1: {Address: resolver.Address{Addr: "127.0.0.1:8001"}},
			sc2: {Address: resolver.Address{Addr: "127.0.0.1:8002"}},
		},
	})

	// 获取前20次选择的序列
	var sequence []string
	for i := 0; i < 20; i++ {
		result, err := picker.Pick(balancer.PickInfo{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		sc := result.SubConn.(*mockSubConn)
		sequence = append(sequence, sc.id)
	}

	// 验证序列不是简单的 sc1, sc2, sc1, sc2... 模式
	// 检查是否存在连续相同的选择（随机时应该会出现）
	hasConsecutive := false
	for i := 1; i < len(sequence); i++ {
		if sequence[i] == sequence[i-1] {
			hasConsecutive = true
			break
		}
	}

	if !hasConsecutive {
		t.Errorf("sequence appears to be round-robin, not random: %v", sequence)
	}
}

func BenchmarkRandomPicker_Pick(b *testing.B) {
	sc1 := &mockSubConn{id: "sc1"}
	sc2 := &mockSubConn{id: "sc2"}
	sc3 := &mockSubConn{id: "sc3"}

	builder := &randomBuilder{}

	picker := builder.Build(base.PickerBuildInfo{
		ReadySCs: map[balancer.SubConn]base.SubConnInfo{
			sc1: {Address: resolver.Address{Addr: "127.0.0.1:8001"}},
			sc2: {Address: resolver.Address{Addr: "127.0.0.1:8002"}},
			sc3: {Address: resolver.Address{Addr: "127.0.0.1:8003"}},
		},
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = picker.Pick(balancer.PickInfo{})
	}
}

func BenchmarkWeightedRoundRobinPicker_Pick(b *testing.B) {
	sc1 := &mockSubConn{id: "sc1"}
	sc2 := &mockSubConn{id: "sc2"}
	sc3 := &mockSubConn{id: "sc3"}

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

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = picker.Pick(balancer.PickInfo{})
	}
}
