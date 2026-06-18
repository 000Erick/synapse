package usecase

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite" // registers the "sqlite" driver; pure Go, no CGO
)

// HealthResult is the output of HealthUsecase.Run.
type HealthResult struct {
	Status           string
	EngramReachable  bool
	SynapseReachable bool
	EngramPath       string
	SynapsePath      string
	Version          string
}

// HealthUsecase pings both databases and reports liveness.
type HealthUsecase struct {
	engramPath  string
	synapsePath string
}

// NewHealthUsecase creates a new HealthUsecase.
func NewHealthUsecase(engramPath, synapsePath string) *HealthUsecase {
	return &HealthUsecase{engramPath: engramPath, synapsePath: synapsePath}
}

// Run pings both databases and returns a HealthResult.
func (h *HealthUsecase) Run(ctx context.Context) (*HealthResult, error) {
	result := &HealthResult{
		EngramPath:  h.engramPath,
		SynapsePath: h.synapsePath,
		Version:     "0.1.0",
	}

	result.EngramReachable = pingDB(ctx, fmt.Sprintf("file:%s?mode=ro", h.engramPath))
	result.SynapseReachable = pingDB(ctx, h.synapsePath)

	if result.EngramReachable && result.SynapseReachable {
		result.Status = "ok"
	} else {
		result.Status = "degraded"
	}
	return result, nil
}

func pingDB(ctx context.Context, dsn string) bool {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return false
	}
	defer db.Close()
	return db.PingContext(ctx) == nil
}
