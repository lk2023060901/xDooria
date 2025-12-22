package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/lk2023060901/xdooria/pkg/etcd"
)

func main() {
	fmt.Println("=== etcd Election 选举功能示例 ===")

	client, err := etcd.New(&etcd.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	election := client.Election()
	ctx := context.Background()

	// 1. 基本选举
	fmt.Println("【1. 基本选举】")
	testBasicElection(ctx, election)

	// 2. 查询 Leader
	fmt.Println("\n【2. 查询 Leader】")
	testGetLeader(ctx, election)

	// 3. 观察 Leader 变化
	fmt.Println("\n【3. 观察 Leader 变化】")
	testObserveLeader(ctx, election)

	// 4. Leader 辞职
	fmt.Println("\n【4. Leader 辞职】")
	testResign(ctx, election)

	// 5. 多节点竞选
	fmt.Println("\n【5. 多节点竞选】")
	testMultiNodeElection(ctx, election)

	fmt.Println("\n✅ 示例完成")
}

func testBasicElection(ctx context.Context, election *etcd.Election) {
	prefix := "/election/basic"

	// 创建选举器
	elector, err := election.NewElector(prefix, etcd.WithElectionTTL(10))
	if err != nil {
		log.Printf("  ❌ 创建选举器失败: %v", err)
		return
	}
	defer elector.Close()

	fmt.Printf("  ✓ 创建选举器: %s (ttl=10s)\n", prefix)

	// 参与竞选
	value := "node-1"
	fmt.Printf("  %s 开始竞选...\n", value)

	go func() {
		if err := elector.Campaign(ctx, value); err != nil {
			log.Printf("  ❌ 竞选失败: %v", err)
		}
	}()

	// 等待选举结果
	time.Sleep(1 * time.Second)

	// 检查是否是 Leader
	if isLeader, err := elector.IsLeader(ctx); err != nil {
		log.Printf("  ❌ 检查 Leader 失败: %v", err)
	} else if isLeader {
		fmt.Printf("  ✓ %s 成为 Leader\n", value)
	} else {
		fmt.Printf("  ✗ %s 不是 Leader\n", value)
	}

	// 辞职
	time.Sleep(1 * time.Second)
	if err := elector.Resign(ctx); err != nil {
		log.Printf("  ❌ 辞职失败: %v", err)
	} else {
		fmt.Println("  ✓ Leader 已辞职")
	}
}

func testGetLeader(ctx context.Context, election *etcd.Election) {
	prefix := "/election/getleader"

	// 创建选举器1并竞选
	elector1, err := election.NewElector(prefix, etcd.WithElectionTTL(10))
	if err != nil {
		log.Printf("  ❌ 创建选举器1失败: %v", err)
		return
	}
	defer elector1.Close()

	go func() {
		elector1.Campaign(ctx, "node-alpha")
	}()

	time.Sleep(1 * time.Second)

	// 创建选举器2查询 Leader
	elector2, err := election.NewElector(prefix, etcd.WithElectionTTL(10))
	if err != nil {
		log.Printf("  ❌ 创建选举器2失败: %v", err)
		return
	}
	defer elector2.Close()

	leader, err := elector2.Leader(ctx)
	if err != nil {
		log.Printf("  ❌ 获取 Leader 失败: %v", err)
		return
	}

	fmt.Printf("  ✓ 当前 Leader: %s (revision=%d)\n", string(leader.Value), leader.Revision)

	// 清理
	elector1.Resign(ctx)
}

func testObserveLeader(ctx context.Context, election *etcd.Election) {
	prefix := "/election/observe"

	// 创建观察者
	elector, err := election.NewElector(prefix, etcd.WithElectionTTL(10))
	if err != nil {
		log.Printf("  ❌ 创建选举器失败: %v", err)
		return
	}
	defer elector.Close()

	fmt.Println("  启动 Leader 观察...")

	// 观察 Leader 变化
	observeCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	go func() {
		if err := elector.Observe(observeCtx, func(value *etcd.ElectionValue) {
			fmt.Printf("  [观察] Leader 变化: %s (revision=%d)\n", string(value.Value), value.Revision)
		}); err != nil && err != context.DeadlineExceeded {
			log.Printf("  ❌ 观察失败: %v", err)
		}
	}()

	time.Sleep(500 * time.Millisecond)

	// 模拟 Leader 变化
	elector1, _ := election.NewElector(prefix, etcd.WithElectionTTL(5))
	go func() {
		elector1.Campaign(ctx, "leader-1")
	}()

	time.Sleep(2 * time.Second)
	elector1.Resign(ctx)
	elector1.Close()

	time.Sleep(1 * time.Second)

	elector2, _ := election.NewElector(prefix, etcd.WithElectionTTL(5))
	go func() {
		elector2.Campaign(ctx, "leader-2")
	}()

	time.Sleep(2 * time.Second)
	elector2.Resign(ctx)
	elector2.Close()

	time.Sleep(1 * time.Second)
	fmt.Println("  ✓ 观察完成")
}

func testResign(ctx context.Context, election *etcd.Election) {
	prefix := "/election/resign"

	elector, err := election.NewElector(prefix, etcd.WithElectionTTL(10))
	if err != nil {
		log.Printf("  ❌ 创建选举器失败: %v", err)
		return
	}
	defer elector.Close()

	// 竞选
	go func() {
		if err := elector.Campaign(ctx, "node-resign"); err != nil {
			log.Printf("  ❌ 竞选失败: %v", err)
		}
	}()

	time.Sleep(1 * time.Second)

	if isLeader, err := elector.IsLeader(ctx); err != nil {
		log.Printf("  ❌ 检查 Leader 失败: %v", err)
	} else if isLeader {
		fmt.Println("  ✓ 成为 Leader")

		// 主动辞职
		fmt.Println("  Leader 主动辞职...")
		if err := elector.Resign(ctx); err != nil {
			log.Printf("  ❌ 辞职失败: %v", err)
			return
		}
		fmt.Println("  ✓ 辞职成功")

		time.Sleep(500 * time.Millisecond)

		if isLeader, err := elector.IsLeader(ctx); err != nil {
			log.Printf("  ❌ 检查 Leader 失败: %v", err)
		} else if !isLeader {
			fmt.Println("  ✓ 已不再是 Leader")
		}
	}
}

func testMultiNodeElection(ctx context.Context, election *etcd.Election) {
	prefix := "/election/multinode"
	nodes := 5

	var wg sync.WaitGroup
	var mu sync.Mutex
	leaderCount := 0

	fmt.Printf("  启动 %d 个节点参与竞选...\n", nodes)

	for i := 0; i < nodes; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			nodeName := fmt.Sprintf("node-%d", id)

			elector, err := election.NewElector(prefix, etcd.WithElectionTTL(10))
			if err != nil {
				log.Printf("  ❌ [%s] 创建选举器失败: %v", nodeName, err)
				return
			}
			defer elector.Close()

			// 参与竞选（阻塞直到成为 Leader）
			campaignCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			go func() {
				if err := elector.Campaign(campaignCtx, nodeName); err != nil && err != context.DeadlineExceeded {
					mu.Lock()
					fmt.Printf("  ✗ [%s] 竞选失败: %v\n", nodeName, err)
					mu.Unlock()
				}
			}()

			// 等待一小段时间检查
			time.Sleep(200 * time.Millisecond)

			if isLeader, err := elector.IsLeader(campaignCtx); err != nil {
				// 忽略检查错误
			} else if isLeader {
				mu.Lock()
				leaderCount++
				fmt.Printf("  ✓ [%s] 成为 Leader\n", nodeName)
				mu.Unlock()

				// Leader 工作一段时间后辞职
				time.Sleep(1 * time.Second)
				elector.Resign(campaignCtx)

				mu.Lock()
				fmt.Printf("  ✓ [%s] Leader 辞职\n", nodeName)
				mu.Unlock()
			} else {
				mu.Lock()
				fmt.Printf("  - [%s] 等待成为 Leader...\n", nodeName)
				mu.Unlock()
			}

			time.Sleep(time.Duration(id*200) * time.Millisecond)
		}(i)
	}

	wg.Wait()
	fmt.Printf("  ✓ 竞选完成，共产生 %d 个 Leader\n", leaderCount)
}
