package gpc

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/api/androidpublisher/v3"
)

const maxBatchGetOrders = 1000

func (c *Client) GetOrder(ctx context.Context, packageName, orderID string) (OrderInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return OrderInfo{}, fmt.Errorf("package name is required")
	}
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return OrderInfo{}, fmt.Errorf("order id is required")
	}
	if c == nil || c.service == nil {
		return OrderInfo{}, ErrInvalidCredentials
	}

	order, err := c.service.Orders.Get(packageName, orderID).Context(ctx).Do()
	if err != nil {
		return OrderInfo{}, mapGoogleAPIError(err)
	}
	return orderInfoFromOrder(order), nil
}

func (c *Client) BatchGetOrders(ctx context.Context, packageName string, orderIDs []string) ([]OrderInfo, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return nil, fmt.Errorf("package name is required")
	}
	orderIDs = normalizeOrderIDs(orderIDs)
	if len(orderIDs) == 0 {
		return nil, fmt.Errorf("at least one order id is required")
	}
	if len(orderIDs) > maxBatchGetOrders {
		return nil, fmt.Errorf("at most %d order ids are allowed", maxBatchGetOrders)
	}
	if c == nil || c.service == nil {
		return nil, ErrInvalidCredentials
	}

	resp, err := c.service.Orders.Batchget(packageName).OrderIds(orderIDs...).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleAPIError(err)
	}

	orders := make([]OrderInfo, 0, len(resp.Orders))
	for _, order := range resp.Orders {
		orders = append(orders, orderInfoFromOrder(order))
	}
	return orders, nil
}

func (c *Client) RefundOrder(ctx context.Context, packageName, orderID string, revoke bool) error {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return fmt.Errorf("package name is required")
	}
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return fmt.Errorf("order id is required")
	}
	if c == nil || c.service == nil {
		return ErrInvalidCredentials
	}

	if err := c.service.Orders.Refund(packageName, orderID).Revoke(revoke).Context(ctx).Do(); err != nil {
		return mapGoogleAPIError(err)
	}
	return nil
}

func normalizeOrderIDs(orderIDs []string) []string {
	seen := make(map[string]struct{}, len(orderIDs))
	out := make([]string, 0, len(orderIDs))
	for _, orderID := range orderIDs {
		trimmed := strings.TrimSpace(orderID)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func orderInfoFromOrder(order *androidpublisher.Order) OrderInfo {
	if order == nil {
		return OrderInfo{}
	}

	productIDs := make([]string, 0, len(order.LineItems))
	for _, item := range order.LineItems {
		if item == nil {
			continue
		}
		productID := strings.TrimSpace(item.ProductId)
		if productID != "" {
			productIDs = append(productIDs, productID)
		}
	}

	buyerCountry := ""
	if order.BuyerAddress != nil {
		buyerCountry = order.BuyerAddress.BuyerCountry
	}

	return OrderInfo{
		OrderID:                 order.OrderId,
		PurchaseToken:           order.PurchaseToken,
		State:                   order.State,
		SalesChannel:            order.SalesChannel,
		CreateTime:              order.CreateTime,
		LastEventTime:           order.LastEventTime,
		BuyerCountry:            buyerCountry,
		LineItemCount:           len(order.LineItems),
		LineItemProductIDs:      productIDs,
		Total:                   moneyInfoFromMoney(order.Total),
		Tax:                     moneyInfoFromMoney(order.Tax),
		DeveloperRevenueInBuyer: moneyInfoFromMoney(order.DeveloperRevenueInBuyerCurrency),
	}
}

func moneyInfoFromMoney(money *androidpublisher.Money) MoneyInfo {
	if money == nil {
		return MoneyInfo{}
	}
	return MoneyInfo{
		CurrencyCode: money.CurrencyCode,
		Units:        money.Units,
		Nanos:        money.Nanos,
	}
}
