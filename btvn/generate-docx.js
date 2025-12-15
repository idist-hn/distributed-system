const fs = require('fs');
const path = require('path');
const { Document, Packer, Paragraph, TextRun, HeadingLevel, AlignmentType, Table, TableRow, TableCell, WidthType, BorderStyle, ShadingType } = require('docx');

// Read all markdown files
const questionFiles = [
  'question-1.md', 'question-2.md', 'question-3.md', 'question-4.md', 'question-5.md',
  'question-6.md', 'question-7.md', 'question-8.md', 'question-9.md', 'question-10.md'
];

// Remove all emojis and special icons from text
function removeEmojis(text) {
  return text
    // Remove common emojis used in the documents
    .replace(/[⚠️✅❌⭐🎯📊📝📌🔄💡🚀🎓📂📄🔍💾🖥️⚡🔧📈📉🌐💻🔒🔓⬆️⬇️➡️⬅️↔️↕️🔴🟢🟡🔵⚪⚫🟤🟠🟣✓✗★☆●○◆◇▶◀▲▼△▽□■◻◼☑☐🔹🔸▪▫]/g, '')
    // Remove emoji variation selectors
    .replace(/[\uFE0F\uFE0E]/g, '')
    // Remove other common emoji ranges
    .replace(/[\u{1F300}-\u{1F9FF}]/gu, '')
    .replace(/[\u{2600}-\u{26FF}]/gu, '')
    .replace(/[\u{2700}-\u{27BF}]/gu, '')
    .replace(/[\u{1F600}-\u{1F64F}]/gu, '')
    .replace(/[\u{1F680}-\u{1F6FF}]/gu, '')
    .replace(/[\u{1F1E0}-\u{1F1FF}]/gu, '')
    // Clean up extra spaces
    .replace(/\s+/g, ' ')
    .trim();
}

// Remove emojis but keep diagram/box-drawing characters and arrows for code blocks
function removeEmojisKeepDiagram(text) {
  return text
    // Remove common emojis used in the documents (keep → ↓ ↑ ← for diagrams/flow)
    .replace(/[⚠️✅❌⭐🎯📊📝📌🔄💡🚀🎓📂📄🔍💾🖥️⚡🔧📈📉🌐💻🔒🔓🔴🟢🟡🔵⚪⚫🟤🟠🟣✓✗★☆●○◆◇▶◀▲▼△▽□■◻◼☑☐🔹🔸▪▫]/g, '')
    // Remove emoji variation selectors
    .replace(/[\uFE0F\uFE0E]/g, '')
    // Remove other common emoji ranges (but preserve arrows in 2190-21FF range)
    .replace(/[\u{1F300}-\u{1F9FF}]/gu, '')
    .replace(/[\u{1F600}-\u{1F64F}]/gu, '')
    .replace(/[\u{1F680}-\u{1F6FF}]/gu, '')
    .replace(/[\u{1F1E0}-\u{1F1FF}]/gu, '')
    // Remove directional emoji arrows but keep simple arrows (→ ← ↑ ↓)
    .replace(/[⬆️⬇️➡️⬅️↔️↕️]/g, function(match) {
      // Convert emoji arrows to simple arrows
      if (match.includes('⬆') || match.includes('↑')) return '↑';
      if (match.includes('⬇') || match.includes('↓')) return '↓';
      if (match.includes('➡') || match.includes('→')) return '→';
      if (match.includes('⬅') || match.includes('←')) return '←';
      return '';
    });
    // Don't trim or collapse spaces - preserve diagram formatting
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
  // Strong visible border for code blocks
  const borderConfig = {
    top: { style: BorderStyle.SINGLE, size: 8, color: 'cccccc' },
    bottom: { style: BorderStyle.SINGLE, size: 8, color: 'cccccc' },
    left: { style: BorderStyle.SINGLE, size: 24, color: '4a90d9' },  // Blue left border like code editors
    right: { style: BorderStyle.SINGLE, size: 8, color: 'cccccc' }
  };

  // Clean lines - remove emojis but keep diagram characters
  const cleanedLines = codeLines.map(line => removeEmojisKeepDiagram(line));

  // Determine if this is a diagram (ASCII art) or code
  const isDiagram = cleanedLines.some(line =>
    line.includes('┌') || line.includes('└') || line.includes('│') ||
    line.includes('─') || line.includes('├') || line.includes('┤') ||
    line.includes('┬') || line.includes('╱') || line.includes('╲')
  );

  const isJson = language === 'json' || cleanedLines.some(line => line.trim().startsWith('{') || line.trim().startsWith('['));

  // Create paragraphs for each line with proper code styling
  const codeContent = cleanedLines.map(line => {
    let textRuns = [];

    if (isJson) {
      // JSON formatting - highlight keys and values
      const parts = line.split(/(".*?")/g);
      parts.forEach((part) => {
        if (part.match(/^".*"$/)) {
          textRuns.push(new TextRun({
            text: part,
            font: 'Consolas',
            size: 20,
            color: part.includes(':') ? '0066cc' : '008800'
          }));
        } else {
          textRuns.push(new TextRun({
            text: part,
            font: 'Consolas',
            size: 20,
            color: '333333'
          }));
        }
      });
    } else {
      // Regular code or diagram - use Consolas font
      textRuns.push(new TextRun({
        text: line || ' ',
        font: 'Consolas',
        size: 20,
        color: '2d2d2d'
      }));
    }

    return new Paragraph({
      children: textRuns,
      spacing: { before: 20, after: 20, line: 276 }  // 1.15 line spacing
    });
  });

  // Light gray background for code
  const bgColor = isDiagram ? 'f5f7f9' : 'f6f8fa';

  return new Table({
    rows: [
      new TableRow({
        children: [
          new TableCell({
            children: codeContent,
            shading: { fill: bgColor, type: ShadingType.CLEAR },
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

  // Pattern for **bold**, `code`, and regular text
  const pattern = /(\*\*[^*]+\*\*|`[^`]+`)/g;
  let lastIndex = 0;
  let match;

  while ((match = pattern.exec(cleanText)) !== null) {
    // Add text before the match
    if (match.index > lastIndex) {
      runs.push(new TextRun({ text: cleanText.slice(lastIndex, match.index) }));
    }

    const matched = match[0];
    if (matched.startsWith('**')) {
      // Bold text
      runs.push(new TextRun({
        text: matched.slice(2, -2),
        bold: true
      }));
    } else if (matched.startsWith('`')) {
      // Inline code
      runs.push(new TextRun({
        text: matched.slice(1, -1),
        font: 'Courier New',
        size: 20,
        shading: { fill: 'e8e8e8' }
      }));
    }

    lastIndex = match.index + matched.length;
  }

  // Add remaining text
  if (lastIndex < cleanText.length) {
    runs.push(new TextRun({ text: cleanText.slice(lastIndex) }));
  }

  // If no formatting found, return simple text
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

    // Check if line is a table row
    const isTableRow = line.trim().startsWith('|') && line.trim().endsWith('|');
    const isSeparatorRow = isTableRow && line.match(/^\|[\s\-:|]+\|$/);

    // Handle table accumulation
    if (isTableRow && !inCodeBlock) {
      if (!isSeparatorRow) {
        tableRows.push(line);
      }
      inTable = true;
      continue;
    } else if (inTable && !isTableRow) {
      // End of table - create table element
      if (tableRows.length > 0) {
        elements.push(createTable(tableRows));
        elements.push(new Paragraph({ text: '', spacing: { after: 100 } }));
        tableRows = [];
      }
      inTable = false;
    }

    // Handle code blocks
    if (line.startsWith('```')) {
      if (inCodeBlock) {
        // End code block - create diagram box
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

    // Handle headings
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
      // Regular paragraph - handle inline formatting
      elements.push(new Paragraph({
        children: parseInlineFormatting(line),
        spacing: { before: 50, after: 50 }
      }));
    }
  }

  // Handle any remaining table
  if (tableRows.length > 0) {
    elements.push(createTable(tableRows));
  }

  return elements;
}

async function generateDocx() {
  const allElements = [
    new Paragraph({
      text: 'BÀI TẬP CÁC HỆ THỐNG PHÂN TÁN',
      heading: HeadingLevel.TITLE,
      alignment: AlignmentType.CENTER,
      spacing: { after: 400 }
    }),
    new Paragraph({ text: '', spacing: { after: 200 } })
  ];

  for (const file of questionFiles) {
    const filePath = path.join(__dirname, file);
    if (fs.existsSync(filePath)) {
      const content = fs.readFileSync(filePath, 'utf-8');
      const elements = parseMarkdown(content);
      allElements.push(...elements);
      // Add page break between questions
      allElements.push(new Paragraph({ text: '', pageBreakBefore: true }));
    }
  }

  const doc = new Document({
    sections: [{ properties: {}, children: allElements }]
  });

  const buffer = await Packer.toBuffer(doc);
  fs.writeFileSync(path.join(__dirname, 'bai-tap-he-thong-phan-tan.docx'), buffer);
  console.log('Created: bai-tap-he-thong-phan-tan.docx');
}

generateDocx().catch(console.error);

