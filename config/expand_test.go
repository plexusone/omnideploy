package config

import (
	"os"
	"testing"
)

func TestExpandEnv(t *testing.T) {
	t.Setenv("EXPAND_TEST_VAR", "hello")
	t.Setenv("EXPAND_TEST_EMPTY", "")
	os.Unsetenv("EXPAND_TEST_UNSET")

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple var", "value: ${EXPAND_TEST_VAR}", "value: hello"},
		{"var with default, set", "value: ${EXPAND_TEST_VAR:-fallback}", "value: hello"},
		{"var with default, unset", "value: ${EXPAND_TEST_UNSET:-fallback}", "value: fallback"},
		{"var with default, empty env value", "value: ${EXPAND_TEST_EMPTY:-fallback}", "value: fallback"},
		{"unset var, no default", "value: ${EXPAND_TEST_UNSET}", "value: "},
		{"escaped dollar sign", "price: $$100", "price: $100"},
		{"no expansion needed", "plain: text", "plain: text"},
		{"multiple in one line", "a: ${EXPAND_TEST_VAR}, b: ${EXPAND_TEST_UNSET:-x}", "a: hello, b: x"},
		{"var name with underscore and digits", "value: ${EXPAND_TEST_VAR}_suffix", "value: hello_suffix"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(ExpandEnv([]byte(tt.input)))
			if got != tt.want {
				t.Errorf("ExpandEnv(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
