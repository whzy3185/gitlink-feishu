"use strict";

const assert = require("assert");
const fs = require("fs");
const path = require("path");

const skillsDir = path.resolve(__dirname, "..", "..", "skills");

const requiredSkillFiles = {
  "gitlink-maintainer-copilot": [
    "SKILL.md",
    "references/evidence-pack.md",
    "references/playbooks.md",
    "references/governance-issue-template.md",
    "examples/maintainer-copilot-workflow.md",
    "examples/sample-dashboard-report.md",
  ],
};

function readText(file) {
  return fs.readFileSync(file, "utf8");
}

function parseFrontmatter(markdown, file) {
  const match = markdown.match(/^---\n([\s\S]*?)\n---\n/);
  assert.ok(match, `${file} must start with YAML frontmatter`);

  const fields = {};
  for (const line of match[1].split("\n")) {
    const fieldMatch = line.match(/^([A-Za-z0-9_-]+):\s*(.*)$/);
    if (fieldMatch) {
      fields[fieldMatch[1]] = fieldMatch[2].replace(/^["']|["']$/g, "");
    }
  }
  return fields;
}

function markdownFiles(root) {
  const files = [];
  for (const entry of fs.readdirSync(root, { withFileTypes: true })) {
    const fullPath = path.join(root, entry.name);
    if (entry.isDirectory()) {
      files.push(...markdownFiles(fullPath));
    } else if (entry.isFile() && entry.name.endsWith(".md")) {
      files.push(fullPath);
    }
  }
  return files;
}

function localMarkdownLinks(markdown) {
  const links = [];
  const linkPattern = /\[[^\]]+\]\(([^)]+)\)/g;
  let match;
  while ((match = linkPattern.exec(markdown)) !== null) {
    const target = match[1].trim();
    if (
      target.startsWith("http://") ||
      target.startsWith("https://") ||
      target.startsWith("#") ||
      target.startsWith("mailto:")
    ) {
      continue;
    }
    links.push(target.split("#")[0]);
  }
  return links.filter(Boolean);
}

for (const [skillName, files] of Object.entries(requiredSkillFiles)) {
  for (const file of files) {
    assert.ok(
      fs.existsSync(path.join(skillsDir, skillName, file)),
      `${skillName} must include ${file}`
    );
  }
}

const skillDirs = fs
  .readdirSync(skillsDir, { withFileTypes: true })
  .filter((entry) => entry.isDirectory() && entry.name.startsWith("gitlink-"))
  .map((entry) => entry.name)
  .sort();

const names = new Set();

for (const dirName of skillDirs) {
  const skillFile = path.join(skillsDir, dirName, "SKILL.md");
  assert.ok(fs.existsSync(skillFile), `${dirName} must include SKILL.md`);

  const frontmatter = parseFrontmatter(readText(skillFile), skillFile);
  assert.equal(frontmatter.name, dirName, `${dirName} frontmatter name must match directory`);
  assert.ok(frontmatter.description, `${dirName} must include description`);
  assert.ok(!names.has(frontmatter.name), `${frontmatter.name} must be unique`);
  names.add(frontmatter.name);
}

for (const file of markdownFiles(skillsDir)) {
  const markdown = readText(file);
  for (const link of localMarkdownLinks(markdown)) {
    const targetPath = path.resolve(path.dirname(file), link);
    assert.ok(fs.existsSync(targetPath), `${file} links to missing file: ${link}`);
  }
}

console.log("skills structure tests passed");
