package observability

import "testing"

func TestModuleNameFromFunction(t *testing.T) {
	tests := map[string]string{
		"github.com/OSShip/gateway/internal/proxy.(*Handler).ServeHTTP": "github.com/OSShip/gateway/internal/proxy",
		"github.com/OSShip/utils/observability.InitLogger":              "github.com/OSShip/utils/observability",
		"main.main": "main",
		"":          "",
	}

	for fn, want := range tests {
		if got := moduleNameFromFunction(fn); got != want {
			t.Fatalf("moduleNameFromFunction(%q) = %q, want %q", fn, got, want)
		}
	}
}

func TestFunctionNameFromFunction(t *testing.T) {
	tests := map[string]string{
		"github.com/OSShip/gateway/internal/proxy.(*Handler).ServeHTTP": "ServeHTTP",
		"github.com/OSShip/utils/observability.InitLogger":              "InitLogger",
		"main.main": "main",
		"":          "",
	}

	for fn, want := range tests {
		if got := functionNameFromFunction(fn); got != want {
			t.Fatalf("functionNameFromFunction(%q) = %q, want %q", fn, got, want)
		}
	}
}
