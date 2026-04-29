package mesh_test

import (
	"testing"
	"time"

	"github.com/ezzek/skill-sync/internal/mesh"
	"github.com/ezzek/skill-sync/internal/models"
)

func TestResolver_Resolve(t *testing.T) {
	now := time.Now()
	older := now.Add(-1 * time.Hour)
	newer := now.Add(1 * time.Hour)

	tests := []struct {
		name          string
		instances     []models.SkillInstance
		expectWinner  *models.SkillInstance
		expectConflict bool
		expectErr      bool
	}{
		{
			name:      "empty instances",
			instances: []models.SkillInstance{},
			expectWinner: nil,
			expectConflict: false,
			expectErr: true, // Should error on empty
		},
		{
			name: "single instance",
			instances: []models.SkillInstance{
				{TargetDir: "dir1", Hash: "aaa"},
			},
			expectWinner: &models.SkillInstance{TargetDir: "dir1", Hash: "aaa"},
			expectConflict: false,
			expectErr: false,
		},
		{
			name: "identical hashes",
			instances: []models.SkillInstance{
				{TargetDir: "dir1", Hash: "aaa", Mtime: older, Metadata: models.SkillMetadata{Version: 1}},
				{TargetDir: "dir2", Hash: "aaa", Mtime: newer, Metadata: models.SkillMetadata{Version: 2}},
			},
			expectWinner: &models.SkillInstance{TargetDir: "dir1", Hash: "aaa", Mtime: older, Metadata: models.SkillMetadata{Version: 1}},
			expectConflict: false, // If hashes match, we just pick the first
			expectErr: false,
		},
		{
			name: "decimal versions resolve correctly (1.0 < 1.1 < 1.2)",
			instances: []models.SkillInstance{
				{TargetDir: "claude", Hash: "aaa", Metadata: models.SkillMetadata{Version: 1.0}},
				{TargetDir: "cursor", Hash: "bbb", Metadata: models.SkillMetadata{Version: 1.1}},
				{TargetDir: "opencode", Hash: "ccc", Metadata: models.SkillMetadata{Version: 1.2}},
			},
			expectWinner: &models.SkillInstance{TargetDir: "opencode", Hash: "ccc", Metadata: models.SkillMetadata{Version: 1.2}},
			expectConflict: false,
			expectErr: false,
		},
		{
			name: "highest version wins",
			instances: []models.SkillInstance{
				{TargetDir: "dir1", Hash: "aaa", Metadata: models.SkillMetadata{Version: 1}},
				{TargetDir: "dir2", Hash: "bbb", Metadata: models.SkillMetadata{Version: 3}},
				{TargetDir: "dir3", Hash: "ccc", Metadata: models.SkillMetadata{Version: 2}},
			},
			expectWinner: &models.SkillInstance{TargetDir: "dir2", Hash: "bbb", Metadata: models.SkillMetadata{Version: 3}},
			expectConflict: false,
			expectErr: false,
		},
		{
			name: "newer mtime wins on identical versions",
			instances: []models.SkillInstance{
				{TargetDir: "dir1", Hash: "aaa", Mtime: older, Metadata: models.SkillMetadata{Version: 1}},
				{TargetDir: "dir2", Hash: "bbb", Mtime: newer, Metadata: models.SkillMetadata{Version: 1}},
			},
			expectWinner: &models.SkillInstance{TargetDir: "dir2", Hash: "bbb", Mtime: newer, Metadata: models.SkillMetadata{Version: 1}},
			expectConflict: false,
			expectErr: false,
		},
		{
			name: "newer mtime wins on missing versions (0)",
			instances: []models.SkillInstance{
				{TargetDir: "dir1", Hash: "aaa", Mtime: older},
				{TargetDir: "dir2", Hash: "bbb", Mtime: newer},
			},
			expectWinner: &models.SkillInstance{TargetDir: "dir2", Hash: "bbb", Mtime: newer},
			expectConflict: false,
			expectErr: false,
		},
		{
			name: "conflict: different hashes, identical version and mtime",
			instances: []models.SkillInstance{
				{TargetDir: "dir1", Hash: "aaa", Mtime: now, Metadata: models.SkillMetadata{Version: 2}},
				{TargetDir: "dir2", Hash: "bbb", Mtime: now, Metadata: models.SkillMetadata{Version: 2}},
			},
			expectWinner: nil,
			expectConflict: true,
			expectErr: false,
		},
		{
			name: "conflict: identical versions, one has no mtime (somehow), both zero mtimes",
			instances: []models.SkillInstance{
				{TargetDir: "dir1", Hash: "aaa", Mtime: time.Time{}, Metadata: models.SkillMetadata{Version: 2}},
				{TargetDir: "dir2", Hash: "bbb", Mtime: time.Time{}, Metadata: models.SkillMetadata{Version: 2}},
			},
			expectWinner: nil,
			expectConflict: true,
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := mesh.NewDefaultResolver()
			winner, conflicts, err := resolver.Resolve(tt.instances)

			if (err != nil) != tt.expectErr {
				t.Fatalf("expected error: %v, got %v", tt.expectErr, err)
			}

			if conflicts != tt.expectConflict {
				t.Errorf("expected conflict: %v, got %v", tt.expectConflict, conflicts)
			}

			if winner == nil && tt.expectWinner != nil {
				t.Fatalf("expected winner %v, got nil", tt.expectWinner)
			}
			if winner != nil && tt.expectWinner == nil {
				t.Fatalf("expected nil winner, got %v", winner)
			}

			if winner != nil && tt.expectWinner != nil {
				if winner.Hash != tt.expectWinner.Hash {
					t.Errorf("expected winner hash %q, got %q", tt.expectWinner.Hash, winner.Hash)
				}
				if winner.TargetDir != tt.expectWinner.TargetDir {
					t.Errorf("expected winner target dir %q, got %q", tt.expectWinner.TargetDir, winner.TargetDir)
				}
			}
		})
	}
}
