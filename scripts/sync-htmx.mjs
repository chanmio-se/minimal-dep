import { createHash } from "node:crypto";
import { existsSync, readFileSync, writeFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const packageJsonPath = path.join(root, "package.json");
const packageLockPath = path.join(root, "package-lock.json");
const installedPackagePath = path.join(root, "node_modules/htmx.org/package.json");
const sourcePath = path.join(root, "node_modules/htmx.org/dist/htmx.min.js");
const vendoredPath = path.join(root, "web/static/js/htmx.min.js");
const adrPath = path.join(root, "docs/decisions/0001-standard-library-and-htmx.md");

function fail(message) {
  throw new Error(`sync-htmx: ${message}`);
}

const pkg = JSON.parse(readFileSync(packageJsonPath, "utf8"));
const version = pkg.devDependencies?.["htmx.org"] ?? pkg.dependencies?.["htmx.org"];
if (!/^\d+\.\d+\.\d+$/.test(version ?? "")) {
  fail("htmx.org must be pinned to an exact version in package.json");
}

if (!existsSync(installedPackagePath) || !existsSync(sourcePath)) {
  fail("node_modules/htmx.org is missing; run `npm ci` first");
}
const installedVersion = JSON.parse(readFileSync(installedPackagePath, "utf8")).version;
if (installedVersion !== version) {
  fail(`installed htmx.org is ${installedVersion}, but package.json pins ${version}; run \`npm ci --ignore-scripts\``);
}
const lock = JSON.parse(readFileSync(packageLockPath, "utf8"));
if (lock.packages?.["node_modules/htmx.org"]?.version !== version) {
  fail(`package-lock.json does not match the pinned version ${version}`);
}

const source = readFileSync(sourcePath);
const checksum = createHash("sha256").update(source).digest("hex");
let changed = false;

if (!readFileSync(vendoredPath).equals(source)) {
  writeFileSync(vendoredPath, source);
  changed = true;
}

const adrBefore = readFileSync(adrPath, "utf8");
const hashMatch = adrBefore.match(/SHA-256 \(htmx\.min\.js(?: [^)]+)?\):\n`[^`]+`/);
if (!hashMatch) {
  fail("ADR-0001 does not contain the expected SHA-256 line");
}

const adrAfter = adrBefore.replace(hashMatch[0], `SHA-256 (htmx.min.js):\n\`${checksum}\``);
if (adrAfter !== adrBefore) {
  writeFileSync(adrPath, adrAfter);
  changed = true;
}

console.log(`htmx.org ${version} (SHA-256 ${checksum})`);
console.log(changed ? "Updated the vendored asset and/or ADR-0001." : "Vendored asset and ADR-0001 are up to date.");
