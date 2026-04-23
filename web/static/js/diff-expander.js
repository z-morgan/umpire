'use strict';

const DiffExpander = {
  BATCH_SIZE: 20,

  attach() {
    document.querySelectorAll('.d2h-diff-tbody').forEach(tbody => {
      tbody.querySelectorAll('tr').forEach(row => {
        const infoCell = row.querySelector('.d2h-info');
        if (!infoCell) return;

        row.classList.add('d2h-expandable');
        row.addEventListener('click', () => this.handleExpand(row));
      });
    });
  },

  async handleExpand(infoRow) {
    const fileWrapper = infoRow.closest('.d2h-file-wrapper');
    if (!fileWrapper) return;

    const nameEl = fileWrapper.querySelector('.d2h-file-name');
    if (!nameEl) return;
    const filePath = nameEl.textContent.trim();

    const gap = this.computeGap(infoRow);
    if (!gap || gap.start > gap.end) {
      infoRow.remove();
      return;
    }

    const ref = Sidebar.activeCommitSHA || App.info.head_sha;
    const fetchStart = Math.max(gap.start, gap.end - this.BATCH_SIZE + 1);
    const fetchEnd = gap.end;

    const data = await API.getFileLines(ref, filePath, fetchStart, fetchEnd);

    const tbody = infoRow.closest('tbody');
    for (let i = 0; i < data.lines.length; i++) {
      const lineNum = data.start + i;
      const contextRow = this.buildContextRow(lineNum, data.lines[i]);
      tbody.insertBefore(contextRow, infoRow);
    }

    // Remove the info row if the gap is fully consumed
    const remainingGap = { start: gap.start, end: fetchStart - 1 };
    if (remainingGap.start > remainingGap.end) {
      infoRow.remove();
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

  buildContextRow(lineNum, content) {
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
    contentSpan.textContent = ' ' + content;

    codeDiv.appendChild(contentSpan);
    codeTd.appendChild(codeDiv);

    tr.appendChild(lineNumTd);
    tr.appendChild(codeTd);

    return tr;
  },
};
