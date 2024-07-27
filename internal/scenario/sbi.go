package scenario

import (
	"log/slog"
	"os"
)

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
		common: ScenarioCommon{ws: os.Getenv("wsAddr"), outputDir: "/data"},
		user:   user,
		pass:   pass,
	}, nil
}
