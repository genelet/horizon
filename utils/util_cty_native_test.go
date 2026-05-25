package utils

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

func TestCtyToNativePreservesFractionalNumbers(t *testing.T) {
	tests := []struct {
		name string
		val  cty.Value
	}{
		{name: "positive", val: cty.NumberFloatVal(43.6532)},
		{name: "negative", val: cty.NumberFloatVal(-79.3832)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CtyToNative(tt.val)
			if err != nil {
				t.Fatalf("CtyToNative() error = %v", err)
			}
			switch got.(type) {
			case float32, float64:
			default:
				t.Fatalf("CtyToNative() type = %T, want float32 or float64", got)
			}
		})
	}
}

func TestCtyToNativePreservesIntegerNumbers(t *testing.T) {
	got, err := CtyToNative(cty.NumberIntVal(42))
	if err != nil {
		t.Fatalf("CtyToNative() error = %v", err)
	}
	if got != 42 {
		t.Fatalf("CtyToNative() = %#v (%T), want 42 (int)", got, got)
	}
}

func TestExpressionToCtyReturnsNestedObjectWithNumericStringKey(t *testing.T) {
	node := NewEvalContext(nil)
	weather := map[string]any{
		"received_body": map[string]any{
			"rain": map[string]any{"1h": 0.14},
			"name": "Downtown Toronto",
		},
	}
	cv, err := NativeToCty(weather)
	if err != nil {
		t.Fatalf("NativeToCty() error = %v", err)
	}
	node.AddItem("retrieve_weather", cv)

	expr, diags := hclsyntax.ParseExpression([]byte("retrieve_weather.received_body"), "expr.hcl", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		t.Fatalf("ParseExpression() diagnostics = %v", diags.Error())
	}
	got, err := ExpressionToCty(node.GetRef(), node, expr)
	if err != nil {
		t.Fatalf("ExpressionToCty() error = %v", err)
	}
	native, err := CtyToNative(got)
	if err != nil {
		t.Fatalf("CtyToNative() error = %v", err)
	}
	body, ok := native.(map[string]any)
	if !ok {
		t.Fatalf("native type = %T, want map[string]any", native)
	}
	if body["name"] != "Downtown Toronto" {
		t.Fatalf("name = %#v", body["name"])
	}
}

func TestExpressionToCtyReturnsTreeNestedObjectWithMixedWeatherShape(t *testing.T) {
	node := NewEvalContext(nil)
	receivedBody := node.AddNode("retrieve_weather").AddNode("received_body")
	weather := map[string]any{
		"base": "stations",
		"rain": map[string]any{"1h": 0.14},
		"weather": []any{map[string]any{
			"description": "light rain",
			"id":          500,
			"main":        "Rain",
		}},
		"main": map[string]any{
			"temp":     285.78,
			"humidity": 95,
		},
		"name": "Downtown Toronto",
	}
	for key, value := range weather {
		cv, err := NativeToCty(value)
		if err != nil {
			t.Fatalf("NativeToCty(%s) error = %v", key, err)
		}
		receivedBody.AddItem(key, cv)
	}

	expr, diags := hclsyntax.ParseExpression([]byte("retrieve_weather.received_body"), "expr.hcl", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		t.Fatalf("ParseExpression() diagnostics = %v", diags.Error())
	}
	got, err := ExpressionToCty(node.GetRef(), node, expr)
	if err != nil {
		t.Fatalf("ExpressionToCty() error = %v", err)
	}
	native, err := CtyToNative(got)
	if err != nil {
		t.Fatalf("CtyToNative() error = %v", err)
	}
	body, ok := native.(map[string]any)
	if !ok {
		t.Fatalf("native type = %T, want map[string]any", native)
	}
	if body["name"] != "Downtown Toronto" {
		t.Fatalf("name = %#v", body["name"])
	}
}

func TestExpressionToCtyReturnsProviderOutputObject(t *testing.T) {
	node := NewEvalContext(nil)
	weather := map[string]any{
		"base": "stations",
		"rain": map[string]any{"1h": 0.14},
		"weather": []any{map[string]any{
			"description": "light rain",
			"id":          500,
			"main":        "Rain",
		}},
		"main": map[string]any{
			"temp":     285.78,
			"humidity": 95,
		},
		"name": "Downtown Toronto",
	}
	cv, err := NativeToCty(weather)
	if err != nil {
		t.Fatalf("NativeToCty() error = %v", err)
	}
	node.AddNode("retrieve_weather").AddItem("received_body", cv)

	expr, diags := hclsyntax.ParseExpression([]byte("retrieve_weather.received_body"), "expr.hcl", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		t.Fatalf("ParseExpression() diagnostics = %v", diags.Error())
	}
	got, err := ExpressionToCty(node.GetRef(), node, expr)
	if err != nil {
		t.Fatalf("ExpressionToCty() error = %v", err)
	}
	native, err := CtyToNative(got)
	if err != nil {
		t.Fatalf("CtyToNative() error = %v", err)
	}
	body, ok := native.(map[string]any)
	if !ok {
		t.Fatalf("native type = %T, want map[string]any", native)
	}
	if body["name"] != "Downtown Toronto" {
		t.Fatalf("name = %#v", body["name"])
	}
}

func TestExpressionToCtyReturnsProviderOutputObjectWithCurrentWeatherShape(t *testing.T) {
	node := NewEvalContext(nil)
	weather := map[string]any{
		"base": "stations",
		"clouds": map[string]any{
			"all": float64(100),
		},
		"cod": float64(200),
		"coord": map[string]any{
			"lat": float64(43.6535),
			"lon": float64(-79.3839),
		},
		"dt": float64(1779678689),
		"id": float64(6167863),
		"main": map[string]any{
			"feels_like": float64(285.58),
			"grnd_level": float64(1004),
			"humidity":   float64(95),
			"pressure":   float64(1017),
			"sea_level":  float64(1017),
			"temp":       float64(285.78),
			"temp_max":   float64(286.69),
			"temp_min":   float64(284.31),
		},
		"name": "Downtown Toronto",
		"rain": map[string]any{
			"1h": float64(0.14),
		},
		"sys": map[string]any{
			"country": "CA",
			"id":      float64(718),
			"sunrise": float64(1779615857),
			"sunset":  float64(1779669888),
			"type":    float64(1),
		},
		"timezone":   float64(-14400),
		"visibility": float64(10000),
		"weather": []any{map[string]any{
			"description": "light rain",
			"icon":        "10n",
			"id":          float64(500),
			"main":        "Rain",
		}},
		"wind": map[string]any{
			"deg":   float64(79),
			"gust":  float64(4.67),
			"speed": float64(2.65),
		},
	}
	cv, err := NativeToCty(weather)
	if err != nil {
		t.Fatalf("NativeToCty() error = %v", err)
	}
	node.AddNode("retrieve_weather").AddItem("received_body", cv)

	expr, diags := hclsyntax.ParseExpression([]byte("retrieve_weather.received_body"), "expr.hcl", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		t.Fatalf("ParseExpression() diagnostics = %v", diags.Error())
	}
	got, err := ExpressionToCty(node.GetRef(), node, expr)
	if err != nil {
		t.Fatalf("ExpressionToCty() error = %v", err)
	}
	native, err := CtyToNative(got)
	if err != nil {
		t.Fatalf("CtyToNative() error = %v", err)
	}
	body, ok := native.(map[string]any)
	if !ok {
		t.Fatalf("native type = %T, want map[string]any", native)
	}
	if body["name"] != "Downtown Toronto" {
		t.Fatalf("name = %#v", body["name"])
	}
}

func TestExpressionToCtyReturnsProviderOutputWithDuplicateChildNode(t *testing.T) {
	node := NewEvalContext(nil)
	weather := map[string]any{
		"name": "Downtown Toronto",
		"rain": map[string]any{"1h": float64(0.14)},
	}
	cv, err := NativeToCty(weather)
	if err != nil {
		t.Fatalf("NativeToCty() error = %v", err)
	}
	stepNode := node.AddNode("retrieve_weather")
	stepNode.AddItem("received_body", cv)
	receivedBodyNode := stepNode.AddNode("received_body")
	for key, value := range weather {
		item, err := NativeToCty(value)
		if err != nil {
			t.Fatalf("NativeToCty(%s) error = %v", key, err)
		}
		receivedBodyNode.AddItem(key, item)
	}

	expr, diags := hclsyntax.ParseExpression([]byte("retrieve_weather.received_body"), "expr.hcl", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		t.Fatalf("ParseExpression() diagnostics = %v", diags.Error())
	}
	got, err := ExpressionToCty(node.GetRef(), node, expr)
	if err != nil {
		t.Fatalf("ExpressionToCty() error = %v", err)
	}
	native, err := CtyToNative(got)
	if err != nil {
		t.Fatalf("CtyToNative() error = %v", err)
	}
	body, ok := native.(map[string]any)
	if !ok {
		t.Fatalf("native type = %T, want map[string]any", native)
	}
	if body["name"] != "Downtown Toronto" {
		t.Fatalf("name = %#v", body["name"])
	}
}
