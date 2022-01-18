package terrabot

import "fmt"

type OrderType string
type SideType string
type PositionSideType string

const (
	OrderLimit      OrderType = "limit"
	OrderMarket     OrderType = "market"
	OrderStopMarket OrderType = "stop"
	OrderTrailing   OrderType = "trailing"

	SideBuy  SideType = "buy"
	SideSell SideType = "sell"

	PositionSideNet   PositionSideType = "NET"
	PositionSideLong  PositionSideType = "LONG"
	PositionSideShort PositionSideType = "SHORT"
)

type Order struct {
	Type         OrderType        `json:"type"`
	Symbol       string           `json:"symbol"`
	Side         SideType         `json:"side"`
	PositionSide PositionSideType `json:"positionSide"`
	Amount       float64          `json:"amount"`
	ID           string           `json:"id"`
	Price        float64          `json:"price"`
	TriggerPrice float64          `json:"stopPrice"`
	CallbackRate float64          `json:"callbackRate"`
	GridNumber   int64            `json:"gridNumber"`
	ReduceOnly   bool             `json:"reduceOnly"` // this is needed for the net position mode
	Tag          string           `json:"tag"`
}

type OpenOrders map[string]Order

func (o OpenOrders) String() string {
	return fmt.Sprint(len(o))
}

func (order Order) String() string {
	switch order.Type {

	case OrderLimit:
		return fmt.Sprintf("type %s, symbol %s, side %s, position side %s, price %f, amount %f, reduceOnly %t",
			order.Type, order.Symbol, order.Side, order.PositionSide, order.Price, order.Amount, order.ReduceOnly)

	case OrderMarket:
		return fmt.Sprintf("type %s, symbol %s, side %s, position side %s, amount %f",
			order.Type, order.Symbol, order.Side, order.PositionSide, order.Amount)

	case OrderStopMarket:
		return fmt.Sprintf("type %s, symbol %s, side %s, position side %s, trigger price %f, amount %f, ID %s",
			order.Type, order.Symbol, order.Side, order.PositionSide, order.TriggerPrice, order.Amount, order.ID)

	case OrderTrailing:
		return fmt.Sprintf("type %s, symbol %s, side %s, position side %s, trigger price %f, amount %f, callback rate %f",
			order.Type, order.Symbol, order.Side, order.PositionSide, order.TriggerPrice, order.Amount, order.CallbackRate)

	default:
		return "order type not recognized"
	}
}

func NewOrderLimit(symbol string, side SideType, positionSide PositionSideType, amount float64, price float64) *Order {
	return &Order{
		Type:         OrderLimit,
		Symbol:       symbol,
		Side:         side,
		PositionSide: positionSide,
		Amount:       amount,
		Price:        price,
		ID:           "",
	}
}

func NewOrderMarket(symbol string, side SideType, positionSide PositionSideType, amount float64) *Order {
	return &Order{
		Type:         OrderMarket,
		Symbol:       symbol,
		Side:         side,
		PositionSide: positionSide,
		Amount:       amount,
	}
}

func NewOrderStopMarket(symbol string, side SideType, positionSide PositionSideType, amount float64, stopPrice float64) *Order {
	return &Order{
		Type:         OrderStopMarket,
		Symbol:       symbol,
		Side:         side,
		PositionSide: positionSide,
		Amount:       amount,
		TriggerPrice: stopPrice,
		ID:           "",
	}
}

func NewOrderTrailing(symbol string, side SideType, positionSide PositionSideType, amount float64, activationPrice float64, callbackRate float64) *Order {
	return &Order{
		Type:         OrderTrailing,
		Symbol:       symbol,
		Side:         side,
		PositionSide: positionSide,
		Amount:       amount,
		TriggerPrice: activationPrice,
		CallbackRate: callbackRate,
		ID:           "",
	}
}

type Position struct {
	Symbol       string           `json:"symbol"`
	PositionSide PositionSideType `json:"positionSide"`
	EntryPrice   float64          `json:"entryPrice"`
	Size         float64          `json:"size"`
	MarkPrice    float64          `json:"markPrice"`
}

func (p Position) String() string {

	return fmt.Sprintf("symbol %s, position side %s, entry price %.4f, size %.4f",
		p.Symbol, string(p.PositionSide), p.EntryPrice, p.Size)

}
