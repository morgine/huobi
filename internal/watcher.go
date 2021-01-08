package internal

import (
	"github.com/huobirdcenter/huobi_golang/pkg/model/market"
	"sync"
)

type HandlerOption struct {
	OffsetSeconds   int64 // 监控时间相对于最新现金流时间的偏移量，单位秒
	DurationSeconds int64 // 监控时间范围
	BuyCash         int64 // 购买资金总量, 0 代表不监控
	SellCash        int64 // 卖出资金总量, 0 代表不监控
	InflowCash      int64 // 资金净流入, 0 代表不监控
	OutflowCash     int64 // 资金净流出, 0 代表不监控
}

// 现金流
type CashFlow struct {
	BuyCash    int64 // 购买资金总量
	SellCash   int64 // 卖出资金总量
	InflowCash int64 // 资金净流入，负数表示净流出
	Timestamp  int64 // 时间戳，单位：秒
}

type SectionFlow struct {
	BuyCash    int64 // 购买资金总量
	SellCash   int64 // 卖出资金总量
	InflowCash int64 // 资金净流入，负数代表净流出
	StartTime  int64 // 开始时间戳，单位：秒
	EndTime    int64 // 结束时间戳，单位：秒
}

// 是否达到触发警报
func (w *HandlerOption) IsAlert(flows []*CashFlow) (sectionFlow *SectionFlow, alert bool) {
	sectionFlow = getSectionFlow(0, w.OffsetSeconds, w.DurationSeconds, flows)
	if sectionFlow == nil {
		return nil, false
	}
	// 资金净流入流出
	if w.InflowCash > 0 && sectionFlow.InflowCash >= w.InflowCash {
		//inflow = total.InflowCash
		alert = true
	} else if w.InflowCash < 0 && sectionFlow.InflowCash <= w.InflowCash {
		//inflow = total.InflowCash
		alert = true
	} else {
		sectionFlow.InflowCash = 0
	}
	// 购买总量
	if w.BuyCash > 0 && sectionFlow.BuyCash >= w.BuyCash {
		alert = true
	} else {
		sectionFlow.BuyCash = 0
	}
	// 卖出总量
	if w.SellCash > 0 && sectionFlow.SellCash >= w.SellCash {
		alert = true
	} else {
		sectionFlow.SellCash = 0
	}
	if alert {
		return sectionFlow, true
	} else {
		return nil, false
	}
}

type FlowHandler struct {
	Option HandlerOption
	// 达到警报条件处理警报结果, 根据统计的结果值来判断触发了那些结果, 值为 0 表示未触发该结果
	Handle func(sf *SectionFlow)
}

type WatcherOptions struct {
	Symbol         string
	ClientId       string
	SectionSeconds int64 // 数据分段时差
	MaxFlows       int   // 最大队列数量，超过该阈值则删除一部分老的数据
	DelFlows       int   // 队列超过阈值时删除数据量
}

type SectionFlowContainer struct {
	total *SectionFlow
	flows []*CashFlow
	flow  *CashFlow
}

type FlowWatcher struct {
	symbol         string
	clientId       string
	flows          []*CashFlow
	flow           *CashFlow
	sectionSeconds int64 // 数据分段时差
	maxFlows       int   // 最大队列数量，超过该阈值则删除一部分老的数据
	delFlows       int   // 队列超过阈值时删除数据量
	handlers       []*FlowHandler
	mu             sync.Mutex
}

func NewWatcher(options *WatcherOptions) *FlowWatcher {
	return &FlowWatcher{
		symbol:         options.Symbol,
		clientId:       options.ClientId,
		flows:          nil,
		flow:           &CashFlow{},
		sectionSeconds: options.SectionSeconds,
		maxFlows:       options.MaxFlows,
		delFlows:       options.DelFlows,
		mu:             sync.Mutex{},
	}
}

func (w *FlowWatcher) Handle(handler ...*FlowHandler) {
	w.handlers = append(w.handlers, handler...)
}

func (w *FlowWatcher) pushFlow() {
	w.flows = append(w.flows, w.flow)
	if len(w.flows) > w.maxFlows {
		w.flows = w.flows[w.delFlows:]
	}
	for _, handler := range w.handlers {
		if total, alert := handler.Option.IsAlert(w.flows); alert {
			handler.Handle(total)
		}
	}
	w.flow = &CashFlow{}
}

func (w *FlowWatcher) GetSectionFlow(startTime, duration int64) *SectionFlow {
	w.mu.Lock()
	defer w.mu.Unlock()
	endTime := startTime + duration
	return getSectionFlow(endTime, 0, duration, append(w.flows, w.flow))
}

func (w *FlowWatcher) GetFlow() *CashFlow {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.flow
}

func (w *FlowWatcher) GetFlows() []*CashFlow {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append(w.flows, w.flow)
}

func (w *FlowWatcher) Subscribe() (closeFunc func()) {
	return Subscribe(w.symbol, w.clientId, func(response market.SubscribeTradeResponse) {
		if response.Tick != nil && response.Tick.Data != nil {
			w.mu.Lock()
			defer w.mu.Unlock()
			for idx, t := range response.Tick.Data {
				t.Timestamp = t.Timestamp / 1000
				if w.flow.Timestamp == 0 {
					w.flow.Timestamp = t.Timestamp
				}
				cash := t.Price.Mul(t.Amount).IntPart()
				if t.Direction == "buy" {
					w.flow.BuyCash += cash
					w.flow.InflowCash += cash
				} else {
					w.flow.SellCash += cash
					w.flow.InflowCash -= cash
				}
				if idx == len(response.Tick.Data)-1 {
					if t.Timestamp >= w.flow.Timestamp+w.sectionSeconds {
						w.pushFlow()
						w.flow = &CashFlow{}
					}
				}
			}
		}
	})
}

// 获取时间段内总量, 注意时间线是从后往前推
func getSectionFlow(endTime, offsetTime, duration int64, flows []*CashFlow) *SectionFlow {
	if length := len(flows); length > 0 {
		if endTime <= 0 {
			endTime = flows[0].Timestamp
		}
		endTime -= offsetTime
		startTime := endTime - duration
		var total = &SectionFlow{StartTime: startTime, EndTime: endTime}
		for i := length - 1; i >= 0; i-- {
			flow := flows[i]
			if flow.Timestamp >= startTime {
				total.BuyCash += flow.BuyCash
				total.SellCash += flow.SellCash
				total.InflowCash += flow.InflowCash
			} else {
				break
			}
		}
		return total
	}
	return nil
}
