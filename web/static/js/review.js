'use strict';

const ReviewState = {
  comments: [],
  nextId: 1,

  addComment(file, lineNumber, side, body, diffHunk) {
    const comment = {
      id: `c${this.nextId++}`,
      file,
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
