'use strict';

const DiffView = {
  init() {
    this.attachLineClickHandlers();
  },

  attachLineClickHandlers() {
    document.querySelectorAll('.d2h-code-linenumber').forEach(lineNumEl => {
      lineNumEl.style.cursor = 'pointer';
      lineNumEl.addEventListener('click', (e) => {
        e.preventDefault();
        this.handleLineClick(lineNumEl);
      });
    });
  },

  handleLineClick(lineNumEl) {
    const row = lineNumEl.closest('tr') || lineNumEl.closest('.d2h-code-line');
    if (!row) return;

    const fileInfo = this.getFileForElement(row);
    if (!fileInfo) return;

    const lineNumber = this.getLineNumber(lineNumEl);
    if (!lineNumber) return;

    const side = this.getSide(lineNumEl);
    const existingForm = row.parentNode.querySelector(`.comment-form[data-line="${lineNumber}"][data-file="${fileInfo}"]`);
    if (existingForm) {
      existingForm.querySelector('textarea').focus();
      return;
    }

    this.showCommentForm(row, fileInfo, lineNumber, side);
  },

  getFileForElement(el) {
    const fileWrapper = el.closest('.d2h-file-wrapper');
    if (!fileWrapper) return null;
    const nameEl = fileWrapper.querySelector('.d2h-file-name');
    return nameEl ? nameEl.textContent.trim() : null;
  },

  getLineNumber(lineNumEl) {
    // diff2html line-by-line: line numbers are in the element text
    const text = lineNumEl.textContent.trim();
    // The right-side line number is what we want for new lines
    const nums = text.split(/\s+/).filter(n => n && !isNaN(n));
    if (nums.length === 0) return null;
    // Use the last number (right side) for additions, first for deletions
    return parseInt(nums[nums.length - 1], 10);
  },

  getSide(lineNumEl) {
    const row = lineNumEl.closest('tr');
    if (!row) return 'right';
    if (row.querySelector('.d2h-del')) return 'left';
    return 'right';
  },

  getDiffHunk(row) {
    // Collect a few lines of context around the comment
    const lines = [];
    let current = row;
    for (let i = 0; i < 3 && current && current.previousElementSibling; i++) {
      current = current.previousElementSibling;
    }
    for (let i = 0; i < 7 && current; i++) {
      const codeEl = current.querySelector('.d2h-code-line-ctn');
      if (codeEl) {
        lines.push(codeEl.textContent);
      }
      current = current.nextElementSibling;
    }
    return lines.join('\n');
  },

  showCommentForm(row, file, lineNumber, side) {
    const formRow = document.createElement('tr');
    formRow.className = 'comment-form-row';

    const formCell = document.createElement('td');
    formCell.colSpan = 3;

    const form = document.createElement('div');
    form.className = 'comment-form';
    form.dataset.line = lineNumber;
    form.dataset.file = file;

    form.innerHTML = `
      <textarea placeholder="Leave a comment..." rows="3"></textarea>
      <div class="comment-form-actions">
        <button class="btn btn-save">Comment</button>
        <button class="btn btn-cancel">Cancel</button>
      </div>
    `;

    const textarea = form.querySelector('textarea');
    const saveBtn = form.querySelector('.btn-save');
    const cancelBtn = form.querySelector('.btn-cancel');

    saveBtn.addEventListener('click', () => {
      const body = textarea.value.trim();
      if (!body) return;
      const diffHunk = this.getDiffHunk(row);
      const comment = ReviewState.addComment(file, lineNumber, side, body, diffHunk);
      formRow.remove();
      this.renderComment(row, comment);
    });

    cancelBtn.addEventListener('click', () => formRow.remove());

    textarea.addEventListener('keydown', (e) => {
      if (e.key === 'Escape') {
        formRow.remove();
      } else if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
        saveBtn.click();
      }
    });

    formCell.appendChild(form);
    formRow.appendChild(formCell);

    // Insert after the clicked row
    if (row.nextSibling) {
      row.parentNode.insertBefore(formRow, row.nextSibling);
    } else {
      row.parentNode.appendChild(formRow);
    }

    textarea.focus();
  },

  renderComment(row, comment) {
    const commentRow = document.createElement('tr');
    commentRow.className = 'comment-row';
    commentRow.dataset.commentId = comment.id;

    const cell = document.createElement('td');
    cell.colSpan = 3;

    cell.innerHTML = `
      <div class="inline-comment">
        <div class="comment-body">${this.escapeHTML(comment.body)}</div>
        <div class="comment-actions">
          <button class="btn-link btn-edit">Edit</button>
          <button class="btn-link btn-delete">Delete</button>
        </div>
      </div>
    `;

    cell.querySelector('.btn-edit').addEventListener('click', () => {
      this.editComment(commentRow, row, comment);
    });

    cell.querySelector('.btn-delete').addEventListener('click', () => {
      ReviewState.deleteComment(comment.id);
      commentRow.remove();
    });

    commentRow.appendChild(cell);

    if (row.nextSibling) {
      row.parentNode.insertBefore(commentRow, row.nextSibling);
    } else {
      row.parentNode.appendChild(commentRow);
    }
  },

  editComment(commentRow, originalRow, comment) {
    const cell = commentRow.querySelector('td');
    cell.innerHTML = `
      <div class="comment-form">
        <textarea rows="3">${this.escapeHTML(comment.body)}</textarea>
        <div class="comment-form-actions">
          <button class="btn btn-save">Save</button>
          <button class="btn btn-cancel">Cancel</button>
        </div>
      </div>
    `;

    const textarea = cell.querySelector('textarea');
    const saveBtn = cell.querySelector('.btn-save');
    const cancelBtn = cell.querySelector('.btn-cancel');

    saveBtn.addEventListener('click', () => {
      const body = textarea.value.trim();
      if (!body) return;
      ReviewState.updateComment(comment.id, body);
      commentRow.remove();
      this.renderComment(originalRow, { ...comment, body });
    });

    cancelBtn.addEventListener('click', () => {
      commentRow.remove();
      this.renderComment(originalRow, comment);
    });

    textarea.addEventListener('keydown', (e) => {
      if (e.key === 'Escape') cancelBtn.click();
      if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) saveBtn.click();
    });

    textarea.focus();
  },

  reattachComments() {
    // After re-rendering the diff, re-attach saved comments to matching lines
    const comments = ReviewState.getAllComments();
    for (const comment of comments) {
      const row = this.findRowForComment(comment);
      if (row) {
        this.renderComment(row, comment);
      }
    }
    this.attachLineClickHandlers();
  },

  findRowForComment(comment) {
    const fileWrappers = document.querySelectorAll('.d2h-file-wrapper');
    for (const wrapper of fileWrappers) {
      const nameEl = wrapper.querySelector('.d2h-file-name');
      if (!nameEl || nameEl.textContent.trim() !== comment.file) continue;

      const lineNumEls = wrapper.querySelectorAll('.d2h-code-linenumber');
      for (const lineNumEl of lineNumEls) {
        const lineNum = this.getLineNumber(lineNumEl);
        if (lineNum === comment.line_start) {
          const row = lineNumEl.closest('tr');
          if (row) return row;
        }
      }
    }
    return null;
  },

  escapeHTML(str) {
    const div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
  },
};
