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
        case 'n': // Next commit
          this.navigateCommit(1);
          break;
        case 'p': // Previous commit
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
  },

  updateCommentCount() {
    const count = ReviewState.getCommentCount();
    const label = count === 1 ? '1 comment' : `${count} comments`;
    document.getElementById('comment-count').textContent = label;
  },

  async loadFullDiff() {
    const diff = await API.getDiff();
    this.renderDiff(diff);
  },

  async loadCommitDiff(sha) {
    const diff = await API.getDiff(sha);
    this.renderDiff(diff);
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
