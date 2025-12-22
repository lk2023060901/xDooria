package main

import (
	"fmt"
	"log"

	"github.com/spf13/viper"
)

// 道具配置
type ItemConfig struct {
	ID          int                    `json:"id" mapstructure:"id"`
	Name        string                 `json:"name" mapstructure:"name"`
	Type        int                    `json:"type" mapstructure:"type"`
	Rarity      int                    `json:"rarity" mapstructure:"rarity"`
	MaxStack    int                    `json:"max_stack" mapstructure:"max_stack"`
	Price       int                    `json:"price" mapstructure:"price"`
	Description string                 `json:"description" mapstructure:"description"`
	Icon        string                 `json:"icon" mapstructure:"icon"`
	Attributes  map[string]interface{} `json:"attributes,omitempty" mapstructure:"attributes,omitempty"`
}

type ItemsConfig struct {
	Items []ItemConfig `json:"items" mapstructure:"items"`
}

// 关卡配置
type LevelConfig struct {
	ID              int      `json:"id" mapstructure:"id"`
	Name            string   `json:"name" mapstructure:"name"`
	Difficulty      int      `json:"difficulty" mapstructure:"difficulty"`
	MaxPlayers      int      `json:"max_players" mapstructure:"max_players"`
	TimeLimit       int      `json:"time_limit" mapstructure:"time_limit"`
	RewardExp       int      `json:"reward_exp" mapstructure:"reward_exp"`
	RewardGold      int      `json:"reward_gold" mapstructure:"reward_gold"`
	RequiredLevel   int      `json:"required_level" mapstructure:"required_level"`
	UnlockCondition []string `json:"unlock_condition" mapstructure:"unlock_condition"`
}

type LevelsConfig struct {
	Levels []LevelConfig `json:"levels" mapstructure:"levels"`
}

func main() {
	fmt.Println("=== Viper JSON 示例 ===")

	// 示例 1: 加载道具配置
	example1()

	// 示例 2: 加载关卡配置
	example2()

	// 示例 3: UnmarshalKey - 只加载配置的某个部分
	example3()
}

// 示例 1: 加载道具配置
func example1() {
	fmt.Println("--- 示例 1: 加载道具配置 ---")

	v := viper.New()
	v.SetConfigFile("items.json")
	v.SetConfigType("json")

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		log.Fatalf("读取配置失败: %v", err)
	}

	// 解析到结构体
	var itemsCfg ItemsConfig
	if err := v.Unmarshal(&itemsCfg); err != nil {
		log.Fatalf("解析配置失败: %v", err)
	}

	// 输出配置
	fmt.Printf("加载了 %d 个道具:\n", len(itemsCfg.Items))
	for _, item := range itemsCfg.Items {
		fmt.Printf("  [%d] %s (稀有度:%d, 最大堆叠:%d, 价格:%d)\n",
			item.ID, item.Name, item.Rarity, item.MaxStack, item.Price)
		fmt.Printf("      描述: %s\n", item.Description)
		if len(item.Attributes) > 0 {
			fmt.Printf("      属性: %v\n", item.Attributes)
		}
	}
	fmt.Println()
}

// 示例 2: 加载关卡配置
func example2() {
	fmt.Println("--- 示例 2: 加载关卡配置 ---")

	v := viper.New()
	v.SetConfigFile("levels.json")
	v.SetConfigType("json")

	if err := v.ReadInConfig(); err != nil {
		log.Fatalf("读取配置失败: %v", err)
	}

	var levelsCfg LevelsConfig
	if err := v.Unmarshal(&levelsCfg); err != nil {
		log.Fatalf("解析配置失败: %v", err)
	}

	fmt.Printf("加载了 %d 个关卡:\n", len(levelsCfg.Levels))
	for _, level := range levelsCfg.Levels {
		fmt.Printf("  [%d] %s\n", level.ID, level.Name)
		fmt.Printf("      难度:%d | 最大玩家数:%d | 时间限制:%ds\n",
			level.Difficulty, level.MaxPlayers, level.TimeLimit)
		fmt.Printf("      奖励: 经验值:%d, 金币:%d\n",
			level.RewardExp, level.RewardGold)
		fmt.Printf("      要求等级:%d | 解锁条件:%v\n",
			level.RequiredLevel, level.UnlockCondition)
	}
	fmt.Println()
}

// 示例 3: UnmarshalKey - 只加载配置数组的某个部分
func example3() {
	fmt.Println("--- 示例 3: UnmarshalKey - 只加载 items 数组 ---")

	v := viper.New()
	v.SetConfigFile("items.json")
	v.SetConfigType("json")

	if err := v.ReadInConfig(); err != nil {
		log.Fatalf("读取配置失败: %v", err)
	}

	// 直接解析 items 数组
	var items []ItemConfig
	if err := v.UnmarshalKey("items", &items); err != nil {
		log.Fatalf("解析 items 失败: %v", err)
	}

	fmt.Printf("通过 UnmarshalKey 加载了 %d 个道具\n", len(items))
	for _, item := range items {
		fmt.Printf("  - %s (ID:%d)\n", item.Name, item.ID)
	}
	fmt.Println()
}
