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

    const result = await API.submitReview({ summary, comments });

    const submitBar = document.getElementById('submit-bar');
    submitBar.innerHTML = `
      <div class="submit-success">
        Review saved to <code>${result.path}</code>
        <p>Server shutting down...</p>
      </div>
    `;
  },
};

document.addEventListener('DOMContentLoaded', () => App.init());
