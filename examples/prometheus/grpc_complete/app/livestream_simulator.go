package main

import (
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"
)

// GiftType 礼物类型
type GiftType struct {
	Type  string
	Name  string
	Value float64 // 价值（分）
}

var giftTypes = []GiftType{
	{"common", "玫瑰", 100},
	{"common", "棒棒糖", 200},
	{"common", "咖啡", 500},
	{"rare", "钻石", 1000},
	{"rare", "跑车", 5000},
	{"epic", "火箭", 10000},
	{"epic", "城堡", 50000},
	{"legendary", "嘉年华", 100000},
}

// LiveRoom 直播间
type LiveRoom struct {
	ID             string
	StreamerID     string
	StreamerName   string
	Title          string
	CurrentViewers int
	TotalViewers   int
	PeakViewers    map[string]int // period -> peak
	IsLive         bool
	StartTime      time.Time
	mu             sync.RWMutex
}

// StreamerInfo 主播信息
type StreamerInfo struct {
	ID                string
	Name              string
	CreatedAt         time.Time
	LastOnlineTime    time.Time
	LastStreamingTime time.Time
	IsOnline          bool
	IsStreaming       bool

	// 统计数据
	TotalOnlineDuration     time.Duration
	TotalStreamingDuration  time.Duration
	TodayStreamingDuration  time.Duration
	WeekStreamingDuration   time.Duration
	MonthStreamingDuration  time.Duration
	StreamSessionsCount     int
	StreamingStreakDays     int
	Last7DaysActiveDays     int
	Last30DaysActiveDays    int

	// 观众关系
	AttractedViewers       []string // 观众ID列表
	AttractedViewersActive int      // 仍活跃的观众数

	mu sync.RWMutex
}

// ViewerInfo 观众信息
type ViewerInfo struct {
	ID             string
	Name           string
	CreatedAt      time.Time
	LastActiveTime time.Time
	IsOnline       bool

	// 来源
	SourceStreamerID   string
	SourceStreamerName string

	// 统计数据
	TotalOnlineDuration    time.Duration
	TotalWatchDuration     time.Duration
	TodayWatchDuration     time.Duration
	WeekWatchDuration      time.Duration
	MonthWatchDuration     time.Duration
	RoomVisitsCount        int
	StreamersWatchedCount  int
	WatchStreakDays        int
	Last7DaysActiveDays    int
	Last30DaysActiveDays   int
	WatchedStreamers       map[string]bool // 观看过的主播

	mu sync.RWMutex
}

// Viewer 直播间观众（临时在线）
type Viewer struct {
	ID       string
	Name     string
	JoinTime time.Time
	RoomID   string
}

// LiveStreamSimulator 直播间模拟器
type LiveStreamSimulator struct {
	rooms          map[string]*LiveRoom
	viewers        map[string]*Viewer        // 临时在线观众
	streamers      map[string]*StreamerInfo  // 主播信息
	viewerInfos    map[string]*ViewerInfo    // 观众信息
	liveMetrics    *LiveStreamMetrics
	streamerMetrics *StreamerMetrics
	viewerMetrics   *ViewerMetrics
	mu             sync.RWMutex

	// 峰值追踪
	globalPeak map[string]int // period -> peak
	peakMu     sync.RWMutex
}

// NewLiveStreamSimulator 创建模拟器
func NewLiveStreamSimulator(liveMetrics *LiveStreamMetrics, streamerMetrics *StreamerMetrics, viewerMetrics *ViewerMetrics) *LiveStreamSimulator {
	return &LiveStreamSimulator{
		rooms:           make(map[string]*LiveRoom),
		viewers:         make(map[string]*Viewer),
		streamers:       make(map[string]*StreamerInfo),
		viewerInfos:     make(map[string]*ViewerInfo),
		liveMetrics:     liveMetrics,
		streamerMetrics: streamerMetrics,
		viewerMetrics:   viewerMetrics,
		globalPeak:      make(map[string]int),
	}
}

// Start 启动模拟器
func (s *LiveStreamSimulator) Start() {
	log.Println("🎮 启动直播间模拟器...")

	// 初始化直播间和主播信息
	s.initRooms()
	s.initStreamers()
	s.initViewers()

	// 启动模拟协程
	go s.simulateRoomActivity()
	go s.simulateViewerBehavior()
	go s.simulateGifts()
	go s.updateMetrics()
	go s.updatePeakMetrics()
	go s.updateStreamerMetrics()
	go s.updateViewerMetrics()

	log.Println("✅ 直播间模拟器启动完成")
	log.Println("📊 模拟内容:")
	log.Println("   - 15个直播间，动态开关播")
	log.Println("   - 观众随机进入/离开")
	log.Println("   - 8种礼物类型（普通、稀有、史诗、传说）")
	log.Println("   - 实时统计观众数、收入、峰值")
}

// initRooms 初始化直播间
func (s *LiveStreamSimulator) initRooms() {
	streamers := []struct {
		ID   string
		Name string
	}{
		{"1001", "小米直播"},
		{"1002", "游戏高手"},
		{"1003", "美食达人"},
		{"1004", "音乐天才"},
		{"1005", "舞蹈主播"},
		{"1006", "聊天室"},
		{"1007", "户外探险"},
		{"1008", "电竞选手"},
		{"1009", "知识分享"},
		{"1010", "绘画教学"},
		{"1011", "健身教练"},
		{"1012", "宠物乐园"},
		{"1013", "旅游vlog"},
		{"1014", "二次元"},
		{"1015", "手工制作"},
	}

	titles := []string{
		"今日首播！",
		"新人求关注",
		"感谢大家支持",
		"周年庆典",
		"PK大战",
		"才艺展示",
		"互动问答",
		"粉丝见面会",
		"特别活动",
		"日常直播",
	}

	for i, streamer := range streamers {
		roomID := fmt.Sprintf("room_%d", 10000+i)
		room := &LiveRoom{
			ID:             roomID,
			StreamerID:     streamer.ID,
			StreamerName:   streamer.Name,
			Title:          titles[rand.Intn(len(titles))],
			CurrentViewers: 0,
			TotalViewers:   0,
			PeakViewers:    make(map[string]int),
			IsLive:         rand.Float64() > 0.3, // 70% 概率开播
			StartTime:      time.Now().Add(-time.Duration(rand.Intn(120)) * time.Minute),
		}
		s.rooms[roomID] = room
	}
}

// simulateRoomActivity 模拟直播间开关播
func (s *LiveStreamSimulator) simulateRoomActivity() {
	ticker := time.NewTicker(45 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		for _, room := range s.rooms {
			room.mu.Lock()

			// 8% 概率切换开播状态
			if rand.Float64() < 0.08 {
				room.IsLive = !room.IsLive
				if room.IsLive {
					room.StartTime = time.Now()
					room.Title = []string{
						"今日首播！", "新人求关注", "感谢大家支持", "周年庆典",
						"PK大战", "才艺展示", "互动问答", "粉丝见面会",
					}[rand.Intn(8)]
					log.Printf("📺 [%s] %s 开播了！标题: %s\n", room.ID, room.StreamerName, room.Title)
				} else {
					log.Printf("⏹️  [%s] %s 下播了 (峰值观众: %d)\n", room.ID, room.StreamerName, room.PeakViewers["hourly"])
					room.CurrentViewers = 0
				}
			}

			room.mu.Unlock()
		}
		s.mu.Unlock()
	}
}

// simulateViewerBehavior 模拟观众行为
func (s *LiveStreamSimulator) simulateViewerBehavior() {
	ticker := time.NewTicker(1500 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.RLock()
		liveRooms := make([]*LiveRoom, 0)
		for _, room := range s.rooms {
			room.mu.RLock()
			if room.IsLive {
				liveRooms = append(liveRooms, room)
			}
			room.mu.RUnlock()
		}
		s.mu.RUnlock()

		if len(liveRooms) == 0 {
			continue
		}

		// 随机选择直播间
		room := liveRooms[rand.Intn(len(liveRooms))]

		// 60% 概率有观众进入，40% 概率有观众离开
		if rand.Float64() < 0.6 {
			// 观众进入
			count := rand.Intn(8) + 1 // 1-8 人
			for i := 0; i < count; i++ {
				viewerID := fmt.Sprintf("viewer_%d_%d", time.Now().UnixNano(), rand.Intn(100000))
				viewer := &Viewer{
					ID:       viewerID,
					Name:     fmt.Sprintf("用户%d", rand.Intn(10000)),
					JoinTime: time.Now(),
					RoomID:   room.ID,
				}

				s.mu.Lock()
				s.viewers[viewerID] = viewer
				s.mu.Unlock()

				room.mu.Lock()
				room.CurrentViewers++
				room.TotalViewers++
				s.liveMetrics.RecordViewerJoin(room.ID, room.StreamerID, room.StreamerName)
				room.mu.Unlock()
			}
		} else {
			// 观众离开
			room.mu.Lock()
			if room.CurrentViewers > 0 {
				count := rand.Intn(min(5, room.CurrentViewers)) + 1
				room.CurrentViewers -= count

				// 模拟观看时长
				for i := 0; i < count; i++ {
					duration := float64(rand.Intn(3600) + 60) // 1分钟到1小时
					s.liveMetrics.RecordViewerLeave(room.ID, room.StreamerID, duration)
				}
			}
			room.mu.Unlock()
		}
	}
}

// simulateGifts 模拟送礼行为
func (s *LiveStreamSimulator) simulateGifts() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.RLock()
		liveRooms := make([]*LiveRoom, 0)
		for _, room := range s.rooms {
			room.mu.RLock()
			if room.IsLive && room.CurrentViewers > 0 {
				liveRooms = append(liveRooms, room)
			}
			room.mu.RUnlock()
		}
		s.mu.RUnlock()

		if len(liveRooms) == 0 {
			continue
		}

		// 随机选择直播间送礼
		room := liveRooms[rand.Intn(len(liveRooms))]

		// 根据观众数量调整送礼概率
		room.mu.RLock()
		giftProbability := minFloat(float64(room.CurrentViewers)/500.0, 0.9)
		currentViewers := room.CurrentViewers
		room.mu.RUnlock()

		if rand.Float64() < giftProbability {
			// 选择礼物（普通礼物更常见）
			var gift GiftType
			r := rand.Float64()
			if r < 0.6 {
				// 60% 普通礼物
				commonGifts := []GiftType{giftTypes[0], giftTypes[1], giftTypes[2]}
				gift = commonGifts[rand.Intn(len(commonGifts))]
			} else if r < 0.85 {
				// 25% 稀有礼物
				rareGifts := []GiftType{giftTypes[3], giftTypes[4]}
				gift = rareGifts[rand.Intn(len(rareGifts))]
			} else if r < 0.98 {
				// 13% 史诗礼物
				epicGifts := []GiftType{giftTypes[5], giftTypes[6]}
				gift = epicGifts[rand.Intn(len(epicGifts))]
			} else {
				// 2% 传说礼物
				gift = giftTypes[7]
			}

			// 随机用户（固定用户池，模拟真实场景）
			userID := fmt.Sprintf("user_%d", rand.Intn(500))
			userName := fmt.Sprintf("用户%d", rand.Intn(500))

			room.mu.RLock()
			s.liveMetrics.RecordGift(room.ID, room.StreamerID, room.StreamerName, userID, userName, gift.Type, gift.Name, gift.Value)

			// 记录主播收到的礼物
			s.streamerMetrics.StreamerGiftsReceivedTotal.WithLabelValues(
				room.StreamerID, room.StreamerName, gift.Type, gift.Name,
			).Inc()

			// 记录观众送出的礼物
			s.viewerMetrics.ViewerGiftsSentTotal.WithLabelValues(
				userID, userName, gift.Type, gift.Name,
			).Inc()

			if gift.Value >= 10000 {
				log.Printf("💎 [%s] %s 收到 %s 的 %s！(%.2f元) [观众:%d]\n",
					room.ID, room.StreamerName, userName, gift.Name, gift.Value/100.0, currentViewers)
			}
			room.mu.RUnlock()
		}
	}
}

// updateMetrics 更新实时指标
func (s *LiveStreamSimulator) updateMetrics() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.RLock()

		totalViewers := 0
		activeRooms := 0
		activeStreamers := 0

		for _, room := range s.rooms {
			room.mu.RLock()

			if room.IsLive {
				activeRooms++
				activeStreamers++
				totalViewers += room.CurrentViewers

				// 更新直播间观众数
				s.liveMetrics.UpdateRoomViewers(
					room.ID,
					room.StreamerID,
					room.StreamerName,
					room.Title,
					float64(room.CurrentViewers),
				)

				// 更新直播间状态
				s.liveMetrics.UpdateRoomStatus(room.ID, room.StreamerID, room.StreamerName, 1)
			} else {
				s.liveMetrics.UpdateRoomStatus(room.ID, room.StreamerID, room.StreamerName, 0)
			}

			room.mu.RUnlock()
		}

		// 更新全局指标
		s.liveMetrics.TotalViewers.Set(float64(totalViewers))
		s.liveMetrics.ActiveRooms.Set(float64(activeRooms))
		s.liveMetrics.ActiveStreamers.Set(float64(activeStreamers))

		s.mu.RUnlock()
	}
}

// updatePeakMetrics 更新峰值指标
func (s *LiveStreamSimulator) updatePeakMetrics() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.RLock()

		globalViewers := 0

		for _, room := range s.rooms {
			room.mu.Lock()

			if room.IsLive {
				globalViewers += room.CurrentViewers

				// 更新各时段峰值
				periods := []string{"hourly", "daily", "weekly", "monthly"}
				for _, period := range periods {
					if room.CurrentViewers > room.PeakViewers[period] {
						room.PeakViewers[period] = room.CurrentViewers
						s.liveMetrics.UpdatePeakViewers(room.ID, room.StreamerID, float64(room.CurrentViewers), period)
					}
				}
			}

			room.mu.Unlock()
		}

		// 更新全局峰值
		s.peakMu.Lock()
		periods := []string{"hourly", "daily", "weekly", "monthly"}
		for _, period := range periods {
			if globalViewers > s.globalPeak[period] {
				s.globalPeak[period] = globalViewers
				s.liveMetrics.UpdateGlobalPeakViewers(float64(globalViewers), period)
			}
		}
		s.peakMu.Unlock()

		s.mu.RUnlock()
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// initStreamers 初始化主播信息
func (s *LiveStreamSimulator) initStreamers() {
	now := time.Now()

	for roomID, room := range s.rooms {
		// 为每个主播创建详细信息
		streamer := &StreamerInfo{
			ID:                  room.StreamerID,
			Name:                room.StreamerName,
			CreatedAt:           now.Add(-time.Duration(rand.Intn(365)) * 24 * time.Hour), // 1-365天前创建
			IsOnline:            room.IsLive,
			IsStreaming:         room.IsLive,
			LastOnlineTime:      now,
			LastStreamingTime:   now,
			AttractedViewers:    make([]string, 0),
			StreamingStreakDays: rand.Intn(30),
		}

		// 模拟历史数据
		streamer.TotalOnlineDuration = time.Duration(rand.Intn(1500)) * time.Hour
		streamer.TotalStreamingDuration = time.Duration(rand.Intn(1000)) * time.Hour
		streamer.StreamSessionsCount = rand.Intn(500)
		streamer.Last7DaysActiveDays = rand.Intn(7)
		streamer.Last30DaysActiveDays = rand.Intn(30)

		s.streamers[roomID] = streamer

		// 初始化 Counter 指标的历史值
		s.streamerMetrics.StreamerOnlineDuration.WithLabelValues(streamer.ID, streamer.Name).Add(streamer.TotalOnlineDuration.Seconds())
		s.streamerMetrics.StreamerStreamingDurationTotal.WithLabelValues(streamer.ID, streamer.Name).Add(streamer.TotalStreamingDuration.Seconds())
		s.streamerMetrics.StreamerStreamSessionsTotal.WithLabelValues(streamer.ID, streamer.Name).Add(float64(streamer.StreamSessionsCount))

		log.Printf("📺 初始化主播: %s (ID: %s)", streamer.Name, streamer.ID)
	}
}

// initViewers 初始化观众信息
func (s *LiveStreamSimulator) initViewers() {
	now := time.Now()
	streamerList := make([]*StreamerInfo, 0)
	for _, streamer := range s.streamers {
		streamerList = append(streamerList, streamer)
	}

	// 创建 500 个虚拟观众
	for i := 0; i < 500; i++ {
		viewerID := fmt.Sprintf("viewer_%d", i+1)
		viewerName := fmt.Sprintf("观众%d", i+1)

		// 随机选择一个主播作为来源
		sourceStreamer := streamerList[rand.Intn(len(streamerList))]

		viewer := &ViewerInfo{
			ID:                 viewerID,
			Name:               viewerName,
			CreatedAt:          now.Add(-time.Duration(rand.Intn(180)) * 24 * time.Hour), // 1-180天前创建
			IsOnline:           false,
			SourceStreamerID:   sourceStreamer.ID,
			SourceStreamerName: sourceStreamer.Name,
			LastActiveTime:     now.Add(-time.Duration(rand.Intn(48)) * time.Hour), // 最近48小时内活跃
			WatchedStreamers:   make(map[string]bool),
			WatchStreakDays:    rand.Intn(30),
		}

		// 模拟历史数据
		viewer.TotalOnlineDuration = time.Duration(rand.Intn(800)) * time.Hour
		viewer.TotalWatchDuration = time.Duration(rand.Intn(500)) * time.Hour
		viewer.RoomVisitsCount = rand.Intn(1000)
		viewer.StreamersWatchedCount = rand.Intn(len(streamerList))
		viewer.Last7DaysActiveDays = rand.Intn(7)
		viewer.Last30DaysActiveDays = rand.Intn(30)

		s.viewerInfos[viewerID] = viewer

		// 初始化 Counter 指标的历史值
		s.viewerMetrics.ViewerOnlineDuration.WithLabelValues(viewer.ID, viewer.Name).Add(viewer.TotalOnlineDuration.Seconds())
		s.viewerMetrics.ViewerWatchDurationTotal.WithLabelValues(viewer.ID, viewer.Name).Add(viewer.TotalWatchDuration.Seconds())
		s.viewerMetrics.ViewerRoomVisitsTotal.WithLabelValues(viewer.ID, viewer.Name).Add(float64(viewer.RoomVisitsCount))
		s.viewerMetrics.ViewerStreamersWatchedTotal.WithLabelValues(viewer.ID, viewer.Name).Add(float64(viewer.StreamersWatchedCount))

		// 更新主播的吸引观众列表
		sourceStreamer.mu.Lock()
		sourceStreamer.AttractedViewers = append(sourceStreamer.AttractedViewers, viewerID)
		sourceStreamer.mu.Unlock()

		// 增加注册观众计数
		s.viewerMetrics.ViewersRegisteredTotal.Inc()
	}

	log.Printf("👥 初始化了 500 个观众")
}

// updateStreamerMetrics 更新主播指标
func (s *LiveStreamSimulator) updateStreamerMetrics() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.RLock()

		onlineCount := 0
		streamingCount := 0
		active7Days := 0
		active30Days := 0
		now := time.Now()
		hour := now.Hour()
		dayType := "weekday"
		if now.Weekday() == time.Saturday || now.Weekday() == time.Sunday {
			dayType = "weekend"
		}

		for _, streamer := range s.streamers {
			streamer.mu.RLock()

			// 更新单个主播指标
			status := 0.0
			if streamer.IsStreaming {
				status = 2.0
				streamingCount++
				onlineCount++
			} else if streamer.IsOnline {
				status = 1.0
				onlineCount++
			}

			s.streamerMetrics.StreamerOnlineStatus.WithLabelValues(streamer.ID, streamer.Name).Set(status)
			s.streamerMetrics.StreamerLastOnlineTime.WithLabelValues(streamer.ID, streamer.Name).Set(float64(streamer.LastOnlineTime.Unix()))

			// Counter 类型指标 - 使用 Add(0) 确保可被抓取
			s.streamerMetrics.StreamerOnlineDuration.WithLabelValues(streamer.ID, streamer.Name).Add(0)
			s.streamerMetrics.StreamerStreamingDurationTotal.WithLabelValues(streamer.ID, streamer.Name).Add(0)
			s.streamerMetrics.StreamerStreamSessionsTotal.WithLabelValues(streamer.ID, streamer.Name).Add(0)

			// 直播时长
			s.streamerMetrics.StreamerStreamingDurationToday.WithLabelValues(streamer.ID, streamer.Name).Set(streamer.TodayStreamingDuration.Seconds())
			s.streamerMetrics.StreamerStreamingDurationWeek.WithLabelValues(streamer.ID, streamer.Name).Set(streamer.WeekStreamingDuration.Seconds())
			s.streamerMetrics.StreamerStreamingDurationMonth.WithLabelValues(streamer.ID, streamer.Name).Set(streamer.MonthStreamingDuration.Seconds())

			// 计算平均每天直播时长
			daysSinceCreated := now.Sub(streamer.CreatedAt).Hours() / 24
			if daysSinceCreated > 0 {
				avgDaily := streamer.TotalStreamingDuration.Seconds() / daysSinceCreated
				s.streamerMetrics.StreamerStreamingDurationDailyAvg.WithLabelValues(streamer.ID, streamer.Name).Set(avgDaily)
			}

			s.streamerMetrics.StreamerStreamingStreakDays.WithLabelValues(streamer.ID, streamer.Name).Set(float64(streamer.StreamingStreakDays))

			// 活跃度
			s.streamerMetrics.StreamerActiveDaysLast7Days.WithLabelValues(streamer.ID, streamer.Name).Set(float64(streamer.Last7DaysActiveDays))
			s.streamerMetrics.StreamerActiveDaysLast30Days.WithLabelValues(streamer.ID, streamer.Name).Set(float64(streamer.Last30DaysActiveDays))

			// 观众关系
			activeViewers := 0
			for _, viewerID := range streamer.AttractedViewers {
				if vInfo, ok := s.viewerInfos[viewerID]; ok {
					// 7天内活跃算作活跃观众
					if now.Sub(vInfo.LastActiveTime) < 7*24*time.Hour {
						activeViewers++
					}
				}
			}
			s.streamerMetrics.StreamerAttractedViewersTotal.WithLabelValues(streamer.ID, streamer.Name).Set(float64(len(streamer.AttractedViewers)))
			s.streamerMetrics.StreamerAttractedViewersActive.WithLabelValues(streamer.ID, streamer.Name).Set(float64(activeViewers))

			// 统计活跃主播
			if streamer.Last7DaysActiveDays > 0 {
				active7Days++
			}
			if streamer.Last30DaysActiveDays > 0 {
				active30Days++
			}

			streamer.mu.RUnlock()
		}

		// 更新全局主播统计
		s.streamerMetrics.StreamersOnlineTotal.Set(float64(onlineCount))
		s.streamerMetrics.StreamersStreamingTotal.Set(float64(streamingCount))
		s.streamerMetrics.StreamersOnlineByHour.WithLabelValues(fmt.Sprintf("%d", hour)).Set(float64(onlineCount))
		s.streamerMetrics.StreamersOnlineByDayType.WithLabelValues(dayType).Set(float64(onlineCount))
		s.streamerMetrics.StreamersActiveLast7Days.Set(float64(active7Days))
		s.streamerMetrics.StreamersActiveLast30Days.Set(float64(active30Days))

		s.mu.RUnlock()
	}
}

// updateViewerMetrics 更新观众指标
func (s *LiveStreamSimulator) updateViewerMetrics() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.RLock()

		onlineCount := 0
		active7Days := 0
		active30Days := 0
		churned7Days := 0
		churned30Days := 0
		now := time.Now()
		hour := now.Hour()
		dayType := "weekday"
		if now.Weekday() == time.Saturday || now.Weekday() == time.Sunday {
			dayType = "weekend"
		}

		for _, viewer := range s.viewerInfos {
			viewer.mu.RLock()

			// 更新单个观众指标
			status := 0.0
			if viewer.IsOnline {
				status = 1.0
				onlineCount++
			}

			s.viewerMetrics.ViewerOnlineStatus.WithLabelValues(viewer.ID, viewer.Name).Set(status)
			s.viewerMetrics.ViewerLastActiveTime.WithLabelValues(viewer.ID, viewer.Name).Set(float64(viewer.LastActiveTime.Unix()))

			// Counter 类型指标 - 使用 Add(0) 确保可被抓取
			s.viewerMetrics.ViewerOnlineDuration.WithLabelValues(viewer.ID, viewer.Name).Add(0)
			s.viewerMetrics.ViewerWatchDurationTotal.WithLabelValues(viewer.ID, viewer.Name).Add(0)
			s.viewerMetrics.ViewerRoomVisitsTotal.WithLabelValues(viewer.ID, viewer.Name).Add(0)
			s.viewerMetrics.ViewerStreamersWatchedTotal.WithLabelValues(viewer.ID, viewer.Name).Add(0)

			// 观看时长
			s.viewerMetrics.ViewerWatchDurationToday.WithLabelValues(viewer.ID, viewer.Name).Set(viewer.TodayWatchDuration.Seconds())
			s.viewerMetrics.ViewerWatchDurationWeek.WithLabelValues(viewer.ID, viewer.Name).Set(viewer.WeekWatchDuration.Seconds())
			s.viewerMetrics.ViewerWatchDurationMonth.WithLabelValues(viewer.ID, viewer.Name).Set(viewer.MonthWatchDuration.Seconds())

			// 计算平均每天观看时长
			daysSinceCreated := now.Sub(viewer.CreatedAt).Hours() / 24
			if daysSinceCreated > 0 {
				avgDaily := viewer.TotalWatchDuration.Seconds() / daysSinceCreated
				s.viewerMetrics.ViewerWatchDurationDailyAvg.WithLabelValues(viewer.ID, viewer.Name).Set(avgDaily)
			}

			s.viewerMetrics.ViewerWatchStreakDays.WithLabelValues(viewer.ID, viewer.Name).Set(float64(viewer.WatchStreakDays))

			// 活跃度
			s.viewerMetrics.ViewerActiveDaysLast7Days.WithLabelValues(viewer.ID, viewer.Name).Set(float64(viewer.Last7DaysActiveDays))
			s.viewerMetrics.ViewerActiveDaysLast30Days.WithLabelValues(viewer.ID, viewer.Name).Set(float64(viewer.Last30DaysActiveDays))

			// 来源主播
			s.viewerMetrics.ViewerSourceStreamer.WithLabelValues(
				viewer.ID, viewer.Name,
				viewer.SourceStreamerID, viewer.SourceStreamerName,
			).Set(1)

			// 统计活跃和流失
			hoursSinceActive := now.Sub(viewer.LastActiveTime).Hours()
			if viewer.Last7DaysActiveDays > 0 {
				active7Days++
			}
			if viewer.Last30DaysActiveDays > 0 {
				active30Days++
			}
			if hoursSinceActive > 7*24 {
				churned7Days++
			}
			if hoursSinceActive > 30*24 {
				churned30Days++
			}

			viewer.mu.RUnlock()
		}

		// 更新全局观众统计
		s.viewerMetrics.ViewersOnlineByHour.WithLabelValues(fmt.Sprintf("%d", hour)).Set(float64(onlineCount))
		s.viewerMetrics.ViewersOnlineByDayType.WithLabelValues(dayType).Set(float64(onlineCount))
		s.viewerMetrics.ViewersActiveLast7Days.Set(float64(active7Days))
		s.viewerMetrics.ViewersActiveLast30Days.Set(float64(active30Days))
		s.viewerMetrics.ViewersChurned7Days.Set(float64(churned7Days))
		s.viewerMetrics.ViewersChurned30Days.Set(float64(churned30Days))

		// 模拟留存率（简化计算）
		if len(s.viewerInfos) > 0 {
			retention1d := float64(active7Days) / float64(len(s.viewerInfos)) * 100
			retention7d := float64(active7Days) / float64(len(s.viewerInfos)) * 100
			retention30d := float64(active30Days) / float64(len(s.viewerInfos)) * 100

			s.viewerMetrics.ViewerRetentionRate.WithLabelValues("1d").Set(retention1d)
			s.viewerMetrics.ViewerRetentionRate.WithLabelValues("7d").Set(retention7d)
			s.viewerMetrics.ViewerRetentionRate.WithLabelValues("30d").Set(retention30d)
		}

		s.mu.RUnlock()
	}
}
