'use strict';

const DiffExpander = {
  BATCH_SIZE: 20,

  // A bar eats its gap from one end, a batch at a time. Both directions insert
  // the fetched lines before the bar, so they differ only in which slice of the
  // gap they take and which end of the gap is left over.
  DIRECTIONS: {
    up: {
      fetchWindow(gapStart, gapEnd, batchSize) {
        return { start: Math.max(gapStart, gapEnd - batchSize + 1), end: gapEnd };
      },
      remainingGap(gapStart, gapEnd, fetched) {
        return { start: gapStart, end: fetched.start - 1 };
      },
    },
    down: {
      fetchWindow(gapStart, gapEnd, batchSize) {
        return { start: gapStart, end: Math.min(gapEnd, gapStart + batchSize - 1) };
      },
      remainingGap(gapStart, gapEnd, fetched) {
        return { start: fetched.end + 1, end: gapEnd };
      },
    },
  },

  attach() {
    document.querySelectorAll('.d2h-diff-tbody').forEach(tbody => {
      tbody.querySelectorAll('tr').forEach(row => {
        const infoCell = row.querySelector('.d2h-info');
        if (!infoCell) return;

        const gap = this.computeGap(row);
        if (!gap || gap.start > gap.end) return;

        row.dataset.gapStart = gap.start;
        row.dataset.gapEnd = gap.end;
        row.classList.add('d2h-expandable');
        row.addEventListener('click', () => this.handleExpand(row, this.DIRECTIONS.up));
      });
    });
  },

  async handleExpand(infoRow, direction) {
    const fileWrapper = infoRow.closest('.d2h-file-wrapper');
    if (!fileWrapper) return;

    const nameEl = fileWrapper.querySelector('.d2h-file-name');
    if (!nameEl) return;
    const filePath = nameEl.textContent.trim();

    const gapStart = parseInt(infoRow.dataset.gapStart, 10);
    const gapEnd = parseInt(infoRow.dataset.gapEnd, 10);
    if (isNaN(gapStart) || isNaN(gapEnd) || gapStart > gapEnd) {
      infoRow.remove();
      return;
    }

    const ref = Sidebar.activeCommitSHA || App.info.head_sha;
    const fetched = direction.fetchWindow(gapStart, gapEnd, this.BATCH_SIZE);

    const data = await API.getFileLines(ref, filePath, fetched.start, fetched.end);

    const language = this.resolveLanguage(fileWrapper);
    const tbody = infoRow.closest('tbody');
    for (let i = 0; i < data.lines.length; i++) {
      const lineNum = data.start + i;
      const contextRow = this.buildContextRow(lineNum, data.lines[i], language);
      tbody.insertBefore(contextRow, infoRow);
    }

    const remaining = direction.remainingGap(gapStart, gapEnd, fetched);
    if (remaining.start > remaining.end) {
      infoRow.remove();
    } else {
      infoRow.dataset.gapStart = remaining.start;
      infoRow.dataset.gapEnd = remaining.end;
    }
  },

  computeGap(infoRow) {
    const codeCell = infoRow.querySelector('.d2h-code-line');
    if (!codeCell) return null;

    const hunkText = codeCell.textContent;
    const match = hunkText.match(/@@ .+?\+(\d+)/);
    if (!match) return null;

    const hunkNewStart = parseInt(match[1], 10);
    const prevEnd = this.findPrevLineNumber(infoRow);

    return {
      start: prevEnd + 1,
      end: hunkNewStart - 1,
    };
  },

  findPrevLineNumber(infoRow) {
    let row = infoRow.previousElementSibling;
    while (row) {
      // Skip comment rows and form rows
      if (row.classList.contains('comment-row') || row.classList.contains('comment-form-row')) {
        row = row.previousElementSibling;
        continue;
      }

      const lineNumEl = row.querySelector('.d2h-code-linenumber');
      if (lineNumEl) {
        const text = lineNumEl.textContent.trim();
        const nums = text.split(/\s+/).filter(n => n && !isNaN(n));
        if (nums.length > 0) {
          return parseInt(nums[nums.length - 1], 10);
        }
      }
      row = row.previousElementSibling;
    }
    // First hunk in file — gap starts at line 1
    return 0;
  },

  buildContextRow(lineNum, content, language) {
    const tr = document.createElement('tr');

    const lineNumTd = document.createElement('td');
    lineNumTd.className = 'd2h-code-linenumber d2h-cntx';

    const num1 = document.createElement('div');
    num1.className = 'line-num1';
    num1.textContent = lineNum;

    const num2 = document.createElement('div');
    num2.className = 'line-num2';
    num2.textContent = lineNum;

    lineNumTd.appendChild(num1);
    lineNumTd.appendChild(num2);

    const codeTd = document.createElement('td');
    codeTd.className = 'd2h-cntx';

    const codeDiv = document.createElement('div');
    codeDiv.className = 'd2h-code-line d2h-cntx';

    const contentSpan = document.createElement('span');
    contentSpan.className = 'd2h-code-line-ctn';
    this.applyHighlight(contentSpan, ' ' + content, language);

    codeDiv.appendChild(contentSpan);
    codeTd.appendChild(codeDiv);

    tr.appendChild(lineNumTd);
    tr.appendChild(codeTd);

    return tr;
  },

  resolveLanguage(fileWrapper) {
    if (typeof hljs === 'undefined') return null;
    const lang = fileWrapper.getAttribute('data-lang');
    if (lang && hljs.getLanguage(lang)) return lang;
    return null;
  },

  applyHighlight(span, text, language) {
    if (!language) {
      span.textContent = text;
      return;
    }
    const result = hljs.highlight(text, { language, ignoreIllegals: true });
    span.innerHTML = result.value;
    span.classList.add('hljs');
    if (result.language) {
      span.classList.add(result.language);
    }
  },
};
