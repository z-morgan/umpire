package server

import (
	"encoding/json"
	"net/http"

	"github.com/zmorgan/umpire/internal/git"
)

// ReviewContext holds the resolved git state for the review session.
type ReviewContext struct {
	Repo     *git.Repo
	BaseRef  string
	HeadRef  string
	BaseSHA  string
	HeadSHA  string
	MergeBase string
}

// RegisterAPI registers the API routes on the server's mux.
func RegisterAPI(mux *http.ServeMux, rc *ReviewContext) {
	mux.HandleFunc("GET /api/commits", rc.handleCommits)
	mux.HandleFunc("GET /api/diff", rc.handleDiff)
	mux.HandleFunc("GET /api/files", rc.handleFiles)
	mux.HandleFunc("GET /api/info", rc.handleInfo)
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

func writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
