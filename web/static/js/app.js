'use strict';

const App = {
  diffContainer: null,

  async init() {
    this.diffContainer = document.getElementById('diff-container');

    const info = await API.getInfo();
    this.info = info;
    document.getElementById('branch-info').textContent =
      `${info.base_ref}..${info.head_ref}`;

    this.initSubmitBar();
    this.initKeyboardShortcuts();
    Resize.init();

    await Promise.all([
      this.loadFullDiff(),
      Sidebar.init(),
    ]);
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

    const message = document.createElement('div');
    message.className = 'commit-message';

    const shortSHA = commit.sha.substring(0, 7);
    const subject = document.createElement('h2');
    subject.className = 'commit-message-subject';
    subject.textContent = commit.subject;

    const meta = document.createElement('div');
    meta.className = 'commit-message-meta';
    meta.innerHTML = `<span class="commit-sha">${shortSHA}</span> ${Sidebar.escapeHTML(commit.author)} &middot; ${commit.date}`;

    message.append(subject, meta);

    if (commit.body) {
      const body = document.createElement('pre');
      body.className = 'commit-message-body';
      body.textContent = commit.body;
      message.append(body);
    }

    header.append(nav, message);
    this.diffContainer.parentNode.insertBefore(header, this.diffContainer);
  },

  removeCommitHeader() {
    const existing = document.getElementById('commit-header');
    if (existing) existing.remove();
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

    if (!summary && comments.length === 0) {
      alert('Add a summary or comments before submitting.');
      return;
    }

    const submitBtn = document.getElementById('submit-review');
    submitBtn.disabled = true;
    submitBtn.textContent = 'Submitting...';

    this.lastReview = { summary, comments };
    const result = await API.submitReview({ summary, comments });
    this.savedReviewPath = result.path;

    const submitBar = document.getElementById('submit-bar');
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

    document.getElementById('feedback-yes').addEventListener('click', () => this.recordFeedback(submitBar));
    document.getElementById('feedback-no').addEventListener('click', () => this.shutdownAndShow(submitBar));
  },

  renderSavedPath() {
    return `<p class="review-saved-path">Review saved to <code>${this.savedReviewPath}</code></p>`;
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
      API.shutdown();
      return;
    }

    submitBar.innerHTML = `
      <div class="submit-success">
        ${this.renderSavedPath()}
        <p>Feedback recorded &mdash; ${result.count} snapshots available.</p>
        <p class="feedback-message">Generate a prompt to analyze your feedback and propose Claude config updates?</p>
        <div class="feedback-actions">
          <button class="btn btn-save" id="feedback-copy">Copy Prompt to Clipboard</button>
          <button class="btn btn-cancel" id="feedback-skip">Not now</button>
        </div>
      </div>
    `;

    document.getElementById('feedback-copy').addEventListener('click', () => this.copyPromptAndShutdown(submitBar));
    document.getElementById('feedback-skip').addEventListener('click', () => this.shutdownAndShow(submitBar));
  },

  async copyPromptAndShutdown(submitBar) {
    const result = await API.getFeedbackPrompt();
    await navigator.clipboard.writeText(result.prompt);

    submitBar.innerHTML = `
      <div class="submit-success">
        ${this.renderSavedPath()}
        <p>Prompt copied to clipboard.</p>
        <p class="feedback-message">Server shutting down...</p>
      </div>
    `;
    API.shutdown();
  },

  shutdownAndShow(submitBar) {
    submitBar.innerHTML = `
      <div class="submit-success">
        ${this.renderSavedPath()}
        <p>Server shutting down...</p>
      </div>
    `;
    API.shutdown();
  },
};

document.addEventListener('DOMContentLoaded', () => App.init());
