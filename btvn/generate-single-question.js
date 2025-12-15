const fs = require('fs');
const path = require('path');
const { Document, Packer } = require('docx');

// Get question number from command line argument
const questionNum = process.argv[2] || '3';
const inputFile = `question-${questionNum}.md`;
const outputFile = `question-${questionNum}.docx`;

// Import functions from shared module
const { parseMarkdown, Paragraph, HeadingLevel, AlignmentType } = require('./generate-docx-functions.js');

async function generateSingleQuestion() {
  const filePath = path.join(__dirname, inputFile);

  if (!fs.existsSync(filePath)) {
    console.error(`File not found: ${inputFile}`);
    process.exit(1);
  }

  const content = fs.readFileSync(filePath, 'utf-8');
  const elements = parseMarkdown(content);

  const doc = new Document({
    sections: [{
      properties: {},
      children: elements
    }]
  });

  const buffer = await Packer.toBuffer(doc);
  fs.writeFileSync(path.join(__dirname, outputFile), buffer);
  console.log(`Created: ${outputFile}`);
}

generateSingleQuestion().catch(console.error);

