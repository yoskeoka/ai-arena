"use strict";

const fs = require("node:fs");
const path = require("node:path");

const TERMS_PATH = path.join(process.cwd(), "config/textlint/terms.jsonl");

function loadTerms() {
  const content = fs.readFileSync(TERMS_PATH, "utf8");
  const lines = content.split(/\r?\n/);
  const terms = [];

  for (const [index, line] of lines.entries()) {
    const trimmed = line.trim();
    if (!trimmed) {
      continue;
    }

    let record;
    try {
      record = JSON.parse(trimmed);
    } catch (error) {
      throw new Error(
        `Failed to parse ${TERMS_PATH} line ${index + 1}: ${error.message}`
      );
    }

    if (!record.pattern || !record.replacement) {
      throw new Error(
        `Expected pattern and replacement in ${TERMS_PATH} line ${index + 1}`
      );
    }

    let matcher;
    try {
      matcher = new RegExp(record.pattern, "g");
    } catch (error) {
      throw new Error(
        `Invalid pattern in ${TERMS_PATH} line ${index + 1}: ${error.message}`
      );
    }

    terms.push({
      matcher,
      replacement: record.replacement,
    });
  }

  return terms;
}

module.exports = function preferAiArenaTerms(context) {
  const { RuleError, Syntax, getSource, report } = context;
  const terms = loadTerms();

  return {
    [Syntax.Str](node) {
      const text = getSource(node);

      for (const term of terms) {
        term.matcher.lastIndex = 0;
        let match;
        while ((match = term.matcher.exec(text)) !== null) {
          const matchedText = match[0];
          report(
            node,
            new RuleError(
              `Prefer "${term.replacement}" instead of "${matchedText}".`,
              { index: match.index }
            )
          );

          if (match[0].length === 0) {
            term.matcher.lastIndex += 1;
          }
        }
      }
    },
  };
};
