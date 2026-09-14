package api

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/reportgen"
)

// reportDir — тека для PDF-файлів звітів; встановлюється з main через SetReportDir.
var reportDir = filepath.Join("data", "reports")

func (s *Server) SetReportDir(dir string) {
	reportDir = dir
}

func (s *Server) HandleRunReportCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		writeErr(w, http.StatusMethodNotAllowed, "метод не підтримується")
		return
	}
	runID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || runID <= 0 {
		writeErr(w, http.StatusBadRequest, "неправильний id прогона")
		return
	}
	report, err := s.store.GetPDFRunReport(r.Context(), runID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeErr(w, http.StatusNotFound, "прогін не знайдено")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	pdfBytes, err := reportgen.BuildRunReport(report)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	fileName := fmt.Sprintf("run-%d-%s.pdf", runID, time.Now().Format("20060102-150405"))
	storageKey := filepath.Join(reportDir, fileName)
	if err := os.WriteFile(storageKey, pdfBytes, 0o644); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	reportID, err := s.store.InsertReport(r.Context(), runID, fileName, storageKey, len(report.Jobs), "{}")
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"id":           reportID,
		"file_name":    fileName,
		"orders_total": len(report.Jobs),
		"download":     fmt.Sprintf("/api/reports/%d/download", reportID),
	})
}

func (s *Server) HandleReportDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		writeErr(w, http.StatusMethodNotAllowed, "метод не підтримується")
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeErr(w, http.StatusBadRequest, "неправильний id звіту")
		return
	}
	storageKey, fileName, err := s.store.GetReport(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusNotFound, "звіт не знайдено")
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fileName))
	http.ServeFile(w, r, storageKey)
}
