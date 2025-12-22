package main

import (
	"github.com/prometheus/client_golang/prometheus"
)

// LiveStreamMetrics 直播间业务指标
type LiveStreamMetrics struct {
	// 实时在线指标
	RoomViewers      *prometheus.GaugeVec // 直播间当前观众数
	RoomStatus       *prometheus.GaugeVec // 直播间状态 (0=关闭, 1=开播)
	TotalViewers     prometheus.Gauge     // 全局总观众数
	ActiveRooms      prometheus.Gauge     // 活跃直播间数量
	ActiveStreamers  prometheus.Gauge     // 在线主播数量

	// 累计统计指标
	RoomViewersTotal *prometheus.CounterVec // 累计观看次数
	RoomGiftsTotal   *prometheus.CounterVec // 累计礼物数量
	RoomRevenueTotal *prometheus.CounterVec // 累计收入（分）
	UserGiftsTotal   *prometheus.CounterVec // 用户累计送礼金额

	// 礼物详细指标
	GiftCount   *prometheus.CounterVec // 礼物类型计数
	GiftRevenue *prometheus.CounterVec // 礼物类型收入

	// 观众行为指标
	ViewerDuration   *prometheus.HistogramVec // 观众观看时长分布
	ViewerJoinTotal  *prometheus.CounterVec   // 观众进入次数
	ViewerLeaveTotal *prometheus.CounterVec   // 观众离开次数

	// 峰值指标
	RoomPeakViewers   *prometheus.GaugeVec // 直播间峰值观众数
	GlobalPeakViewers *prometheus.GaugeVec // 全局峰值观众数
}

// NewLiveStreamMetrics 创建直播间业务指标
func NewLiveStreamMetrics(reg prometheus.Registerer) *LiveStreamMetrics {
	metrics := &LiveStreamMetrics{
		RoomViewers: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "live_room_viewers",
				Help: "直播间当前观众数",
			},
			[]string{"room_id", "streamer_id", "streamer_name", "room_title"},
		),
		RoomStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "live_room_status",
				Help: "直播间状态 (0=关闭, 1=开播)",
			},
			[]string{"room_id", "streamer_id", "streamer_name"},
		),
		TotalViewers: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "live_total_viewers",
				Help: "全局总观众数",
			},
		),
		ActiveRooms: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "live_active_rooms",
				Help: "活跃直播间数量",
			},
		),
		ActiveStreamers: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "live_active_streamers",
				Help: "在线主播数量",
			},
		),
		RoomViewersTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "live_room_viewers_total",
				Help: "直播间累计观看次数",
			},
			[]string{"room_id", "streamer_id", "streamer_name"},
		),
		RoomGiftsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "live_room_gifts_total",
				Help: "直播间累计礼物数量",
			},
			[]string{"room_id", "streamer_id"},
		),
		RoomRevenueTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "live_room_revenue_total",
				Help: "直播间累计收入（分）",
			},
			[]string{"room_id", "streamer_id", "streamer_name"},
		),
		UserGiftsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "live_user_gifts_total",
				Help: "用户累计送礼金额（分）",
			},
			[]string{"user_id", "user_name", "room_id"},
		),
		GiftCount: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "live_gift_count_total",
				Help: "礼物类型计数",
			},
			[]string{"gift_type", "gift_name", "room_id"},
		),
		GiftRevenue: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "live_gift_revenue_total",
				Help: "礼物类型收入（分）",
			},
			[]string{"gift_type", "gift_name", "room_id"},
		),
		ViewerDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "live_viewer_duration_seconds",
				Help:    "观众观看时长分布（秒）",
				Buckets: []float64{60, 300, 600, 1800, 3600, 7200, 14400}, // 1分钟到4小时
			},
			[]string{"room_id"},
		),
		ViewerJoinTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "live_viewer_join_total",
				Help: "观众进入直播间次数",
			},
			[]string{"room_id", "streamer_id"},
		),
		ViewerLeaveTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "live_viewer_leave_total",
				Help: "观众离开直播间次数",
			},
			[]string{"room_id", "streamer_id"},
		),
		RoomPeakViewers: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "live_room_peak_viewers",
				Help: "直播间峰值观众数",
			},
			[]string{"room_id", "streamer_id", "period"}, // period: hourly, daily, weekly, monthly
		),
		GlobalPeakViewers: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "live_global_peak_viewers",
				Help: "全局峰值观众数",
			},
			[]string{"period"},
		),
	}

	// 注册所有指标
	reg.MustRegister(
		metrics.RoomViewers,
		metrics.RoomStatus,
		metrics.TotalViewers,
		metrics.ActiveRooms,
		metrics.ActiveStreamers,
		metrics.RoomViewersTotal,
		metrics.RoomGiftsTotal,
		metrics.RoomRevenueTotal,
		metrics.UserGiftsTotal,
		metrics.GiftCount,
		metrics.GiftRevenue,
		metrics.ViewerDuration,
		metrics.ViewerJoinTotal,
		metrics.ViewerLeaveTotal,
		metrics.RoomPeakViewers,
		metrics.GlobalPeakViewers,
	)

	return metrics
}

// RecordViewerJoin 记录观众进入
func (m *LiveStreamMetrics) RecordViewerJoin(roomID, streamerID, streamerName string) {
	m.ViewerJoinTotal.WithLabelValues(roomID, streamerID).Inc()
	m.RoomViewersTotal.WithLabelValues(roomID, streamerID, streamerName).Inc()
}

// RecordViewerLeave 记录观众离开
func (m *LiveStreamMetrics) RecordViewerLeave(roomID, streamerID string, duration float64) {
	m.ViewerLeaveTotal.WithLabelValues(roomID, streamerID).Inc()
	m.ViewerDuration.WithLabelValues(roomID).Observe(duration)
}

// RecordGift 记录礼物
func (m *LiveStreamMetrics) RecordGift(roomID, streamerID, streamerName, userID, userName, giftType, giftName string, giftValue float64) {
	m.RoomGiftsTotal.WithLabelValues(roomID, streamerID).Inc()
	m.RoomRevenueTotal.WithLabelValues(roomID, streamerID, streamerName).Add(giftValue)
	m.UserGiftsTotal.WithLabelValues(userID, userName, roomID).Add(giftValue)
	m.GiftCount.WithLabelValues(giftType, giftName, roomID).Inc()
	m.GiftRevenue.WithLabelValues(giftType, giftName, roomID).Add(giftValue)
}

// UpdateRoomViewers 更新直播间观众数
func (m *LiveStreamMetrics) UpdateRoomViewers(roomID, streamerID, streamerName, roomTitle string, viewers float64) {
	m.RoomViewers.WithLabelValues(roomID, streamerID, streamerName, roomTitle).Set(viewers)
}

// UpdateRoomStatus 更新直播间状态
func (m *LiveStreamMetrics) UpdateRoomStatus(roomID, streamerID, streamerName string, status float64) {
	m.RoomStatus.WithLabelValues(roomID, streamerID, streamerName).Set(status)
}

// UpdatePeakViewers 更新峰值观众数
func (m *LiveStreamMetrics) UpdatePeakViewers(roomID, streamerID string, viewers float64, period string) {
	m.RoomPeakViewers.WithLabelValues(roomID, streamerID, period).Set(viewers)
}

// UpdateGlobalPeakViewers 更新全局峰值观众数
func (m *LiveStreamMetrics) UpdateGlobalPeakViewers(viewers float64, period string) {
	m.GlobalPeakViewers.WithLabelValues(period).Set(viewers)
}
