package mesh

import (
	"errors"

	"github.com/ezzek/skill-sync/internal/models"
)

// Resolver determines the winning skill version.
type Resolver interface {
	Resolve(instances []models.SkillInstance) (winner *models.SkillInstance, conflicts bool, err error)
}

type defaultResolver struct{}

func NewDefaultResolver() Resolver {
	return &defaultResolver{}
}

func (r *defaultResolver) Resolve(instances []models.SkillInstance) (*models.SkillInstance, bool, error) {
	if len(instances) == 0 {
		return nil, false, errors.New("no instances provided")
	}

	// First check: if all instances have the identical hash, there is no conflict
	// and no resolution needed. Just return the first one.
	allSame := true
	firstHash := instances[0].Hash
	for _, inst := range instances[1:] {
		if inst.Hash != firstHash {
			allSame = false
			break
		}
	}
	if allSame {
		// Return a copy so we don't return a pointer to the slice element directly if it might escape
		winner := instances[0]
		return &winner, false, nil
	}

	var winner *models.SkillInstance
	conflict := false

	for i := range instances {
		inst := &instances[i]

		if winner == nil {
			winner = inst
			continue
		}

		// Precedence: Highest Version -> Newest Mtime
		if inst.Metadata.Version > winner.Metadata.Version {
			winner = inst
			conflict = false // A clear winner resolves any previous conflicts
		} else if inst.Metadata.Version == winner.Metadata.Version {
			if inst.Mtime.After(winner.Mtime) {
				winner = inst
				conflict = false
			} else if inst.Mtime.Equal(winner.Mtime) {
				if inst.Hash != winner.Hash {
					conflict = true
				}
			}
		}
	}

	if conflict {
		return nil, true, nil
	}

	// Return a copy to avoid pointer issues with loop variables
	finalWinner := *winner
	return &finalWinner, false, nil
}
