package main

import (
	"github.com/prometheus/client_golang/prometheus"
)

// ViewerMetrics 观众相关指标
type ViewerMetrics struct {
	// 单个观众指标 - 在线状态
	ViewerOnlineStatus   *prometheus.GaugeVec   // 观众在线状态 (0=离线, 1=在线)
	ViewerOnlineDuration *prometheus.CounterVec // 观众总在线时长(秒)
	ViewerLastActiveTime *prometheus.GaugeVec   // 观众最后活跃时间戳

	// 单个观众指标 - 观看时长
	ViewerWatchDurationTotal    *prometheus.CounterVec // 累计观看时长(秒)
	ViewerWatchDurationToday    *prometheus.GaugeVec   // 今日观看时长(秒)
	ViewerWatchDurationWeek     *prometheus.GaugeVec   // 本周观看时长(秒)
	ViewerWatchDurationMonth    *prometheus.GaugeVec   // 本月观看时长(秒)
	ViewerWatchDurationDailyAvg *prometheus.GaugeVec   // 平均每天观看时长(秒)
	ViewerWatchStreakDays       *prometheus.GaugeVec   // 连续观看天数

	// 单个观众指标 - 活跃度
	ViewerRoomVisitsTotal       *prometheus.CounterVec // 进入直播间总次数
	ViewerStreamersWatchedTotal *prometheus.CounterVec // 观看过的主播数量
	ViewerActiveDaysLast7Days   *prometheus.GaugeVec   // 最近7天活跃天数
	ViewerActiveDaysLast30Days  *prometheus.GaugeVec   // 最近30天活跃天数

	// 单个观众指标 - 来源与关系
	ViewerSourceStreamer *prometheus.GaugeVec // 该观众是通过哪个主播创建的角色

	// 单个观众指标 - 礼物统计
	ViewerGiftsSentTotal *prometheus.CounterVec // 观众送出的礼物总数（按礼物类型）

	// 全局观众统计 - 注册与在线
	ViewersRegisteredTotal prometheus.Counter // 注册观众总数
	// ViewersOnlineTotal - 已在 LiveStreamMetrics 中定义

	// 全局观众统计 - 在线分布
	ViewersOnlineByHour    *prometheus.GaugeVec // 按小时统计在线观众数
	ViewersOnlineByDayType *prometheus.GaugeVec // 按工作日/周末统计

	// 全局观众统计 - 活跃度
	ViewersActiveLast7Days  prometheus.Gauge   // 最近7天活跃观众数
	ViewersActiveLast30Days prometheus.Gauge   // 最近30天活跃观众数
	ViewersNewToday         prometheus.Counter // 今日新增观众
	ViewersNewWeek          prometheus.Counter // 本周新增观众
	ViewersNewMonth         prometheus.Counter // 本月新增观众

	// 全局观众统计 - 留存与流失
	ViewersChurned7Days  prometheus.Gauge     // 7天未活跃观众数
	ViewersChurned30Days prometheus.Gauge     // 30天未活跃观众数
	ViewerRetentionRate  *prometheus.GaugeVec // 留存率 (period=1d/7d/30d)
}

// NewViewerMetrics 创建观众指标
func NewViewerMetrics(reg *prometheus.Registry) *ViewerMetrics {
	metrics := &ViewerMetrics{
		// 单个观众 - 在线状态
		ViewerOnlineStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "viewer_online_status",
				Help: "观众在线状态 (0=离线, 1=在线)",
			},
			[]string{"viewer_id", "viewer_name"},
		),
		ViewerOnlineDuration: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "viewer_online_duration_seconds",
				Help: "观众总在线时长(秒)",
			},
			[]string{"viewer_id", "viewer_name"},
		),
		ViewerLastActiveTime: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "viewer_last_active_timestamp",
				Help: "观众最后活跃时间戳",
			},
			[]string{"viewer_id", "viewer_name"},
		),

		// 单个观众 - 观看时长
		ViewerWatchDurationTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "viewer_watch_duration_total_seconds",
				Help: "观众累计观看时长(秒)",
			},
			[]string{"viewer_id", "viewer_name"},
		),
		ViewerWatchDurationToday: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "viewer_watch_duration_today_seconds",
				Help: "观众今日观看时长(秒)",
			},
			[]string{"viewer_id", "viewer_name"},
		),
		ViewerWatchDurationWeek: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "viewer_watch_duration_week_seconds",
				Help: "观众本周观看时长(秒)",
			},
			[]string{"viewer_id", "viewer_name"},
		),
		ViewerWatchDurationMonth: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "viewer_watch_duration_month_seconds",
				Help: "观众本月观看时长(秒)",
			},
			[]string{"viewer_id", "viewer_name"},
		),
		ViewerWatchDurationDailyAvg: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "viewer_watch_duration_daily_avg_seconds",
				Help: "观众平均每天观看时长(秒)",
			},
			[]string{"viewer_id", "viewer_name"},
		),
		ViewerWatchStreakDays: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "viewer_watch_streak_days",
				Help: "观众连续观看天数",
			},
			[]string{"viewer_id", "viewer_name"},
		),

		// 单个观众 - 活跃度
		ViewerRoomVisitsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "viewer_room_visits_total",
				Help: "观众进入直播间总次数",
			},
			[]string{"viewer_id", "viewer_name"},
		),
		ViewerStreamersWatchedTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "viewer_streamers_watched_total",
				Help: "观众观看过的主播数量",
			},
			[]string{"viewer_id", "viewer_name"},
		),
		ViewerActiveDaysLast7Days: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "viewer_active_days_last_7days",
				Help: "观众最近7天活跃天数",
			},
			[]string{"viewer_id", "viewer_name"},
		),
		ViewerActiveDaysLast30Days: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "viewer_active_days_last_30days",
				Help: "观众最近30天活跃天数",
			},
			[]string{"viewer_id", "viewer_name"},
		),

		// 单个观众 - 来源与关系
		ViewerSourceStreamer: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "viewer_source_streamer",
				Help: "观众是通过哪个主播创建的角色 (值为1表示关联)",
			},
			[]string{"viewer_id", "viewer_name", "source_streamer_id", "source_streamer_name"},
		),

		// 单个观众 - 礼物统计
		ViewerGiftsSentTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "viewer_gifts_sent_total",
				Help: "观众送出的礼物总数（按礼物类型）",
			},
			[]string{"viewer_id", "viewer_name", "gift_type", "gift_name"},
		),

		// 全局观众统计 - 注册与在线
		ViewersRegisteredTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "viewers_registered_total",
				Help: "注册观众总数",
			},
		),

		// 全局观众统计 - 在线分布
		ViewersOnlineByHour: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "viewers_online_by_hour",
				Help: "按小时统计的在线观众数",
			},
			[]string{"hour"},
		),
		ViewersOnlineByDayType: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "viewers_online_by_day_type",
				Help: "按工作日/周末统计的在线观众数",
			},
			[]string{"day_type"},
		),

		// 全局观众统计 - 活跃度
		ViewersActiveLast7Days: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "viewers_active_last_7days",
				Help: "最近7天活跃观众数",
			},
		),
		ViewersActiveLast30Days: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "viewers_active_last_30days",
				Help: "最近30天活跃观众数",
			},
		),
		ViewersNewToday: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "viewers_new_today",
				Help: "今日新增观众",
			},
		),
		ViewersNewWeek: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "viewers_new_week",
				Help: "本周新增观众",
			},
		),
		ViewersNewMonth: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "viewers_new_month",
				Help: "本月新增观众",
			},
		),

		// 全局观众统计 - 留存与流失
		ViewersChurned7Days: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "viewers_churned_7days",
				Help: "7天未活跃观众数",
			},
		),
		ViewersChurned30Days: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "viewers_churned_30days",
				Help: "30天未活跃观众数",
			},
		),
		ViewerRetentionRate: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "viewer_retention_rate",
				Help: "观众留存率 (period=1d/7d/30d)",
			},
			[]string{"period"},
		),
	}

	// 注册所有指标
	reg.MustRegister(
		metrics.ViewerOnlineStatus,
		metrics.ViewerOnlineDuration,
		metrics.ViewerLastActiveTime,
		metrics.ViewerWatchDurationTotal,
		metrics.ViewerWatchDurationToday,
		metrics.ViewerWatchDurationWeek,
		metrics.ViewerWatchDurationMonth,
		metrics.ViewerWatchDurationDailyAvg,
		metrics.ViewerWatchStreakDays,
		metrics.ViewerRoomVisitsTotal,
		metrics.ViewerStreamersWatchedTotal,
		metrics.ViewerActiveDaysLast7Days,
		metrics.ViewerActiveDaysLast30Days,
		metrics.ViewerSourceStreamer,
		metrics.ViewerGiftsSentTotal,
		metrics.ViewersRegisteredTotal,
		metrics.ViewersOnlineByHour,
		metrics.ViewersOnlineByDayType,
		metrics.ViewersActiveLast7Days,
		metrics.ViewersActiveLast30Days,
		metrics.ViewersNewToday,
		metrics.ViewersNewWeek,
		metrics.ViewersNewMonth,
		metrics.ViewersChurned7Days,
		metrics.ViewersChurned30Days,
		metrics.ViewerRetentionRate,
	)

	return metrics
}
