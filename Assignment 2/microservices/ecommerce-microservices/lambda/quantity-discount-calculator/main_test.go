package main

import (
	"context"
	"encoding/json"
	"math"
	"strings"
	"testing"
)

func TestHandler(t *testing.T) {
	tests := []struct {
		name          string
		request       Request
		wantStatus    int
		wantError     string
		wantUnitPrice Money
		wantSubtotal  Money
		wantDiscount  Money
		wantTotal     Money
		wantRate      float64
	}{
		{
			name:          "quantity 1 has no discount",
			request:       Request{ProductID: "P001", Quantity: 1, UnitPrice: 185000},
			wantStatus:    200,
			wantUnitPrice: 185000,
			wantSubtotal:  185000,
			wantDiscount:  0,
			wantTotal:     185000,
		},
		{
			name:          "quantity 4 is the no discount boundary",
			request:       Request{ProductID: "P002", Quantity: 4, UnitPrice: 7200},
			wantStatus:    200,
			wantUnitPrice: 7200,
			wantSubtotal:  28800,
			wantDiscount:  0,
			wantTotal:     28800,
		},
		{
			name:          "quantity 5 is the discount boundary",
			request:       Request{ProductID: "P001", Quantity: 5, UnitPrice: 185000},
			wantStatus:    200,
			wantUnitPrice: 185000,
			wantSubtotal:  925000,
			wantDiscount:  92500,
			wantTotal:     832500,
			wantRate:      0.10,
		},
		{
			name:          "large valid quantity",
			request:       Request{ProductID: "P003", Quantity: 10000, UnitPrice: 2600},
			wantStatus:    200,
			wantUnitPrice: 2600,
			wantSubtotal:  26000000,
			wantDiscount:  2600000,
			wantTotal:     23400000,
			wantRate:      0.10,
		},
		{
			name:          "decimal price uses half-up cents",
			request:       Request{ProductID: "P004", Quantity: 5, UnitPrice: 10.125},
			wantStatus:    200,
			wantUnitPrice: 10.13,
			wantSubtotal:  50.65,
			wantDiscount:  5.07,
			wantTotal:     45.58,
			wantRate:      0.10,
		},
		{name: "empty productId", request: Request{Quantity: 1, UnitPrice: 1}, wantStatus: 400, wantError: "productId is required"},
		{name: "quantity zero", request: Request{ProductID: "P001", Quantity: 0, UnitPrice: 1}, wantStatus: 400, wantError: "quantity must be an integer between 1 and 10000"},
		{name: "negative quantity", request: Request{ProductID: "P001", Quantity: -1, UnitPrice: 1}, wantStatus: 400, wantError: "quantity must be an integer between 1 and 10000"},
		{name: "quantity over maximum", request: Request{ProductID: "P001", Quantity: 10001, UnitPrice: 1}, wantStatus: 400, wantError: "quantity must be an integer between 1 and 10000"},
		{name: "zero unitPrice", request: Request{ProductID: "P001", Quantity: 1, UnitPrice: 0}, wantStatus: 400, wantError: "unitPrice must be a finite number greater than zero"},
		{name: "negative unitPrice", request: Request{ProductID: "P001", Quantity: 1, UnitPrice: -1}, wantStatus: 400, wantError: "unitPrice must be a finite number greater than zero"},
		{name: "NaN unitPrice", request: Request{ProductID: "P001", Quantity: 1, UnitPrice: math.NaN()}, wantStatus: 400, wantError: "unitPrice must be a finite number greater than zero"},
		{name: "infinite unitPrice", request: Request{ProductID: "P001", Quantity: 1, UnitPrice: math.Inf(1)}, wantStatus: 400, wantError: "unitPrice must be a finite number greater than zero"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response, err := handler(context.Background(), test.request)
			if err != nil {
				t.Fatalf("handler returned unexpected error: %v", err)
			}
			if response.StatusCode != test.wantStatus {
				t.Fatalf("statusCode = %d, want %d", response.StatusCode, test.wantStatus)
			}

			if test.wantStatus == 400 {
				body, ok := response.Body.(ErrorBody)
				if !ok {
					t.Fatalf("body type = %T, want ErrorBody", response.Body)
				}
				if body.Error != test.wantError {
					t.Fatalf("error = %q, want %q", body.Error, test.wantError)
				}
				return
			}

			body, ok := response.Body.(SuccessBody)
			if !ok {
				t.Fatalf("body type = %T, want SuccessBody", response.Body)
			}
			if body.ProductID != test.request.ProductID || body.Quantity != test.request.Quantity {
				t.Fatalf("identity fields = %#v, want productId=%q quantity=%d", body, test.request.ProductID, test.request.Quantity)
			}
			if body.UnitPrice != test.wantUnitPrice || body.Subtotal != test.wantSubtotal || body.Discount != test.wantDiscount || body.Total != test.wantTotal || body.DiscountRate != test.wantRate {
				t.Fatalf("amounts = %#v, want unitPrice=%v subtotal=%v discountRate=%v discount=%v total=%v", body, test.wantUnitPrice, test.wantSubtotal, test.wantRate, test.wantDiscount, test.wantTotal)
			}
		})
	}
}

func TestResponseJSONUsesTwoDecimalMoneyValues(t *testing.T) {
	response, err := handler(context.Background(), Request{ProductID: "P001", Quantity: 5, UnitPrice: 185000})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	for _, expected := range []string{`"unitPrice":185000.00`, `"subtotal":925000.00`, `"discount":92500.00`, `"total":832500.00`} {
		if !strings.Contains(text, expected) {
			t.Fatalf("JSON = %s, missing %s", text, expected)
		}
	}
}
