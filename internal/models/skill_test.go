package models

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestSkillMetadata_Unmarshal(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  SkillMetadata
	}{
		{
			name: "all fields present",
			input: `
version: 1.2
name: Git Expert
description: Standards for Git branching and commits
`,
			want: SkillMetadata{
				Version:     1.2,
				Name:        "Git Expert",
				Description: "Standards for Git branching and commits",
			},
		},
		{
			name: "description absent defaults to empty string",
			input: `
version: 2
name: Docker Dev
`,
			want: SkillMetadata{
				Version:     2,
				Name:        "Docker Dev",
				Description: "",
			},
		},
		{
			name:  "all fields absent give zero values",
			input: `{}`,
			want:  SkillMetadata{},
		},
		{
			name: "integer version parses as float64",
			input: `
version: 3
name: K8s Ops
description: Kubernetes operations skill
`,
			want: SkillMetadata{
				Version:     3.0,
				Name:        "K8s Ops",
				Description: "Kubernetes operations skill",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got SkillMetadata
			if err := yaml.Unmarshal([]byte(tt.input), &got); err != nil {
				t.Fatalf("yaml.Unmarshal error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestSkillMetadata_RoundTrip(t *testing.T) {
	original := SkillMetadata{
		Version:     1.5,
		Name:        "My Skill",
		Description: "A test description",
	}

	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("yaml.Marshal error: %v", err)
	}

	var got SkillMetadata
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("yaml.Unmarshal error: %v", err)
	}

	if got != original {
		t.Errorf("round-trip failed: got %+v, want %+v", got, original)
	}
}
