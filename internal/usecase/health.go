package usecase

import "sync-audio-platforms-go/internal/domain"

type HealthUseCase struct {
	version string
}

func NewHealthUseCase(version string) *HealthUseCase {
	return &HealthUseCase{
		version: version,
	}
}

func (h *HealthUseCase) Check() domain.HealthStatus {
	return domain.HealthStatus{
		Status:  "OK",
		Version: h.version,
	}
}
