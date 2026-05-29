package model

import "testing"

func TestAllPlatforms(t *testing.T) {
	got := AllPlatforms()
	want := []string{
		PlatformAnthropic,
		PlatformOpenAI,
		PlatformGemini,
		PlatformAntigravity,
		PlatformOpenAICompatible,
		PlatformAnthropicCompatible,
	}
	if len(got) != len(want) {
		t.Fatalf("AllPlatforms length = %d, want %d", len(got), len(want))
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("AllPlatforms[%d] = %q, want %q", index, got[index], want[index])
		}
	}
}

func TestErrorPassthroughRuleValidate(t *testing.T) {
	responseCode := 418
	message := "blocked"

	valid := &ErrorPassthroughRule{
		Name:            "mask unsafe upstream errors",
		Enabled:         true,
		MatchMode:       MatchModeAny,
		ErrorCodes:      []int{400},
		PassthroughCode: false,
		ResponseCode:    &responseCode,
		PassthroughBody: false,
		CustomMessage:   &message,
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}

	tests := []struct {
		name string
		rule ErrorPassthroughRule
		want string
	}{
		{
			name: "missing name",
			rule: ErrorPassthroughRule{
				MatchMode:       MatchModeAny,
				ErrorCodes:      []int{400},
				PassthroughCode: true,
				PassthroughBody: true,
			},
			want: "name: name is required",
		},
		{
			name: "invalid match mode",
			rule: ErrorPassthroughRule{
				Name:            "x",
				MatchMode:       "invalid",
				ErrorCodes:      []int{400},
				PassthroughCode: true,
				PassthroughBody: true,
			},
			want: "match_mode: match_mode must be 'any' or 'all'",
		},
		{
			name: "missing conditions",
			rule: ErrorPassthroughRule{
				Name:            "x",
				MatchMode:       MatchModeAll,
				PassthroughCode: true,
				PassthroughBody: true,
			},
			want: "conditions: at least one error_code or keyword is required",
		},
		{
			name: "missing response code when passthrough code disabled",
			rule: ErrorPassthroughRule{
				Name:            "x",
				MatchMode:       MatchModeAny,
				ErrorCodes:      []int{400},
				PassthroughCode: false,
				PassthroughBody: true,
			},
			want: "response_code: response_code is required when passthrough_code is false",
		},
		{
			name: "missing custom message when passthrough body disabled",
			rule: ErrorPassthroughRule{
				Name:            "x",
				MatchMode:       MatchModeAny,
				ErrorCodes:      []int{400},
				PassthroughCode: true,
				PassthroughBody: false,
			},
			want: "custom_message: custom_message is required when passthrough_body is false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rule.Validate()
			if err == nil {
				t.Fatalf("Validate() error = nil, want %q", tt.want)
			}
			if err.Error() != tt.want {
				t.Fatalf("Validate() error = %q, want %q", err.Error(), tt.want)
			}
		})
	}
}
