import { notFound } from "next/navigation";
import { getPackageInfo, getPackage } from "@/lib/db";
import type { Package } from "@/lib/types";
import CopyButton from "@/components/CopyButton";

export const dynamic = "force-dynamic";

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const unit = 1024;
  if (bytes < unit) return `${bytes} B`;

  let div = unit;
  let exp = 0;
  for (let n = Math.floor(bytes / unit); n >= unit; n = Math.floor(n / unit)) {
    div *= unit;
    exp++;
  }

  const units = ["B", "KB", "MB", "GB", "TB"];
  return `${(bytes / div).toFixed(1)} ${units[exp + 1]}`;
}

export default async function PackageDetailPage({
  params,
}: {
  params: Promise<{ name: string }>;
}) {
  const { name } = await params;

  const info = await getPackageInfo(name);
  if (!info) notFound();

  const packages: Package[] = [];
  for (const version of info.versions) {
    const pkg = await getPackage(name, version);
    if (pkg) packages.push(pkg);
  }

  return (
    <>
      <h1>{info.name}</h1>
      <p>{info.description}</p>

      <div className="stats">
        <p>latest: {info.latest}</p>
        <p>downloads: {info.downloads}</p>
        {info.license && <p>license: {info.license}</p>}
        {info.homepage && (
          <p>
            homepage: <a href={info.homepage}>{info.homepage}</a>
          </p>
        )}
      </div>

      <h2>install</h2>
      <div className="install-command">cupertino install {info.name}</div>
      <br />
      <div className="install-command">
        cupertino install {info.name}@{info.latest}
      </div>

      <h2>versions</h2>
      {packages.length > 0 ? (
        packages.map((pkg) => (
          <div key={pkg.version} className="version">
            <div className="version-header">
              v{pkg.version} -{" "}
              {new Date(pkg.upload_date).toISOString().slice(0, 10)}
            </div>

            <div className="version-info">
              <div>size: {formatBytes(pkg.size)}</div>
              <div>
                checksum: <span className="checksum">{pkg.checksum}</span>
              </div>
            </div>

            {pkg.dependencies && Object.keys(pkg.dependencies).length > 0 && (
              <div className="dependencies">
                <strong>dependencies:</strong>
                <ul>
                  {Object.entries(pkg.dependencies).map(([depName, depVer]) => (
                    <li key={depName}>
                      {depName}: {depVer}
                    </li>
                  ))}
                </ul>
              </div>
            )}

            {pkg.files && Object.keys(pkg.files).length > 0 && (
              <div className="files">
                <strong>files:</strong>
                {Object.entries(pkg.files).map(([src, dst]) => (
                  <div key={src} className="file-mapping">
                    {src} → {dst}
                  </div>
                ))}
              </div>
            )}

            <div>
              <a href={pkg.download_url}>download</a> |{" "}
              <span className="install-command">
                cupertino install {pkg.name}@{pkg.version}
              </span>
              <CopyButton
                command={`cupertino install ${pkg.name}@${pkg.version}`}
              />
            </div>
          </div>
        ))
      ) : (
        <p>no versions available</p>
      )}
    </>
  );
}
