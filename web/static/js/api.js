'use strict';

const API = {
  async getInfo() {
    const resp = await fetch('/api/info');
    return resp.json();
  },

  async getCommits() {
    const resp = await fetch('/api/commits');
    return resp.json();
  },

  async getDiff(commitSHA) {
    const url = commitSHA ? `/api/diff?commit=${commitSHA}` : '/api/diff';
    const resp = await fetch(url);
    return resp.text();
  },

  async getFiles() {
    const resp = await fetch('/api/files');
    return resp.json();
  },

  async submitReview(review) {
    const resp = await fetch('/api/review', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(review),
    });
    return resp.json();
  },
};
