package internal

import (
	"github.com/shopspring/decimal"
	"time"
)

type Player struct {
	Name         string
	Wallet       Wallet
	BuyStrategy  *BuyStrategy
	SellStrategy *SellStrategy
}

func (p *Player) Handle(lastPrice decimal.Decimal, queue *Queue, histories []*Queue) {
	if p.Wallet.Coins.IsPositive() && p.IsSell(queue, histories) {
		p.Wallet.Sell(queue.Timestamp, lastPrice)
	} else if p.Wallet.Cash.IsPositive() && p.IsBuy(queue, histories) {
		p.Wallet.Buy(queue.Timestamp, lastPrice)
	}
}

type SellStrategy struct {
	ListenSeconds        int64           // 监控最近秒数内资金净流出值
	CountSeconds         int64           // 统计最近秒数内平均资金净流出值
	MaxCountEverySeconds decimal.Decimal // 负数，每秒平均值阈值，统计结果小于该平均值才会比较最近净流出值
	TriggerTimes         decimal.Decimal // 最近秒数内资金净流出值达到平均净流入值的多少倍后开始卖出, 可以是小数
}

type BuyStrategy struct {
	ListenSeconds        int64           // 监控最近秒数内资金净流入值
	CountSeconds         int64           // 统计最近秒数内的平均资金净流入值
	MinCountEverySeconds decimal.Decimal // 正数，平均值阈值，统计结果大于该平均值才会开始比较净流入值
	TriggerTimes         decimal.Decimal // 最近秒数内资金净流入值达到平均净流入值的多少倍后开始买入, 可以是小数
}

func NewPlayer(name string, ss *SellStrategy, bs *BuyStrategy) *Player {
	return &Player{
		Name:         name,
		Wallet:       Wallet{Cash: decimal.New(1000, 0)},
		BuyStrategy:  bs,
		SellStrategy: ss,
	}
}

func (p *Player) IsBuy(queue *Queue, histories []*Queue) bool {
	now := time.Now().Unix() * 1000
	counts := countInflowCash(now, p.BuyStrategy.ListenSeconds, p.BuyStrategy.CountSeconds, histories)
	if counts.GreaterThanOrEqual(p.BuyStrategy.MinCountEverySeconds.Mul(decimal.NewFromFloat(float64(60) / float64(p.BuyStrategy.CountSeconds)))) {
		latest := countInflowCash(now, 0, p.BuyStrategy.ListenSeconds, append(histories, queue))
		// 按时间比列换算等比数值
		latest = latest.Mul(decimal.NewFromFloat(float64(p.BuyStrategy.CountSeconds) / float64(p.BuyStrategy.ListenSeconds)))
		if latest.Mul(p.BuyStrategy.TriggerTimes).GreaterThanOrEqual(counts) {
			return true
		}
	}
	return false
}

func (p *Player) IsSell(queue *Queue, histories []*Queue) bool {
	now := time.Now().Unix() * 1000
	counts := countInflowCash(now, p.SellStrategy.ListenSeconds, p.SellStrategy.CountSeconds, histories)
	if counts.LessThanOrEqual(p.SellStrategy.MaxCountEverySeconds.Mul(decimal.NewFromFloat(float64(60) / float64(p.SellStrategy.CountSeconds)))) {
		latest := countInflowCash(now, 0, p.SellStrategy.ListenSeconds, append(histories, queue))
		// 按时间比列换算等比数值
		latest = latest.Mul(decimal.NewFromFloat(float64(p.SellStrategy.CountSeconds) / float64(p.SellStrategy.ListenSeconds)))
		if latest.Mul(p.SellStrategy.TriggerTimes).LessThanOrEqual(counts) {
			return true
		}
	}
	return false
}

// 获得 startSeconds 为起始时间， totalSeconds 时间内资金净流入总额
func countInflowCash(now, startSeconds, totalSeconds int64, histories []*Queue) (total decimal.Decimal) {
	start := now + startSeconds*1000
	end := start + totalSeconds*1000
	total = decimal.Decimal{}
	for idx := len(histories) - 1; idx >= 0; idx-- {
		history := histories[idx]
		if start <= history.Timestamp && history.Timestamp <= end {
			total = total.Add(history.InflowCash)
		} else if end < history.Timestamp {
			break
		}
	}
	return total
}
