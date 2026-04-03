package server

import (
	"encoding/json"
	"net/http"

	"github.com/zmorgan/umpire/internal/git"
	"github.com/zmorgan/umpire/internal/review"
)

// ReviewContext holds the resolved git state for the review session.
type ReviewContext struct {
	Repo       *git.Repo
	BaseRef    string
	HeadRef    string
	BaseSHA    string
	HeadSHA    string
	MergeBase  string
	Store      *review.Store
	OnSubmit   func(path string) // called after review is saved
}

// RegisterAPI registers the API routes on the server's mux.
func RegisterAPI(mux *http.ServeMux, rc *ReviewContext) {
	mux.HandleFunc("GET /api/commits", rc.handleCommits)
	mux.HandleFunc("GET /api/diff", rc.handleDiff)
	mux.HandleFunc("GET /api/files", rc.handleFiles)
	mux.HandleFunc("GET /api/info", rc.handleInfo)
	mux.HandleFunc("POST /api/review", rc.handleReview)
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

	resp := map[string]string{"path": path}
	writeJSON(w, resp)

	if rc.OnSubmit != nil {
		go rc.OnSubmit(path)
	}
}

func writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
