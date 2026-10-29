#!/usr/bin/env node
/**
 * Design-token contrast gate.
 *
 * Reads the HSL design tokens from src/index.css and validates WCAG 2.1 AA
 * contrast (>= 4.5:1 for body-size text) for every text/surface pair in both
 * themes. Exits 1 on the first failing report.
 */
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import path from "node:path";

const cssPath = path.join(
  path.dirname(fileURLToPath(import.meta.url)),
  "..",
  "src",
  "index.css",
);

const css = readFileSync(cssPath, "utf8");

/** Parse an hsl token value like "222.2 84% 4.9%" or with alpha suffix "/ 0.5". */
function parseVar(raw) {
  const match = raw.match(/^\s*(\d+(?:\.\d+)?)\s+(\d+(?:\.\d+)?)%\s+(\d+(?:\.\d+)?)%/);
  if (!match) return null;
  return { h: Number(match[1]), s: Number(match[2]), l: Number(match[3]) };
}

function hslToRgb({ h, s, l }) {
  const sn = s / 100;
  const ln = l / 100;
  const k = (n) => (n + h / 30) % 12;
  const a = sn * Math.min(ln, 1 - ln);
  const f = (n) =>
    ln - a * Math.max(-1, Math.min(k(n) - 3, Math.min(9 - k(n), 1)));
  return [f(0), f(8), f(4)].map((v) => Math.round(v * 255));
}

function relativeLuminance([r, g, b]) {
  const [lr, lg, lb] = [r, g, b].map((v) => {
    const c = v / 255;
    return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4;
  });
  return 0.2126 * lr + 0.7152 * lg + 0.0722 * lb;
}

function contrastRatio(foreground, background) {
  const l1 = relativeLuminance(foreground);
  const l2 = relativeLuminance(background);
  const [hi, lo] = l1 >= l2 ? [l1, l2] : [l2, l1];
  return (hi + 0.05) / (lo + 0.05);
}

/** Blend `alpha` of the foreground over the background (for soft surfaces). */
function blend(foreground, background, alpha) {
  return foreground.map((c, i) =>
    Math.round(c * alpha + background[i] * (1 - alpha)),
  );
}

const MIN_RATIO = 4.5;

/** Text/surface pairs by token name, with the alpha used for soft surfaces. */
const PAIRS = [
  { fg: "foreground", bg: "background" },
  { fg: "muted-foreground", bg: "background" },
  { fg: "muted-foreground", bg: "card" },
  { fg: "muted-foreground", bg: "muted" },
  { fg: "primary-foreground", bg: "primary" },
  { fg: "secondary-foreground", bg: "secondary" },
  { fg: "popover-foreground", bg: "popover" },
  { fg: "accent-foreground", bg: "accent" },
  { fg: "card-foreground", bg: "card" },
  { fg: "destructive-foreground", bg: "destructive" },
  {
    fg: "status-success-text",
    base: "status-success",
    bg: "card",
    softAlpha: 0.1,
  },
  {
    fg: "status-running-text",
    base: "status-running",
    bg: "card",
    softAlpha: 0.1,
  },
  {
    fg: "status-failed-text",
    base: "status-failed",
    bg: "card",
    softAlpha: 0.1,
  },
  {
    fg: "status-pending-text",
    base: "status-pending",
    bg: "card",
    softAlpha: 0.1,
  },
  { fg: "status-success-text", bg: "card" },
  { fg: "status-running-text", bg: "card" },
  { fg: "status-failed-text", bg: "card" },
  { fg: "status-pending-text", bg: "card" },
];

function extractBlock(css, selector) {
  const index = css.indexOf(selector);
  if (index === -1) return null;
  const open = css.indexOf("{", index);
  const close = css.indexOf("}", open);
  if (open === -1 || close === -1) return null;
  const body = css.slice(open + 1, close);
  const tokens = {};
  for (const match of body.matchAll(/--([\w-]+):\s*([^;]+);/g)) {
    tokens[match[1]] = match[2];
  }
  return tokens;
}

let failures = 0;
const missingDark = [];

for (const [label, selector] of [["light", ":root"], ["dark", ".dark"]]) {
  const tokens = extractBlock(css, selector);
  if (!tokens) {
    console.error(`✖ theme "${label}": ${selector} block not found`);
    failures += 1;
    continue;
  }

  // Every pair must be measured with real values — a missing or non-HSL token
  // is a gate failure, never a silent skip.
  const skipped = [];
  for (const pair of PAIRS) {
    if (!tokens[pair.fg]) skipped.push(`${pair.fg} (fg)`);
    if (!tokens[pair.bg]) skipped.push(`${pair.bg} (bg)`);
  }
  if (skipped.length > 0) {
    failures += 1;
    console.error(
      `✖ theme "${label}": missing tokens, cannot evaluate: ${skipped.join(", ")}`,
    );
    continue;
  }

  console.log(`\n${label} theme`);

  for (const pair of PAIRS) {
    const fg = parseVar(tokens[pair.fg]);
    const bg = parseVar(tokens[pair.bg]);
    if (!fg || !bg) {
      failures += 1;
      console.error(
        `✖ ${pair.fg} on ${pair.bg}: non-HSL value (${pair.fg}="${tokens[pair.fg]}", ${pair.bg}="${tokens[pair.bg]}")`,
      );
      continue;
    }
    const fgRgb = hslToRgb(fg);
    let bgRgb = hslToRgb(bg);
    if (pair.softAlpha && pair.base) {
      const base = parseVar(tokens[pair.base]);
      if (base) {
        bgRgb = blend(hslToRgb(base), bgRgb, pair.softAlpha);
      } else {
        failures += 1;
        console.error(
          `✖ ${pair.fg} on ${pair.bg}: missing soft-fill base token ${pair.base}`,
        );
        continue;
      }
    }
    const ratio = contrastRatio(fgRgb, bgRgb);
    const ok = ratio >= MIN_RATIO;
    if (!ok) failures += 1;
    console.log(
      `  ${ok ? "✔" : "✖"} ${pair.fg} on ${pair.bg}: ${ratio.toFixed(2)}:1 (min ${MIN_RATIO})`,
    );
  }
}

// Parity check: tokens present in :root must exist in .dark.
const lightTokens = extractBlock(css, ":root") ?? {};
const darkTokens = extractBlock(css, ".dark") ?? {};
const requiredInDark = [
  "foreground",
  "card-foreground",
  "popover-foreground",
  "primary",
  "primary-foreground",
  "secondary",
  "secondary-foreground",
  "muted",
  "muted-foreground",
  "accent",
  "accent-foreground",
  "destructive",
  "destructive-foreground",
  "border",
  "input",
  "ring",
  "status-success",
  "status-success-text",
  "status-running",
  "status-running-text",
  "status-failed",
  "status-failed-text",
  "status-pending",
  "status-pending-text",
];
for (const name of requiredInDark) {
  if (lightTokens[name] && !darkTokens[name]) {
    missingDark.push(name);
  }
}
if (missingDark.length > 0) {
  failures += 1;
  console.error(`\n✖ tokens missing from .dark: ${missingDark.join(", ")}`);
}

console.log(
  `\n${PAIRS.length * 2} pairs expected, ${failures} failures, ${missingDark.length} dark-parity gaps`,
);

if (failures > 0) process.exit(1);
