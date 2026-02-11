import { listPackages, getStats } from "@/lib/db";
import type { PackageInfo, RegistryStats } from "@/lib/types";
import SearchBar from "@/components/SearchBar";

export const dynamic = "force-dynamic";

export default async function Home() {
  let packages: PackageInfo[] = [];
  let stats: RegistryStats = { total_packages: 0, total_versions: 0, total_downloads: 0 };

  try {
    [packages, stats] = await Promise.all([
      listPackages(20, 0),
      getStats(),
    ]);
  } catch {
    // DB not connected yet — render empty state
  }

  return (
    <>
      <div className="stats">
        <p>packages: {stats.total_packages}</p>
        <p>versions: {stats.total_versions}</p>
        <p>downloads: {stats.total_downloads}</p>
      </div>

      <h2>search</h2>
      <SearchBar />

      <h2>packages</h2>
      {packages.length > 0 ? (
        packages.map((pkg) => (
          <div key={pkg.name} className="package">
            <div className="package-name">
              <a href={`/packages/${pkg.name}`}>{pkg.name}</a>
            </div>
            <div className="package-description">{pkg.description}</div>
            <div className="package-meta">
              version: {pkg.latest} | downloads: {pkg.downloads}
              {pkg.license ? ` | license: ${pkg.license}` : ""}
            </div>
            <div className="install-command">cupertino install {pkg.name}</div>
          </div>
        ))
      ) : (
        <p>no packages available</p>
      )}

      <h2>usage</h2>
      <p>install the cupertino package manager, then:</p>
      <div className="install-command">cupertino search &lt;package&gt;</div>
      <br />
      <div className="install-command">cupertino install &lt;package&gt;</div>
      <br />
      <div className="install-command">
        cupertino install &lt;package&gt;@&lt;version&gt;
      </div>
    </>
  );
}
