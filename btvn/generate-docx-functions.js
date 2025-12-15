const { Document, Packer, Paragraph, TextRun, HeadingLevel, AlignmentType, Table, TableRow, TableCell, WidthType, BorderStyle, ShadingType } = require('docx');

// Remove all emojis and special icons from text
function removeEmojis(text) {
  return text
    .replace(/[⚠️✅❌⭐🎯📊📝📌🔄💡🚀🎓📂📄🔍💾🖥️⚡🔧📈📉🌐💻🔒🔓⬆️⬇️➡️⬅️↔️↕️🔴🟢🟡🔵⚪⚫🟤🟠🟣✓✗★☆●○◆◇▶◀▲▼△▽□■◻◼☑☐🔹🔸▪▫]/g, '')
    .replace(/[\uFE0F\uFE0E]/g, '')
    .replace(/[\u{1F300}-\u{1F9FF}]/gu, '')
    .replace(/[\u{2600}-\u{26FF}]/gu, '')
    .replace(/[\u{2700}-\u{27BF}]/gu, '')
    .replace(/[\u{1F600}-\u{1F64F}]/gu, '')
    .replace(/[\u{1F680}-\u{1F6FF}]/gu, '')
    .replace(/[\u{1F1E0}-\u{1F1FF}]/gu, '')
    .replace(/\s+/g, ' ')
    .trim();
}

// Remove emojis but keep diagram/box-drawing characters and arrows for code blocks
function removeEmojisKeepDiagram(text) {
  return text
    .replace(/[⚠️✅❌⭐🎯📊📝📌🔄💡🚀🎓📂📄🔍💾🖥️⚡🔧📈📉🌐💻🔒🔓🔴🟢🟡🔵⚪⚫🟤🟠🟣✓✗★☆●○◆◇▶◀▲▼△▽□■◻◼☑☐🔹🔸▪▫]/g, '')
    .replace(/[\uFE0F\uFE0E]/g, '')
    .replace(/[\u{1F300}-\u{1F9FF}]/gu, '')
    .replace(/[\u{1F600}-\u{1F64F}]/gu, '')
    .replace(/[\u{1F680}-\u{1F6FF}]/gu, '')
    .replace(/[\u{1F1E0}-\u{1F1FF}]/gu, '')
    .replace(/[⬆️⬇️➡️⬅️↔️↕️]/g, function(match) {
      if (match.includes('⬆') || match.includes('↑')) return '↑';
      if (match.includes('⬇') || match.includes('↓')) return '↓';
      if (match.includes('➡') || match.includes('→')) return '→';
      if (match.includes('⬅') || match.includes('←')) return '←';
      return '';
    });
}

// Create a Word table from markdown table rows
function createTable(tableRows) {
  const rows = tableRows.map((row, rowIndex) => {
    const cells = row.split('|').filter(c => c.trim() !== '').map(c => c.trim());
    return new TableRow({
      children: cells.map(cellText => {
        const isHeader = rowIndex === 0;
        const cleanText = removeEmojis(cellText.replace(/\*\*/g, '').replace(/`/g, ''));
        return new TableCell({
          children: [new Paragraph({
            children: [new TextRun({ text: cleanText, bold: isHeader, size: 20 })],
            alignment: AlignmentType.CENTER
          })],
          shading: isHeader ? { fill: 'e0e0e0' } : undefined,
          margins: { top: 50, bottom: 50, left: 75, right: 75 }
        });
      })
    });
  });

  return new Table({
    rows: rows,
    width: { size: 100, type: WidthType.PERCENTAGE }
  });
}

// Create a code/diagram box with proper formatting
function createDiagramBox(codeLines, language = '') {
  const borderConfig = {
    top: { style: BorderStyle.SINGLE, size: 8, color: 'cccccc' },
    bottom: { style: BorderStyle.SINGLE, size: 8, color: 'cccccc' },
    left: { style: BorderStyle.SINGLE, size: 24, color: '4a90d9' },
    right: { style: BorderStyle.SINGLE, size: 8, color: 'cccccc' }
  };

  const cleanedLines = codeLines.map(line => removeEmojisKeepDiagram(line));

  const codeContent = cleanedLines.map(line => {
    return new Paragraph({
      children: [new TextRun({
        text: line || ' ',
        font: 'Consolas',
        size: 20,
        color: '2d2d2d'
      })],
      spacing: { before: 20, after: 20, line: 276 }
    });
  });

  return new Table({
    rows: [
      new TableRow({
        children: [
          new TableCell({
            children: codeContent,
            shading: { fill: 'f6f8fa', type: ShadingType.CLEAR },
            margins: { top: 150, bottom: 150, left: 250, right: 250 },
            borders: borderConfig
          })
        ]
      })
    ],
    width: { size: 100, type: WidthType.PERCENTAGE }
  });
}

// Parse inline formatting and return TextRun array
function parseInlineFormatting(text) {
  const runs = [];
  const cleanText = removeEmojis(text);
  const pattern = /(\*\*[^*]+\*\*|`[^`]+`)/g;
  let lastIndex = 0;
  let match;

  while ((match = pattern.exec(cleanText)) !== null) {
    if (match.index > lastIndex) {
      runs.push(new TextRun({ text: cleanText.slice(lastIndex, match.index) }));
    }
    const matched = match[0];
    if (matched.startsWith('**')) {
      runs.push(new TextRun({ text: matched.slice(2, -2), bold: true }));
    } else if (matched.startsWith('`')) {
      runs.push(new TextRun({ text: matched.slice(1, -1), font: 'Courier New', size: 20 }));
    }
    lastIndex = match.index + matched.length;
  }

  if (lastIndex < cleanText.length) {
    runs.push(new TextRun({ text: cleanText.slice(lastIndex) }));
  }
  if (runs.length === 0) {
    runs.push(new TextRun({ text: cleanText }));
  }
  return runs;
}

function parseMarkdown(content) {
  const elements = [];
  const lines = content.split('\n');
  let inCodeBlock = false;
  let codeContent = [];
  let tableRows = [];
  let inTable = false;

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const isTableRow = line.trim().startsWith('|') && line.trim().endsWith('|');
    const isSeparatorRow = isTableRow && line.match(/^\|[\s\-:|]+\|$/);

    if (isTableRow && !inCodeBlock) {
      if (!isSeparatorRow) {
        tableRows.push(line);
      }
      inTable = true;
      continue;
    } else if (inTable && !isTableRow) {
      if (tableRows.length > 0) {
        elements.push(createTable(tableRows));
        elements.push(new Paragraph({ text: '', spacing: { after: 100 } }));
        tableRows = [];
      }
      inTable = false;
    }

    if (line.startsWith('```')) {
      if (inCodeBlock) {
        elements.push(createDiagramBox(codeContent));
        elements.push(new Paragraph({ text: '', spacing: { after: 100 } }));
        codeContent = [];
      }
      inCodeBlock = !inCodeBlock;
      continue;
    }

    if (inCodeBlock) {
      codeContent.push(line);
      continue;
    }

    if (line.startsWith('#### ')) {
      elements.push(new Paragraph({
        children: [new TextRun({ text: removeEmojis(line.replace('#### ', '')), bold: true, size: 22 })],
        spacing: { before: 150, after: 75 }
      }));
    } else if (line.startsWith('### ')) {
      elements.push(new Paragraph({
        text: removeEmojis(line.replace('### ', '')),
        heading: HeadingLevel.HEADING_3,
        spacing: { before: 200, after: 100 }
      }));
    } else if (line.startsWith('## ')) {
      elements.push(new Paragraph({
        text: removeEmojis(line.replace('## ', '')),
        heading: HeadingLevel.HEADING_2,
        spacing: { before: 300, after: 150 }
      }));
    } else if (line.startsWith('# ')) {
      elements.push(new Paragraph({
        text: removeEmojis(line.replace('# ', '')),
        heading: HeadingLevel.HEADING_1,
        spacing: { before: 400, after: 200 }
      }));
    } else if (line.startsWith('**') && line.endsWith('**')) {
      elements.push(new Paragraph({
        children: [new TextRun({ text: removeEmojis(line.replace(/\*\*/g, '')), bold: true })],
        spacing: { before: 100, after: 50 }
      }));
    } else if (line.startsWith('- ') || line.startsWith('• ') || line.startsWith('* ')) {
      const text = line.replace(/^[-•*]\s+/, '');
      const inlineRuns = parseInlineFormatting(text);
      elements.push(new Paragraph({
        children: [new TextRun({ text: '• ' }), ...inlineRuns],
        indent: { left: 360 },
        spacing: { before: 50, after: 50 }
      }));
    } else if (line.trim() === '---') {
      elements.push(new Paragraph({ text: '', spacing: { before: 200, after: 200 } }));
    } else if (line.trim()) {
      elements.push(new Paragraph({
        children: parseInlineFormatting(line),
        spacing: { before: 50, after: 50 }
      }));
    }
  }

  if (tableRows.length > 0) {
    elements.push(createTable(tableRows));
  }

  return elements;
}

module.exports = { removeEmojis, removeEmojisKeepDiagram, createTable, createDiagramBox, parseInlineFormatting, parseMarkdown, Paragraph, TextRun, HeadingLevel, AlignmentType };

