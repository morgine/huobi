package internal

import (
	"github.com/huobirdcenter/huobi_golang/config"
	"github.com/huobirdcenter/huobi_golang/logging/applogger"
	"github.com/huobirdcenter/huobi_golang/pkg/client/marketwebsocketclient"
	"github.com/huobirdcenter/huobi_golang/pkg/model/market"
	"github.com/shopspring/decimal"
	"sync"
)

// 新队列处理器
type QueueHandler func(lastPrice decimal.Decimal, queue *Queue, histories []*Queue)

type Client struct {
	symbol    string
	clientId  string
	histories []*Queue
	queue     *Queue
	duration  int64 // 统计时差(毫秒)，如 10000 毫秒
	maxQueues int   // 最大队列数量，超过该阈值则删除一部分老的数据
	delQueues int   // 队列超过阈值时删除数据量
	handlers  []QueueHandler
	mu        sync.Mutex
}

type ClientOptions struct {
	Symbol    string
	ClientId  string
	Duration  int64 // 统计时差(毫秒)，如 10000 毫秒
	MaxQueues int   // 最大队列数量，超过该阈值则删除一部分老的数据
	DelQueues int   // 队列超过阈值时删除数据量
}

func NewClient(options *ClientOptions) *Client {
	return &Client{
		symbol:    options.Symbol,
		clientId:  options.ClientId,
		histories: nil,
		queue:     &Queue{},
		duration:  options.Duration,
		maxQueues: options.MaxQueues,
		delQueues: options.DelQueues,
		mu:        sync.Mutex{},
	}
}

type Queue struct {
	InputCash   decimal.Decimal // 流入资金
	OutputCash  decimal.Decimal // 流出资金
	InflowCash  decimal.Decimal // 净流入资金, InputCash - OutputCash
	InputCoins  decimal.Decimal // 流入币
	OutputCoins decimal.Decimal // 流出币
	InflowCoins decimal.Decimal // 净流入币, InputCoins - OutputCoins
	BuyPrice    decimal.Decimal // 买入单价, InputCash / OutputCoins
	SellPrice   decimal.Decimal // 卖出单价, OutputCash / InputCoins
	InflowPrice decimal.Decimal // 买卖差价, BuyPrice - SellPrice
	Timestamp   int64           // 时间戳
}

// 计算净流入，及流入资金比例
func (q *Queue) calculate() (newQueue *Queue) {
	newQueue = &Queue{
		InputCash:   q.InputCash,
		OutputCash:  q.OutputCash,
		InflowCash:  q.InputCash.Sub(q.OutputCash),
		InputCoins:  q.InputCoins,
		OutputCoins: q.OutputCoins,
		InflowCoins: q.InputCoins.Sub(q.OutputCoins),
		Timestamp:   q.Timestamp,
	}
	if !q.InputCoins.IsZero() {
		newQueue.SellPrice = q.OutputCash.Div(q.InputCoins)
	}
	if !q.OutputCoins.IsZero() {
		newQueue.BuyPrice = q.InputCash.Div(q.OutputCoins)
	}
	newQueue.InflowPrice = newQueue.BuyPrice.Sub(newQueue.SellPrice)
	return newQueue
}

// 重置数据
func (q *Queue) reset() {
	q.InputCash = decimal.Decimal{}
	q.OutputCash = decimal.Decimal{}
	q.InflowCash = decimal.Decimal{}
	q.InputCoins = decimal.Decimal{}
	q.OutputCoins = decimal.Decimal{}
	q.InflowCoins = decimal.Decimal{}
	q.BuyPrice = decimal.Decimal{}
	q.SellPrice = decimal.Decimal{}
	q.InflowPrice = decimal.Decimal{}
	q.Timestamp = 0
}

func (c *Client) Handle(handler QueueHandler) {
	c.handlers = append(c.handlers, handler)
}

func (c *Client) GetQueues() []*Queue {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.histories
}

// 统计数据掐掉头尾，从每分钟的起始时间开始统计
func (c *Client) CountQueues(durationSeconds int64) []*Queue {
	c.mu.Lock()
	defer c.mu.Unlock()
	var histories []*Queue
	var startTime, endTime int64
	var current *Queue
	if len(c.histories) > 0 {
		startTime = c.histories[0].Timestamp / 60000 * 60000 // 删除秒以及毫秒数据，以每分钟的开始为起始时间
		endTime = startTime + durationSeconds*1000
	}
	for _, history := range c.histories {
		// 从第一个分钟的开始时间开始计数，前面的数据忽略掉
		if startTime > 0 && history.Timestamp < startTime {
			continue
		}
		if current == nil {
			current = history.calculate()                       // 使用该方法仅用来初始化，计算是多余的
			current.Timestamp = current.Timestamp / 1000 * 1000 // 删除毫秒数据，定位到秒
			endTime = current.Timestamp + durationSeconds*1000
		} else {
			if history.Timestamp < endTime {
				current.InputCash = current.InputCash.Add(history.InputCash)
				current.OutputCoins = current.OutputCoins.Add(history.OutputCoins)
				current.OutputCash = current.OutputCash.Add(history.OutputCash)
				current.InputCoins = current.InputCoins.Add(history.InputCoins)
			} else {
				// 进入下一轮计算
				histories = append(histories, current.calculate())  // calculate 用于计算
				current = history.calculate()                       // calculate 用于初始化
				current.Timestamp = current.Timestamp / 1000 * 1000 // 删除毫秒数据，定位到秒
				endTime = current.Timestamp + durationSeconds*1000
			}
		}
	}
	return histories
	// 抛弃临界值，数据不全可能造成数据异常，不够平均
	//if current != nil {
	//	histories = append(histories, current.calculate())
	//}
}

// 获得净流入
func (c *Client) GetInflows(startTime, durationSeconds int64) (cash, coins decimal.Decimal) {
	endTime := startTime + durationSeconds*1000
	c.mu.Lock()
	defer c.mu.Unlock()
	cash = decimal.Decimal{}
	coins = decimal.Decimal{}
	for _, history := range c.histories {
		applogger.Info("%d-%d-%d", startTime, endTime, history.Timestamp)
		if startTime <= history.Timestamp && history.Timestamp <= endTime {
			cash = cash.Add(history.InflowCash)
			coins = cash.Add(history.InflowCoins)
		}
	}
	return
}

func (c *Client) GetQueue() *Queue {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.queue == nil {
		return nil
	}
	return c.queue.calculate()
}

func (c *Client) Subscribe() (closeFunc func()) {
	return Subscribe(c.symbol, c.clientId, func(response market.SubscribeTradeResponse) {
		if response.Tick != nil && response.Tick.Data != nil {
			c.mu.Lock()
			defer c.mu.Unlock()
			for idx, t := range response.Tick.Data {
				if c.queue.Timestamp == 0 {
					c.queue.Timestamp = t.Timestamp
				}
				cash := t.Price.Mul(t.Amount)
				if t.Direction == "buy" {
					c.queue.InputCash = c.queue.InputCash.Add(cash)
					c.queue.OutputCoins = c.queue.OutputCoins.Add(t.Amount)
				} else {
					c.queue.OutputCash = c.queue.OutputCash.Add(cash)
					c.queue.InputCoins = c.queue.InputCoins.Add(t.Amount)
				}
				if idx == len(response.Tick.Data)-1 {
					if t.Timestamp > c.queue.Timestamp+c.duration {
						queue := c.queue.calculate()
						for _, handler := range c.handlers {
							handler(t.Price, queue, c.histories)
						}
						c.histories = append(c.histories, queue)
						c.queue.reset()
						c.queue.Timestamp = t.Timestamp
						if len(c.histories) > c.maxQueues {
							c.histories = c.histories[c.delQueues:]
						}
					}
				}
			}
		}
	})
}

func Subscribe(symbol, clientId string, handler func(response market.SubscribeTradeResponse)) (closeFunc func()) {
	client := new(marketwebsocketclient.TradeWebSocketClient).Init(config.Host)

	client.SetHandler(
		func() {
			client.Subscribe(symbol, clientId)
		},
		func(resp interface{}) {
			response, ok := resp.(market.SubscribeTradeResponse)
			if ok {
				if &response != nil {
					handler(response)
				}
			} else {
				applogger.Warn("symbol %s.%s got unknown response: %v", symbol, clientId, resp)
			}
		},
	)

	client.Connect(true)
	return func() {
		client.UnSubscribe(symbol, clientId)
		applogger.Info("symbol %s unsubscribed", symbol)
		client.Close()
		applogger.Info("client %s closed", clientId)
	}
}
