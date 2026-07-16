import { writeFileSync } from "node:fs";

const [, , outputPath, rawVersion = "dev"] = process.argv;

if (!outputPath) {
  console.error("usage: node scripts/windows-info.mjs <output> [version]");
  process.exit(1);
}

const version = String(rawVersion || "dev");
const fileVersion = /^\d+\.\d+\.\d+$/.test(version)
  ? `${version}.0`
  : /^\d+\.\d+\.\d+\.\d+$/.test(version)
    ? version
    : "0.0.0.0";

const info = {
  fixed: {
    file_version: fileVersion,
    product_version: fileVersion,
  },
  info: {
    "0000": {
      ProductVersion: version,
      CompanyName: "Lsong",
      FileDescription: "Miya Desktop",
      LegalCopyright: "Copyright (c) 2026 Lsong",
      ProductName: "Miya Desktop",
      Comments: "AI agent desktop client",
    },
  },
};

writeFileSync(outputPath, `${JSON.stringify(info, null, "\t")}\n`);
