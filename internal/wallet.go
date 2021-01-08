package internal

import "github.com/shopspring/decimal"

type Wallet struct {
	Coins  decimal.Decimal // 数币
	Cash   decimal.Decimal // 资金
	Orders []*Order
}

func (p *Wallet) Buy(timestamp int64, price decimal.Decimal) {
	p.Orders = append(p.Orders, &Order{
		Timestamp: timestamp,
		Cash:      p.Cash,
		Price:     price,
		IsBuy:     true,
	})
	p.Coins = p.Cash.Div(price)
	p.Cash = decimal.Decimal{}
}

func (p *Wallet) Sell(timestamp int64, price decimal.Decimal) {
	p.Cash = p.Coins.Mul(price)
	p.Orders = append(p.Orders, &Order{
		Timestamp: timestamp,
		Cash:      p.Cash,
		Price:     price,
		IsBuy:     false,
	})
	p.Coins = decimal.Decimal{}
}

type Order struct {
	Timestamp int64
	Cash      decimal.Decimal
	Price     decimal.Decimal
	IsBuy     bool
}
