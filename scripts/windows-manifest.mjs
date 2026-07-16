import { writeFileSync } from "node:fs";

const [, , outputPath, rawVersion = "dev"] = process.argv;

if (!outputPath) {
  console.error("usage: node scripts/windows-manifest.mjs <output> [version]");
  process.exit(1);
}

const version = String(rawVersion || "dev");
const manifestVersion = /^\d+\.\d+\.\d+$/.test(version)
  ? `${version}.0`
  : /^\d+\.\d+\.\d+\.\d+$/.test(version)
    ? version
    : "0.0.0.0";

const manifest = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<assembly manifestVersion="1.0" xmlns="urn:schemas-microsoft-com:asm.v1" xmlns:asmv3="urn:schemas-microsoft-com:asm.v3">
    <assemblyIdentity type="win32" name="com.lsongdev.miya-desktop" version="${manifestVersion}" processorArchitecture="*"/>
    <dependency>
        <dependentAssembly>
            <assemblyIdentity type="win32" name="Microsoft.Windows.Common-Controls" version="6.0.0.0" processorArchitecture="*" publicKeyToken="6595b64144ccf1df" language="*"/>
        </dependentAssembly>
    </dependency>
    <asmv3:application>
        <asmv3:windowsSettings>
            <dpiAware xmlns="http://schemas.microsoft.com/SMI/2005/WindowsSettings">true/pm</dpiAware>
            <dpiAwareness xmlns="http://schemas.microsoft.com/SMI/2016/WindowsSettings">permonitorv2,permonitor</dpiAwareness>
        </asmv3:windowsSettings>
    </asmv3:application>
</assembly>
`;

writeFileSync(outputPath, manifest);
