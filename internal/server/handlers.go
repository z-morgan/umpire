package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/zmorgan/umpire/internal/feedback"
	"github.com/zmorgan/umpire/internal/git"
	"github.com/zmorgan/umpire/internal/review"
)

// ReviewContext holds the resolved git state for the review session.
type ReviewContext struct {
	Repo          *git.Repo
	BaseRef       string
	HeadRef       string
	BaseSHA       string
	HeadSHA       string
	MergeBase     string
	Store         *review.Store
	FeedbackStore *feedback.Store
	ShutdownFn    func() // called by /api/shutdown to trigger server shutdown
}

// RegisterAPI registers the API routes on the server's mux.
func RegisterAPI(mux *http.ServeMux, rc *ReviewContext) {
	mux.HandleFunc("GET /api/commits", rc.handleCommits)
	mux.HandleFunc("GET /api/diff", rc.handleDiff)
	mux.HandleFunc("GET /api/files", rc.handleFiles)
	mux.HandleFunc("GET /api/info", rc.handleInfo)
	mux.HandleFunc("POST /api/review", rc.handleReview)
	mux.HandleFunc("GET /api/file-lines", rc.handleFileLines)
	mux.HandleFunc("POST /api/record-feedback", rc.handleRecordFeedback)
	mux.HandleFunc("GET /api/feedback-prompt", rc.handleFeedbackPrompt)
	mux.HandleFunc("POST /api/shutdown", rc.handleShutdown)
}

func (rc *ReviewContext) handleInfo(w http.ResponseWriter, r *http.Request) {
	info := map[string]string{
		"base_ref":   rc.BaseRef,
		"head_ref":   rc.HeadRef,
		"base_sha":   rc.BaseSHA,
		"head_sha":   rc.HeadSHA,
		"merge_base": rc.MergeBase,
	}
	writeJSON(w, info)
}

func (rc *ReviewContext) handleCommits(w http.ResponseWriter, r *http.Request) {
	commits, err := rc.Repo.CommitsBetween(rc.MergeBase, rc.HeadSHA)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, commits)
}

func (rc *ReviewContext) handleDiff(w http.ResponseWriter, r *http.Request) {
	commitSHA := r.URL.Query().Get("commit")

	var diff string
	var err error
	if commitSHA != "" {
		diff, err = rc.Repo.DiffForCommit(commitSHA)
	} else {
		diff, err = rc.Repo.DiffBetween(rc.MergeBase, rc.HeadSHA)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(diff))
}

func (rc *ReviewContext) handleFiles(w http.ResponseWriter, r *http.Request) {
	files, err := rc.Repo.ChangedFiles(rc.MergeBase, rc.HeadSHA)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, files)
}

func (rc *ReviewContext) handleReview(w http.ResponseWriter, r *http.Request) {
	var req review.SubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	rev := &review.Review{
		BaseRef:  rc.BaseRef,
		HeadRef:  rc.HeadRef,
		BaseSHA:  rc.BaseSHA,
		HeadSHA:  rc.HeadSHA,
		Summary:  req.Summary,
		Comments: req.Comments,
	}

	path, err := rc.Store.Save(rev)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{"path": path})
}

func (rc *ReviewContext) handleFileLines(w http.ResponseWriter, r *http.Request) {
	ref := r.URL.Query().Get("ref")
	path := r.URL.Query().Get("path")
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")

	if ref == "" || path == "" || startStr == "" || endStr == "" {
		http.Error(w, "ref, path, start, and end are required", http.StatusBadRequest)
		return
	}

	start, err := strconv.Atoi(startStr)
	if err != nil || start < 1 {
		http.Error(w, "start must be a positive integer", http.StatusBadRequest)
		return
	}

	end, err := strconv.Atoi(endStr)
	if err != nil || end < start {
		http.Error(w, "end must be an integer >= start", http.StatusBadRequest)
		return
	}

	content, err := rc.Repo.ShowFile(ref, path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	lines := strings.Split(content, "\n")
	if start > len(lines) {
		start = len(lines) + 1
	}
	if end > len(lines) {
		end = len(lines)
	}

	sliced := lines[start-1 : end]

	writeJSON(w, map[string]any{
		"lines": sliced,
		"start": start,
		"end":   end,
	})
}

func (rc *ReviewContext) handleRecordFeedback(w http.ResponseWriter, r *http.Request) {
	var req feedback.SubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	snap := &feedback.Snapshot{
		RepoPath: rc.Repo.Dir,
		BaseRef:  rc.BaseRef,
		HeadRef:  rc.HeadRef,
		BaseSHA:  rc.BaseSHA,
		HeadSHA:  rc.HeadSHA,
		Diff:     req.Diff,
		Review:   req.Review,
	}

	if _, err := rc.FeedbackStore.Save(snap); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	count, err := rc.FeedbackStore.Count()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]any{
		"count":             count,
		"threshold_reached": count >= feedback.Threshold,
	})
}

func (rc *ReviewContext) handleFeedbackPrompt(w http.ResponseWriter, r *http.Request) {
	count, err := rc.FeedbackStore.Count()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{
		"prompt": feedback.GeneratePrompt(count),
	})
}

func (rc *ReviewContext) handleShutdown(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	if rc.ShutdownFn != nil {
		go rc.ShutdownFn()
	}
}

func writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
