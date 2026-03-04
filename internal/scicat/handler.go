package scicat

import "github.com/paulscherrerinstitute/scicat-s3-broker/internal/config"

type Handler struct {
	DatasetsHandler
	PublisheddataHandler
}

func NewHandler(cfg *config.Config) *Handler {
	datasetsService := &DatasetsServiceImpl{config: cfg}
	publisheddataService := &PublisheddataServiceImpl{config: cfg, datasetsService: datasetsService}
	datasetsHandler := DatasetsHandler{datasetsService}
	publisheddataHandler := PublisheddataHandler{publisheddataService}
	return &Handler{
		datasetsHandler,
		publisheddataHandler,
	}
}
