package flow

import (
	"github.com/huobirdcenter/huobi_golang/config"
	"github.com/huobirdcenter/huobi_golang/logging/applogger"
	"github.com/huobirdcenter/huobi_golang/pkg/client/marketwebsocketclient"
	"github.com/huobirdcenter/huobi_golang/pkg/model/market"
	"github.com/shopspring/decimal"
	"sync"
)

type Client struct {
	clientId     string
	symbol       string
	containers   []*container
	handler      Handler
	flowDuration int64
	flow         *Flow
	mu           sync.Mutex
}

func (c *Client) GetSection(duration int64) *Section {
	for _, container := range c.containers {
		if container.duration == duration {
			return container.section
		}
	}
	return nil
}

func NewClient(clientId, symbol string, flowDuration int64) *Client {
	return &Client{
		clientId:     clientId,
		symbol:       symbol,
		containers:   nil,
		flowDuration: flowDuration,
		flow:         &Flow{},
		mu:           sync.Mutex{},
	}
}

type SectionGetter interface {
	GetSection(duration int64) *Section
}

type Handler func(price decimal.Decimal, sectionGetter SectionGetter)

func (c *Client) Listen(durations []int64, handler Handler) {
	for _, duration := range durations {
		c.containers = append(c.containers, newContainer(duration))
	}
	c.handler = handler
}

func (c *Client) Subscribe() (closeFunc func()) {
	return Subscribe(c.symbol, c.clientId, func(response market.SubscribeTradeResponse) {
		if response.Tick != nil && response.Tick.Data != nil {
			c.mu.Lock()
			defer c.mu.Unlock()
			for idx, t := range response.Tick.Data {
				t.Timestamp = t.Timestamp / 1000
				if c.flow.Timestamp == 0 {
					c.flow.Timestamp = t.Timestamp
				}
				cash := t.Price.Mul(t.Amount).IntPart()
				if t.Direction == "buy" {
					c.flow.Buy += cash
					c.flow.Inflow += cash
				} else {
					c.flow.Sell += cash
					c.flow.Inflow -= cash
				}
				if idx == len(response.Tick.Data)-1 {
					if t.Timestamp >= c.flow.Timestamp+c.flowDuration {
						for _, container := range c.containers {
							container.push(c.flow)
						}
						c.handler(t.Price, c)
						c.flow = &Flow{}
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
