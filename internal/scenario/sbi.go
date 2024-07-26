package scenario

import (
	"context"
	"log/slog"
)

type ScenarioCommon struct {
	outputDir string
}

type ScenarioSBI struct {
	common ScenarioCommon
	user   string
	pass   string
}

func NewScenarioSBI(outputDir, user, pass string) (*ScenarioSBI, error) {
	if outputDir == "" {
		slog.Error("outputDir required")
		return nil, ErrorInvalidOption
	}
	if user == "" {
		slog.Error("user required")
		return nil, ErrorInvalidOption
	}
	if pass == "" {
		slog.Error("pass required")
		return nil, ErrorInvalidOption
	}

	return &ScenarioSBI{
		common: ScenarioCommon{
			outputDir: outputDir,
		},
		user: user,
		pass: pass,
	}, nil
}

func (s *ScenarioSBI) Start(ctx context.Context) error {
	return nil
}
