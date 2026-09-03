'use strict';

const App = {
  diffContainer: null,
  commitMessageEdits: {},

  // A commit body is hard-wrapped at 72 columns, but real bodies rarely fill
  // the full width, so a pane sized to an exact 72-column line reads wider than
  // it needs to. sizeCommitHeader measures the 72-column reference, then pulls
  // the pane in to this fraction of it; the occasional long line wraps.
  COMMIT_BODY_COLUMNS: 72,
  COMMIT_HEADER_WIDTH_SCALE: 0.89,

  async init() {
    this.diffContainer = document.getElementById('diff-container');

    const info = await API.getInfo();
    this.info = info;
    document.getElementById('branch-info').textContent =
      `${info.base_ref}..${info.head_ref}`;

    this.initSubmitBar();
    this.initKeyboardShortcuts();
    Resize.init();

    await Sidebar.init();
    const firstCommit = Sidebar.commits[0];
    if (firstCommit) {
      Sidebar.activeCommitSHA = firstCommit.sha;
      Sidebar.render();
      const [fullDiff] = await Promise.all([
        API.getDiff(),
        this.loadCommitDiff(firstCommit.sha),
      ]);
      this.fullDiff = fullDiff;
    } else {
      await this.loadFullDiff();
    }
  },

  initSubmitBar() {
    document.getElementById('submit-review').addEventListener('click', () => {
      this.submitReview();
    });
  },

  initKeyboardShortcuts() {
    document.addEventListener('keydown', (e) => {
      // Don't intercept when typing in inputs/textareas
      if (e.target.tagName === 'TEXTAREA' || e.target.tagName === 'INPUT') return;

      const fileWrappers = document.querySelectorAll('.d2h-file-wrapper');
      const fileArray = Array.from(fileWrappers);

      switch (e.key) {
        case 'j': // Next file
          this.navigateFile(fileArray, 1);
          break;
        case 'k': // Previous file
          this.navigateFile(fileArray, -1);
          break;
        case 'ArrowRight': // Next commit
          this.navigateCommit(1);
          break;
        case 'ArrowLeft': // Previous commit
          this.navigateCommit(-1);
          break;
      }
    });
  },

  navigateFile(fileWrappers, direction) {
    if (fileWrappers.length === 0) return;

    const scrollY = window.scrollY + 60;
    let currentIndex = -1;

    for (let i = 0; i < fileWrappers.length; i++) {
      if (fileWrappers[i].offsetTop <= scrollY) {
        currentIndex = i;
      }
    }

    const nextIndex = Math.max(0, Math.min(fileWrappers.length - 1, currentIndex + direction));
    fileWrappers[nextIndex].scrollIntoView({ behavior: 'smooth', block: 'start' });
  },

  navigateCommit(direction) {
    if (Sidebar.commits.length === 0) return;

    const commits = Sidebar.commits;
    let currentIndex = commits.findIndex(c => c.sha === Sidebar.activeCommitSHA);

    // -1 means "All changes" (before first commit)
    const nextIndex = currentIndex + direction;

    if (nextIndex < -1 || nextIndex >= commits.length) return;

    this.flushPendingEdits();

    if (nextIndex === -1) {
      Sidebar.activeCommitSHA = null;
      Sidebar.switchTab('commits');
      this.loadFullDiff();
    } else {
      Sidebar.activeCommitSHA = commits[nextIndex].sha;
      Sidebar.switchTab('commits');
      this.loadCommitDiff(commits[nextIndex].sha);
    }

    window.scrollTo({ top: 0 });
  },

  // Persist any in-progress edits before navigating away, as if each editor's
  // Save button were clicked. Only an explicit Cancel discards a change.
  flushPendingEdits() {
    this.flushCommitMessageEditor();
    this.flushOpenCommentForms();
  },

  flushCommitMessageEditor() {
    const header = document.getElementById('commit-header');
    if (!header) return;
    const editor = header.querySelector('.commit-message-editing');
    if (!editor) return;

    const commit = Sidebar.commits.find(c => c.sha === Sidebar.activeCommitSHA);
    if (!commit) return;

    const subject = editor.querySelector('.commit-message-subject-input').value;
    const body = editor.querySelector('.commit-message-body-input').value;
    this.persistCommitMessageEdit(commit, subject, body);
  },

  flushOpenCommentForms() {
    document.querySelectorAll('.comment-form .btn-save').forEach(btn => btn.click());
  },

  updateCommentCount() {
    const count = ReviewState.getCommentCount();
    const label = count === 1 ? '1 comment' : `${count} comments`;
    document.getElementById('comment-count').textContent = label;
  },

  async loadFullDiff() {
    this.removeCommitHeader();
    const diff = await API.getDiff();
    this.fullDiff = diff;
    this.renderDiff(diff);
  },

  async loadCommitDiff(sha) {
    const commit = Sidebar.commits.find(c => c.sha === sha);
    this.renderCommitHeader(commit);
    const diff = await API.getDiff(sha);
    this.renderDiff(diff);
  },

  renderCommitHeader(commit) {
    this.removeCommitHeader();
    if (!commit) return;

    const commits = Sidebar.commits;
    const currentIndex = commits.findIndex(c => c.sha === commit.sha);
    const hasPrev = true; // Can always go back to "All changes"
    const hasNext = currentIndex < commits.length - 1;

    const header = document.createElement('div');
    header.id = 'commit-header';

    const nav = document.createElement('div');
    nav.className = 'commit-nav';

    const prevBtn = document.createElement('button');
    prevBtn.className = 'btn commit-nav-btn';
    prevBtn.innerHTML = '<kbd>&larr;</kbd>';
    prevBtn.disabled = !hasPrev;
    prevBtn.addEventListener('click', () => this.navigateCommit(-1));

    const position = document.createElement('span');
    position.className = 'commit-nav-position';
    position.textContent = `Commit ${currentIndex + 1} of ${commits.length}`;

    const nextBtn = document.createElement('button');
    nextBtn.className = 'btn commit-nav-btn';
    nextBtn.innerHTML = '<kbd>&rarr;</kbd>';
    nextBtn.disabled = !hasNext;
    nextBtn.addEventListener('click', () => this.navigateCommit(1));

    nav.append(prevBtn, position, nextBtn);

    const message = this.buildCommitMessageView(commit);

    header.append(nav, message);
    this.diffContainer.parentNode.insertBefore(header, this.diffContainer);
    this.sizeCommitHeader(header);

    const handle = document.createElement('div');
    handle.id = 'commit-resize-handle';
    handle.className = 'resize-handle';
    this.diffContainer.parentNode.insertBefore(handle, this.diffContainer);
    Resize.attachCommitHeaderDrag(handle, header);
  },

  // Size the pane from a 72-column reference line rather than the actual text,
  // so every commit's pane lands at the same width, then scale it in (see
  // COMMIT_HEADER_WIDTH_SCALE) so it hugs typical bodies. The body uses a
  // proportional font, so the reference is measured in that font. It keeps
  // white-space: pre-wrap, so a too-long line -- or dragging narrower -- wraps.
  sizeCommitHeader(header) {
    const message = header.querySelector('.commit-message');
    if (!message) return;

    const referenceWidth = this.measureBodyReference(message);
    const chrome = this.horizontalChrome(header) + this.horizontalChrome(message);
    const scrollbarAllowance = 12;
    const fit = referenceWidth + chrome + scrollbarAllowance;
    const desired = fit * this.COMMIT_HEADER_WIDTH_SCALE;

    const capped = Math.min(desired, window.innerWidth * 0.5, Resize.MAX_COMMIT_HEADER_WIDTH);
    header.style.width = Math.max(Resize.MIN_COMMIT_HEADER_WIDTH, capped) + 'px';
  },

  // Rendered width of a 72-character line in the body's own font. A hidden probe
  // nested where the real body sits inherits the exact font, so this holds even
  // for a commit that has no body of its own to measure.
  measureBodyReference(message) {
    const probe = document.createElement('pre');
    probe.className = 'commit-message-body';
    probe.style.position = 'absolute';
    probe.style.visibility = 'hidden';
    message.append(probe);

    const style = getComputedStyle(probe);
    const font = `${style.fontWeight} ${style.fontSize} ${style.fontFamily}`;
    const reference = 'n'.repeat(this.COMMIT_BODY_COLUMNS);
    const width = this.measureTextWidth(reference, font);

    message.removeChild(probe);
    return width;
  },

  measureTextWidth(text, font) {
    const canvas = document.createElement('canvas');
    const context = canvas.getContext('2d');
    context.font = font;
    return context.measureText(text).width;
  },

  horizontalChrome(element) {
    const style = getComputedStyle(element);
    return parseFloat(style.paddingLeft) + parseFloat(style.paddingRight)
      + parseFloat(style.borderLeftWidth) + parseFloat(style.borderRightWidth);
  },

  buildCommitMessageView(commit) {
    const edit = this.commitMessageEdits[commit.sha];
    const displaySubject = edit ? edit.subject : commit.subject;
    const displayBody = edit ? edit.body : (commit.body || '');

    const message = document.createElement('div');
    message.className = 'commit-message';

    const messageHeader = document.createElement('div');
    messageHeader.className = 'commit-message-header';

    const subject = document.createElement('h2');
    subject.className = 'commit-message-subject';
    subject.textContent = displaySubject;

    const editBtn = document.createElement('button');
    editBtn.className = 'btn commit-message-edit-btn';
    editBtn.textContent = 'Edit';
    editBtn.addEventListener('click', () => this.startEditingCommitMessage(commit));

    messageHeader.append(subject, editBtn);

    const shortSHA = commit.sha.substring(0, 7);
    const meta = document.createElement('div');
    meta.className = 'commit-message-meta';
    meta.innerHTML = `<span class="commit-sha">${shortSHA}</span> ${Sidebar.escapeHTML(commit.author)} &middot; ${commit.date}`;

    if (edit) {
      const editedBadge = document.createElement('span');
      editedBadge.className = 'commit-message-edited';
      editedBadge.textContent = 'edited';
      meta.append(' ', editedBadge);
    }

    message.append(messageHeader, meta);

    if (displayBody) {
      const body = document.createElement('pre');
      body.className = 'commit-message-body';
      body.textContent = displayBody;
      message.append(body);
    }

    return message;
  },

  buildCommitMessageEditor(commit) {
    const edit = this.commitMessageEdits[commit.sha];
    const currentSubject = edit ? edit.subject : commit.subject;
    const currentBody = edit ? edit.body : (commit.body || '');

    const message = document.createElement('div');
    message.className = 'commit-message commit-message-editing';

    const subjectInput = document.createElement('input');
    subjectInput.type = 'text';
    subjectInput.className = 'commit-message-subject-input';
    subjectInput.value = currentSubject;

    const bodyTextarea = document.createElement('textarea');
    bodyTextarea.className = 'commit-message-body-input';
    bodyTextarea.value = currentBody;
    bodyTextarea.rows = 8;
    bodyTextarea.placeholder = 'Body (optional)';

    const actions = document.createElement('div');
    actions.className = 'commit-message-edit-actions';

    const saveBtn = document.createElement('button');
    saveBtn.className = 'btn btn-save';
    saveBtn.textContent = 'Save';
    saveBtn.addEventListener('click', () => {
      this.saveCommitMessageEdit(commit, subjectInput.value, bodyTextarea.value);
    });

    const cancelBtn = document.createElement('button');
    cancelBtn.className = 'btn btn-cancel';
    cancelBtn.textContent = 'Cancel';
    cancelBtn.addEventListener('click', () => this.exitCommitMessageEditor(commit));

    actions.append(saveBtn, cancelBtn);

    if (edit) {
      const resetBtn = document.createElement('button');
      resetBtn.className = 'btn btn-cancel';
      resetBtn.textContent = 'Reset to original';
      resetBtn.addEventListener('click', () => {
        delete this.commitMessageEdits[commit.sha];
        this.exitCommitMessageEditor(commit);
      });
      actions.append(resetBtn);
    }

    message.append(subjectInput, bodyTextarea, actions);
    return message;
  },

  startEditingCommitMessage(commit) {
    const header = document.getElementById('commit-header');
    if (!header) return;
    const existingMessage = header.querySelector('.commit-message');
    if (!existingMessage) return;

    const editor = this.buildCommitMessageEditor(commit);
    existingMessage.replaceWith(editor);
    editor.querySelector('.commit-message-subject-input').focus();
  },

  exitCommitMessageEditor(commit) {
    const header = document.getElementById('commit-header');
    if (!header) return;
    const editor = header.querySelector('.commit-message-editing');
    if (!editor) return;

    editor.replaceWith(this.buildCommitMessageView(commit));
  },

  saveCommitMessageEdit(commit, subject, body) {
    if (!subject.trim()) {
      alert('Subject cannot be empty.');
      return;
    }
    this.persistCommitMessageEdit(commit, subject, body);
    this.exitCommitMessageEditor(commit);
  },

  // Records an edit, or clears it when the text matches the original commit.
  // Returns false without touching state for an empty subject, since that
  // isn't a savable commit message.
  persistCommitMessageEdit(commit, subject, body) {
    const trimmedSubject = subject.trim();
    const trimmedBody = body.trim();

    if (!trimmedSubject) return false;

    if (trimmedSubject === commit.subject && trimmedBody === (commit.body || '')) {
      delete this.commitMessageEdits[commit.sha];
    } else {
      this.commitMessageEdits[commit.sha] = {
        subject: trimmedSubject,
        body: trimmedBody,
      };
    }
    return true;
  },

  collectCommitMessageEdits() {
    const edits = [];
    for (const [sha, edit] of Object.entries(this.commitMessageEdits)) {
      const original = Sidebar.commits.find(c => c.sha === sha);
      edits.push({
        sha,
        original_subject: original ? original.subject : '',
        original_body: original ? (original.body || '') : '',
        edited_subject: edit.subject,
        edited_body: edit.body,
      });
    }
    return edits;
  },

  removeCommitHeader() {
    const existing = document.getElementById('commit-header');
    if (existing) existing.remove();
    const handle = document.getElementById('commit-resize-handle');
    if (handle) handle.remove();
  },

  renderDiff(diffString) {
    if (!diffString.trim()) {
      this.diffContainer.innerHTML = '<p class="empty-diff">No changes found.</p>';
      return;
    }

    this.diffContainer.innerHTML = '';
    const targetElement = document.createElement('div');
    this.diffContainer.appendChild(targetElement);

    const diff2htmlUi = new Diff2HtmlUI(targetElement, diffString, {
      drawFileList: false,
      matching: 'lines',
      outputFormat: 'line-by-line',
      highlight: true,
      fileListToggle: false,
      colorScheme: 'dark',
    });
    diff2htmlUi.draw();
    diff2htmlUi.highlightCode();

    DiffView.reattachComments();
    DiffExpander.attach();
  },

  async submitReview() {
    const summary = document.getElementById('review-summary').value.trim();
    const comments = ReviewState.getAllComments();
    const commitMessageEdits = this.collectCommitMessageEdits();

    if (!summary && comments.length === 0 && commitMessageEdits.length === 0) {
      alert('Add a summary, comments, or a commit message edit before submitting.');
      return;
    }

    const submitBtn = document.getElementById('submit-review');
    submitBtn.disabled = true;
    submitBtn.textContent = 'Submitting...';

    this.lastReview = { summary, comments };
    const result = await API.submitReview({
      summary,
      comments,
      commit_message_edits: commitMessageEdits,
    });
    this.savedReviewPath = result.path;

    const submitBar = document.getElementById('submit-bar');
    document.documentElement.style.removeProperty('--submit-bar-height');
    submitBar.innerHTML = `
      <div class="submit-success">
        ${this.renderSavedPath()}
        <p class="feedback-message">Record this feedback to improve future Claude sessions?</p>
        <div class="feedback-actions">
          <button class="btn btn-save" id="feedback-yes">Yes</button>
          <button class="btn btn-cancel" id="feedback-no">No thanks</button>
        </div>
      </div>
    `;

    this.attachSavedPathCopy();
    document.getElementById('feedback-yes').addEventListener('click', () => this.recordFeedback(submitBar));
    document.getElementById('feedback-no').addEventListener('click', () => this.shutdownAndShow(submitBar));
  },

  renderSavedPath() {
    return `<p class="review-saved-path">Review saved to <code>${this.savedReviewPath}</code><button class="btn-link review-path-copy" id="copy-review-path">Copy</button></p>`;
  },

  attachSavedPathCopy() {
    const btn = document.getElementById('copy-review-path');
    if (btn) btn.addEventListener('click', () => this.copyReviewPath());
  },

  async copyReviewPath() {
    await navigator.clipboard.writeText(this.savedReviewPath);
    const btn = document.getElementById('copy-review-path');
    btn.textContent = 'Copied';
    btn.disabled = true;
    setTimeout(() => {
      btn.textContent = 'Copy';
      btn.disabled = false;
    }, 1500);
  },

  async recordFeedback(submitBar) {
    const review = this.lastReview;
    const result = await API.recordFeedback({
      diff: this.fullDiff || '',
      review: { summary: review.summary, comments: review.comments },
    });

    if (!result.threshold_reached) {
      const remaining = 5 - result.count;
      const noun = remaining === 1 ? 'review' : 'reviews';
      submitBar.innerHTML = `
        <div class="submit-success">
          ${this.renderSavedPath()}
          <p>Feedback recorded (${remaining} more ${noun} until analysis is available).</p>
          <p class="feedback-message">Server shutting down...</p>
        </div>
      `;
      this.attachSavedPathCopy();
      API.shutdown();
      return;
    }

    const promptResult = await API.getFeedbackPrompt();
    this.feedbackPrompt = promptResult.prompt;

    submitBar.innerHTML = `
      <div class="submit-success">
        ${this.renderSavedPath()}
        <p>Feedback recorded &mdash; ${result.count} snapshots available.</p>
        <p class="feedback-message">Paste this prompt into Claude to analyze your feedback and propose Claude config updates:</p>
        <div class="feedback-prompt-wrapper">
          <pre class="feedback-prompt" id="feedback-prompt-text"></pre>
          <button class="btn btn-save feedback-prompt-copy" id="feedback-copy">Copy</button>
        </div>
        <div class="feedback-actions">
          <button class="btn btn-cancel" id="feedback-done">Done</button>
        </div>
      </div>
    `;

    this.attachSavedPathCopy();
    document.getElementById('feedback-prompt-text').textContent = this.feedbackPrompt;
    document.getElementById('feedback-copy').addEventListener('click', () => this.copyFeedbackPrompt());
    document.getElementById('feedback-done').addEventListener('click', () => this.shutdownAndShow(submitBar));
  },

  async copyFeedbackPrompt() {
    await navigator.clipboard.writeText(this.feedbackPrompt);
    const btn = document.getElementById('feedback-copy');
    btn.textContent = 'Copied';
    btn.disabled = true;
    setTimeout(() => {
      btn.textContent = 'Copy';
      btn.disabled = false;
    }, 1500);
  },

  shutdownAndShow(submitBar) {
    submitBar.innerHTML = `
      <div class="submit-success">
        ${this.renderSavedPath()}
        <p>Server shutting down...</p>
      </div>
    `;
    this.attachSavedPathCopy();
    API.shutdown();
  },
};

document.addEventListener('DOMContentLoaded', () => App.init());
