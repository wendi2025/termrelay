package service

import "testing"

func TestGroupAllowsModel(t *testing.T) {
	tests := []struct {
		name  string
		group *Group
		model string
		want  bool
	}{
		{name: "nil group unrestricted", group: nil, model: "cqu-default", want: true},
		{name: "disabled list unrestricted", group: &Group{ModelsListConfig: GroupModelsListConfig{Enabled: false, Models: []string{"MiniMax-M3"}}}, model: "cqu-default", want: true},
		{name: "empty list preserves legacy unrestricted behavior", group: &Group{ModelsListConfig: GroupModelsListConfig{Enabled: true}}, model: "cqu-default", want: true},
		{name: "exact allowed", group: &Group{ModelsListConfig: GroupModelsListConfig{Enabled: true, Models: []string{"cqu-default"}}}, model: "cqu-default", want: true},
		{name: "exact denied", group: &Group{ModelsListConfig: GroupModelsListConfig{Enabled: true, Models: []string{"MiniMax-M3"}}}, model: "cqu-default", want: false},
		{name: "prefix wildcard allowed", group: &Group{ModelsListConfig: GroupModelsListConfig{Enabled: true, Models: []string{"gpt-*"}}}, model: "gpt-5.6", want: true},
		{name: "prefix wildcard denied", group: &Group{ModelsListConfig: GroupModelsListConfig{Enabled: true, Models: []string{"gpt-*"}}}, model: "claude-sonnet-4", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.group.AllowsModel(tt.model); got != tt.want {
				t.Fatalf("AllowsModel(%q) = %v, want %v", tt.model, got, tt.want)
			}
		})
	}
}
