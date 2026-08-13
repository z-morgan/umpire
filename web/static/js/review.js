'use strict';

const ReviewState = {
  comments: [],
  nextId: 1,

  addComment(file, lineNumber, side, body, diffHunk) {
    const comment = {
      id: `c${this.nextId++}`,
      file,
      // Empty when authoring against the full base..head diff, where
      // line_start is head-relative. Set to the active commit's SHA when
      // authoring in a per-commit view, where line_start is relative to
      // that commit and diff_hunk is the authoritative locator.
      commit_sha: Sidebar.activeCommitSHA || '',
      line_start: lineNumber,
      line_end: lineNumber,
      side,
      body,
      diff_hunk: diffHunk,
    };
    this.comments.push(comment);
    return comment;
  },

  updateComment(id, body) {
    const comment = this.comments.find(c => c.id === id);
    if (comment) {
      comment.body = body;
    }
    return comment;
  },

  deleteComment(id) {
    this.comments = this.comments.filter(c => c.id !== id);
  },

  getCommentsForLine(file, lineNumber, side) {
    return this.comments.filter(
      c => c.file === file && c.line_start === lineNumber && c.side === side
    );
  },

  getCommentCount() {
    return this.comments.length;
  },

  getAllComments() {
    return [...this.comments];
  },
};
