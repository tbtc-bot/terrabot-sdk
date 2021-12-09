package terrabot

import (
	"fmt"
	"math"
)

type StrategyType string
type StrategyStatus string
type OrderBaseType string

const (
	StrategyDummy      StrategyType = "Dummy"
	StrategyMartingala StrategyType = "Martingala"

	StatusStart    StrategyStatus = "start"
	StatusStop     StrategyStatus = "stop"
	StatusHardStop StrategyStatus = "hardStop"
	StatusSoftStop StrategyStatus = "softStop"

	OrderBaseTypePerc OrderBaseType = "percentage"
	OrderBaseTypeFix  OrderBaseType = "fix"
)

type StrategyParameters struct {
	GridOrders             uint          `json:"gridOrders"`
	GridStep               float64       `json:"gridStep"`
	OrderBaseType          OrderBaseType `json:"orderBaseType"`
	StepFactor             float64       `json:"stepFactor"`
	OrderSize              float64       `json:"orderSize"`
	OrderType              string        `json:"orderType"`
	OrderFactor            float64       `json:"orderFactor"`
	TakeStep               float64       `json:"takeStep"`
	TakeStepLimit          float64       `json:"takeStepLimit"`
	TakeStepLimitThreshold float64       `json:"takeStepLimitThreshold"`
	StopLoss               float64       `json:"stopLoss"`
	CallBackRate           float64       `json:"callBack"`
}

func (sp StrategyParameters) String() string {
	return fmt.Sprintf("[GridOrders:%d, GridSteps:%.2f, OrderBaseType:%s, StepFactor :%.2f, OrderSize:%.2f, OrderFactor:%.2f, TakeStep:%.2f, StopLoss:%.2f, CallBackRate:%.2f, TakeStepLimit:%.2f, TakeStepLimitThreshold:%.2f]",
		sp.GridOrders, sp.GridStep, sp.OrderBaseType, sp.StepFactor, sp.OrderSize, sp.OrderFactor, sp.TakeStep, sp.StopLoss, sp.CallBackRate, sp.TakeStepLimit, sp.TakeStepLimitThreshold)
}

type Strategy struct {
	Type         StrategyType       `json:"type"`
	Symbol       string             `json:"symbol"`
	PositionSide PositionSideType   `json:"positionSide"`
	Status       StrategyStatus     `json:"status"`
	Parameters   StrategyParameters `json:"parameters"`
}

func NewStrategy(strategyType StrategyType, symbol string, positionSide PositionSideType, parameters StrategyParameters) *Strategy {
	return &Strategy{
		Type:         strategyType,
		Symbol:       symbol,
		PositionSide: positionSide,
		Status:       "",
		Parameters:   parameters,
	}
}

func (s Strategy) String() string {
	return fmt.Sprintf("type %s, symbol %s, position side %s, status %s, parameters %s",
		s.Type, s.Symbol, s.PositionSide, s.Status, s.Parameters.String())
}

func (s *Strategy) GridOrders(balance float64, startPrice float64) ([]*Order, error) {
	orders := []*Order{}
	p0 := startPrice

	// start size
	var s0 float64
	if s.Parameters.OrderBaseType == OrderBaseTypePerc {
		s0 = (balance / startPrice) * (s.Parameters.OrderSize / 100)
	} else if s.Parameters.OrderBaseType == OrderBaseTypeFix {
		s0 = s.Parameters.OrderSize / startPrice
	} else {
		return nil, fmt.Errorf("order base type %s not recognized", s.Parameters.OrderBaseType)
	}

	if s.PositionSide == PositionSideLong || s.PositionSide == PositionSideNet {

		// first grid
		p_1 := p0 * (1 - s.Parameters.GridStep/100)
		s_1 := s0
		p_2 := p0
		order := NewOrderLimit(s.Symbol, SideBuy, PositionSideLong, s_1, p_1)
		order.GridNumber = 1
		orders = append(orders, order)

		// other grids
		for i := 2; i < int(s.Parameters.GridOrders)+1; i++ {
			p_i := p_1 - (p_2-p_1)*s.Parameters.StepFactor
			s_i := s_1 * s.Parameters.OrderFactor
			order := NewOrderLimit(s.Symbol, SideBuy, PositionSideLong, s_i, p_i)
			order.GridNumber = int64(i)
			orders = append(orders, order)
			p_2 = p_1
			p_1 = p_i
			s_1 = s_i
		}

		// set stoploss
		if s.Parameters.StopLoss > 0 {
			stopLossPrice := p_1 * (1 - s.Parameters.StopLoss/100)
			stopLoss := NewOrderStopMarket(s.Symbol, SideSell, PositionSideLong, 0, stopLossPrice)
			orders = append(orders, stopLoss)
		}

	} else if s.PositionSide == PositionSideShort {

		// first grid
		p_1 := p0 * (1 + s.Parameters.GridStep/100)
		s_1 := s0
		p_2 := p0
		order := NewOrderLimit(s.Symbol, SideSell, PositionSideShort, s_1, p_1)
		order.GridNumber = 1
		orders = append(orders, order)

		// other grids
		for i := 2; i < int(s.Parameters.GridOrders)+1; i++ {
			p_i := p_1 + (p_1-p_2)*s.Parameters.StepFactor
			s_i := s_1 * s.Parameters.OrderFactor
			order := NewOrderLimit(s.Symbol, SideSell, PositionSideShort, s_i, p_i)
			order.GridNumber = int64(i)
			orders = append(orders, order)
			p_2 = p_1
			p_1 = p_i
			s_1 = s_i
		}

		// set stoploss
		if s.Parameters.StopLoss > 0 {
			stopLossPrice := p_1 * (1 + s.Parameters.StopLoss/100)
			stopLoss := NewOrderStopMarket(s.Symbol, SideBuy, PositionSideShort, 0, stopLossPrice)
			orders = append(orders, stopLoss)
		}
	} else {
		return nil, fmt.Errorf("position side %s not recognized", s.PositionSide)
	}
	return orders, nil
}

func (s *Strategy) TakeProfitOrder(position Position, gridSize float64) (*Order, error) {
	var TakeStep float64
	if math.Abs(position.Size) >= gridSize*s.Parameters.TakeStepLimitThreshold && s.Parameters.TakeStepLimitThreshold > 0 && gridSize > 0 {
		// use TakeStepLimit if the position size is greater than the grid size * TakeStepLimitThreshold
		TakeStep = s.Parameters.TakeStepLimit
	} else {
		TakeStep = s.Parameters.TakeStep
	}

	if s.Parameters.CallBackRate < 0.1 {
		// limit order
		if s.PositionSide == PositionSideLong || s.PositionSide == PositionSideNet {
			takeProfitPrice := position.EntryPrice * (1 + TakeStep/100)
			return NewOrderLimit(s.Symbol, SideSell, PositionSideLong, position.Size, takeProfitPrice), nil

		} else if s.PositionSide == PositionSideShort {
			takeProfitPrice := position.EntryPrice * (1 - TakeStep/100)
			return NewOrderLimit(s.Symbol, SideBuy, PositionSideShort, math.Abs(position.Size), takeProfitPrice), nil

		} else {
			return nil, fmt.Errorf("position side %s not recognized", s.PositionSide)
		}
	} else {
		// trailing profit order
		if s.PositionSide == PositionSideLong {
			takeProfitPrice := position.EntryPrice * (1 + TakeStep/100)
			return NewOrderTrailing(s.Symbol, SideSell, PositionSideLong, position.Size, takeProfitPrice, s.Parameters.CallBackRate), nil

		} else if s.PositionSide == PositionSideShort {
			takeProfitPrice := position.EntryPrice * (1 - TakeStep/100)
			return NewOrderTrailing(s.Symbol, SideBuy, PositionSideShort, math.Abs(position.Size), takeProfitPrice, s.Parameters.CallBackRate), nil

		} else {
			return nil, fmt.Errorf("position side %s not recognized", s.PositionSide)
		}
	}
}
