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
