package main

import (
	"github.com/prometheus/client_golang/prometheus"
)

// StreamerMetrics 主播相关指标
type StreamerMetrics struct {
	// 单个主播指标 - 在线状态
	StreamerOnlineStatus       *prometheus.GaugeVec   // 主播在线状态 (0=离线, 1=在线未直播, 2=直播中)
	StreamerOnlineDuration     *prometheus.CounterVec // 主播总在线时长(秒)
	StreamerLastOnlineTime     *prometheus.GaugeVec   // 主播最后在线时间戳

	// 单个主播指标 - 直播时长
	StreamerStreamingDurationTotal    *prometheus.CounterVec // 总直播时长(秒)
	StreamerStreamingDurationToday    *prometheus.GaugeVec   // 今日直播时长(秒)
	StreamerStreamingDurationWeek     *prometheus.GaugeVec   // 本周直播时长(秒)
	StreamerStreamingDurationMonth    *prometheus.GaugeVec   // 本月直播时长(秒)
	StreamerStreamingDurationDailyAvg *prometheus.GaugeVec   // 平均每天直播时长(秒)
	StreamerStreamingStreakDays       *prometheus.GaugeVec   // 连续直播天数

	// 单个主播指标 - 活跃度
	StreamerStreamSessionsTotal  *prometheus.CounterVec // 开播总次数
	StreamerActiveDaysLast7Days  *prometheus.GaugeVec   // 最近7天开播天数
	StreamerActiveDaysLast30Days *prometheus.GaugeVec   // 最近30天开播天数

	// 单个主播指标 - 观众关系
	StreamerAttractedViewersTotal  *prometheus.GaugeVec // 通过该主播创建角色的观众总数
	StreamerAttractedViewersActive *prometheus.GaugeVec // 这些观众中仍活跃的数量

	// 单个主播指标 - 礼物统计
	StreamerGiftsReceivedTotal *prometheus.CounterVec // 主播收到的礼物总数（按礼物类型）

	// 全局主播统计 - 在线状态
	StreamersOnlineTotal    prometheus.Gauge // 当前在线主播总数
	StreamersStreamingTotal prometheus.Gauge // 当前正在直播的主播数

	// 全局主播统计 - 在线分布
	StreamersOnlineByHour    *prometheus.GaugeVec // 按小时统计在线主播数
	StreamersOnlineByDayType *prometheus.GaugeVec // 按工作日/周末统计

	// 全局主播统计 - 活跃度
	StreamersActiveLast7Days  prometheus.Gauge // 最近7天活跃主播数
	StreamersActiveLast30Days prometheus.Gauge // 最近30天活跃主播数
}

// NewStreamerMetrics 创建主播指标
func NewStreamerMetrics(reg *prometheus.Registry) *StreamerMetrics {
	metrics := &StreamerMetrics{
		// 单个主播 - 在线状态
		StreamerOnlineStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "streamer_online_status",
				Help: "主播在线状态 (0=离线, 1=在线未直播, 2=直播中)",
			},
			[]string{"streamer_id", "streamer_name"},
		),
		StreamerOnlineDuration: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "streamer_online_duration_seconds",
				Help: "主播总在线时长(秒)",
			},
			[]string{"streamer_id", "streamer_name"},
		),
		StreamerLastOnlineTime: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "streamer_last_online_timestamp",
				Help: "主播最后在线时间戳",
			},
			[]string{"streamer_id", "streamer_name"},
		),

		// 单个主播 - 直播时长
		StreamerStreamingDurationTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "streamer_streaming_duration_total_seconds",
				Help: "主播总直播时长(秒)",
			},
			[]string{"streamer_id", "streamer_name"},
		),
		StreamerStreamingDurationToday: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "streamer_streaming_duration_today_seconds",
				Help: "主播今日直播时长(秒)",
			},
			[]string{"streamer_id", "streamer_name"},
		),
		StreamerStreamingDurationWeek: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "streamer_streaming_duration_week_seconds",
				Help: "主播本周直播时长(秒)",
			},
			[]string{"streamer_id", "streamer_name"},
		),
		StreamerStreamingDurationMonth: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "streamer_streaming_duration_month_seconds",
				Help: "主播本月直播时长(秒)",
			},
			[]string{"streamer_id", "streamer_name"},
		),
		StreamerStreamingDurationDailyAvg: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "streamer_streaming_duration_daily_avg_seconds",
				Help: "主播平均每天直播时长(秒)",
			},
			[]string{"streamer_id", "streamer_name"},
		),
		StreamerStreamingStreakDays: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "streamer_streaming_streak_days",
				Help: "主播连续直播天数",
			},
			[]string{"streamer_id", "streamer_name"},
		),

		// 单个主播 - 活跃度
		StreamerStreamSessionsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "streamer_stream_sessions_total",
				Help: "主播开播总次数",
			},
			[]string{"streamer_id", "streamer_name"},
		),
		StreamerActiveDaysLast7Days: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "streamer_active_days_last_7days",
				Help: "主播最近7天开播天数",
			},
			[]string{"streamer_id", "streamer_name"},
		),
		StreamerActiveDaysLast30Days: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "streamer_active_days_last_30days",
				Help: "主播最近30天开播天数",
			},
			[]string{"streamer_id", "streamer_name"},
		),

		// 单个主播 - 观众关系
		StreamerAttractedViewersTotal: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "streamer_attracted_viewers_total",
				Help: "通过该主播创建角色的观众总数",
			},
			[]string{"streamer_id", "streamer_name"},
		),
		StreamerAttractedViewersActive: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "streamer_attracted_viewers_active",
				Help: "通过该主播创建角色且仍活跃的观众数量",
			},
			[]string{"streamer_id", "streamer_name"},
		),

		// 单个主播 - 礼物统计
		StreamerGiftsReceivedTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "streamer_gifts_received_total",
				Help: "主播收到的礼物总数（按礼物类型）",
			},
			[]string{"streamer_id", "streamer_name", "gift_type", "gift_name"},
		),

		// 全局主播统计 - 在线状态
		StreamersOnlineTotal: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "streamers_online_total",
				Help: "当前在线主播总数",
			},
		),
		StreamersStreamingTotal: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "streamers_streaming_total",
				Help: "当前正在直播的主播数",
			},
		),

		// 全局主播统计 - 在线分布
		StreamersOnlineByHour: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "streamers_online_by_hour",
				Help: "按小时统计的在线主播数",
			},
			[]string{"hour"},
		),
		StreamersOnlineByDayType: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "streamers_online_by_day_type",
				Help: "按工作日/周末统计的在线主播数",
			},
			[]string{"day_type"},
		),

		// 全局主播统计 - 活跃度
		StreamersActiveLast7Days: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "streamers_active_last_7days",
				Help: "最近7天活跃主播数",
			},
		),
		StreamersActiveLast30Days: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "streamers_active_last_30days",
				Help: "最近30天活跃主播数",
			},
		),
	}

	// 注册所有指标
	reg.MustRegister(
		metrics.StreamerOnlineStatus,
		metrics.StreamerOnlineDuration,
		metrics.StreamerLastOnlineTime,
		metrics.StreamerStreamingDurationTotal,
		metrics.StreamerStreamingDurationToday,
		metrics.StreamerStreamingDurationWeek,
		metrics.StreamerStreamingDurationMonth,
		metrics.StreamerStreamingDurationDailyAvg,
		metrics.StreamerStreamingStreakDays,
		metrics.StreamerStreamSessionsTotal,
		metrics.StreamerActiveDaysLast7Days,
		metrics.StreamerActiveDaysLast30Days,
		metrics.StreamerAttractedViewersTotal,
		metrics.StreamerAttractedViewersActive,
		metrics.StreamerGiftsReceivedTotal,
		metrics.StreamersOnlineTotal,
		metrics.StreamersStreamingTotal,
		metrics.StreamersOnlineByHour,
		metrics.StreamersOnlineByDayType,
		metrics.StreamersActiveLast7Days,
		metrics.StreamersActiveLast30Days,
	)

	return metrics
}
