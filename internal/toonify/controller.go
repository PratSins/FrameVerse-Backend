package toonify

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Controller struct {
	service *Service
}

func NewController(service *Service) *Controller {
	return &Controller{
		service: service,
	}
}

func (c *Controller) MountRoutes(r chi.Router) {
	r.Route("/api/v1/toonify", func(r chi.Router) {
		r.Post("/upload-url", c.CreateUploadURL)
		r.Post("/{jobID}/process", c.Process)
		r.Get("/{jobID}", c.GetJob)
	})
}

func (c *Controller) CreateUploadURL(w http.ResponseWriter, r *http.Request) {
	var req CreateUploadRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.ContentType == "" {
		req.ContentType = "video/mp4"
	}

	if req.Style == "" {
		req.Style = "anime"
	}

	result, err := c.service.CreateUpload(r.Context(), req.ContentType, req.Style)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (c *Controller) Process(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobID")

	if jobID == "" {
		http.Error(w, "missing job id", http.StatusBadRequest)
		return
	}

	// MVP:
	// run processing in the background.
	go func() {
		if err := c.service.Process(context.Background(), jobID); err != nil {
			log.Printf("toonify job %s failed: %v", jobID, err)
		}
	}()

	writeJSON(
		w,
		http.StatusAccepted,
		map[string]string{
			"job_id": jobID,
			"status": "processing",
		},
	)
}

func (c *Controller) GetJob(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobID")

	job, err := c.service.dao.Get(r.Context(), jobID)

	if err != nil {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}

	response := JobResponse{
		JobID:  job.ID,
		Status: job.Status,
		Style:  job.Style,
		Error:  job.Error,
	}

	if job.Status == StatusCompleted {
		url, err := c.service.gcs.GenerateDownloadURL(job.OutputObject)

		if err == nil {
			response.DownloadURL = url
		}
	}

	writeJSON(w, http.StatusOK, response)
}

func writeJSON(w http.ResponseWriter, status int, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(value)
}
