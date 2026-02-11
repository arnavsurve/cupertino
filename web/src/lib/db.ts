import { neon } from "@neondatabase/serverless";
import type { Package, PackageInfo, PackageUpload, RegistryStats } from "./types";

let tablesEnsured = false;

function getClient() {
  const url = process.env.DATABASE_URL ?? process.env.POSTGRES_URL;
  if (!url) {
    throw new Error("No database connection string was provided. Set DATABASE_URL or POSTGRES_URL.");
  }
  return neon(url);
}

async function withTables() {
  if (!tablesEnsured) {
    await ensureTables();
    tablesEnsured = true;
  }
  return getClient();
}

export async function ensureTables() {
  const sql = getClient();
  await sql`
    CREATE TABLE IF NOT EXISTS packages (
      id SERIAL PRIMARY KEY,
      name TEXT NOT NULL,
      version TEXT NOT NULL,
      description TEXT NOT NULL,
      homepage TEXT,
      license TEXT,
      dependencies JSONB DEFAULT '{}',
      files JSONB NOT NULL DEFAULT '{}',
      checksum TEXT NOT NULL,
      size BIGINT NOT NULL,
      upload_date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
      download_url TEXT NOT NULL,
      downloads INTEGER DEFAULT 0,
      UNIQUE(name, version)
    )
  `;
  await sql`CREATE INDEX IF NOT EXISTS idx_packages_name ON packages(name)`;
  await sql`CREATE INDEX IF NOT EXISTS idx_packages_upload_date ON packages(upload_date)`;
  await sql`
    CREATE TABLE IF NOT EXISTS download_stats (
      id SERIAL PRIMARY KEY,
      package_name TEXT NOT NULL,
      package_version TEXT NOT NULL,
      download_date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
      ip_address TEXT,
      user_agent TEXT
    )
  `;
  await sql`CREATE INDEX IF NOT EXISTS idx_download_stats_package ON download_stats(package_name, package_version)`;
  await sql`CREATE INDEX IF NOT EXISTS idx_download_stats_date ON download_stats(download_date)`;
}

export async function addPackage(pkg: {
  upload: PackageUpload;
  checksum: string;
  size: number;
  downloadUrl: string;
}): Promise<Package> {
  const sql = await withTables();
  const rows = await sql`
    INSERT INTO packages (name, version, description, homepage, license,
                          dependencies, files, checksum, size, upload_date, download_url)
    VALUES (${pkg.upload.name}, ${pkg.upload.version}, ${pkg.upload.description},
            ${pkg.upload.homepage ?? null}, ${pkg.upload.license ?? null},
            ${JSON.stringify(pkg.upload.dependencies ?? {})}::jsonb,
            ${JSON.stringify(pkg.upload.files)}::jsonb,
            ${pkg.checksum}, ${pkg.size}, NOW(), ${pkg.downloadUrl})
    RETURNING *
  `;
  return rowToPackage(rows[0]);
}

export async function getPackage(name: string, version: string): Promise<Package | null> {
  const sql = await withTables();
  const rows = await sql`
    SELECT name, version, description, homepage, license, dependencies,
           files, checksum, size, upload_date, download_url, downloads
    FROM packages
    WHERE name = ${name} AND version = ${version}
  `;
  if (rows.length === 0) return null;
  return rowToPackage(rows[0]);
}

export async function getPackageInfo(name: string): Promise<PackageInfo | null> {
  const sql = await withTables();
  const infoRows = await sql`
    SELECT DISTINCT name, description, homepage, license
    FROM packages
    WHERE name = ${name}
    LIMIT 1
  `;
  if (infoRows.length === 0) return null;

  const row = infoRows[0];
  const versionRows = await sql`
    SELECT version, downloads
    FROM packages
    WHERE name = ${name}
    ORDER BY upload_date DESC
  `;

  const versions: string[] = [];
  let totalDownloads = 0;
  for (const v of versionRows) {
    versions.push(v.version as string);
    totalDownloads += Number(v.downloads ?? 0);
  }

  return {
    name: row.name as string,
    description: row.description as string,
    homepage: (row.homepage as string) || undefined,
    license: (row.license as string) || undefined,
    versions,
    latest: versions[0] ?? "",
    downloads: totalDownloads,
  };
}

export async function listPackages(limit: number, offset: number): Promise<PackageInfo[]> {
  const sql = await withTables();
  const rows = await sql`
    SELECT name, description, homepage, license,
           STRING_AGG(version, ',' ORDER BY upload_date DESC) as versions,
           SUM(downloads)::int as total_downloads
    FROM packages
    GROUP BY name, description, homepage, license
    ORDER BY name
    LIMIT ${limit} OFFSET ${offset}
  `;

  return rows.map((row) => {
    const versions = (row.versions as string)?.split(",") ?? [];
    return {
      name: row.name as string,
      description: row.description as string,
      homepage: (row.homepage as string) || undefined,
      license: (row.license as string) || undefined,
      versions,
      latest: versions[0] ?? "",
      downloads: Number(row.total_downloads ?? 0),
    };
  });
}

export async function searchPackages(
  query: string,
  limit: number,
  offset: number
): Promise<PackageInfo[]> {
  const sql = await withTables();
  const searchTerm = `%${query}%`;
  const exactTerm = `${query}%`;

  const rows = await sql`
    SELECT name, description, homepage, license,
           STRING_AGG(version, ',' ORDER BY upload_date DESC) as versions,
           SUM(downloads)::int as total_downloads
    FROM packages
    WHERE name ILIKE ${searchTerm} OR description ILIKE ${searchTerm}
    GROUP BY name, description, homepage, license
    ORDER BY
      CASE
        WHEN name ILIKE ${exactTerm} THEN 1
        WHEN description ILIKE ${exactTerm} THEN 2
        ELSE 3
      END,
      name
    LIMIT ${limit} OFFSET ${offset}
  `;

  return rows.map((row) => {
    const versions = (row.versions as string)?.split(",") ?? [];
    return {
      name: row.name as string,
      description: row.description as string,
      homepage: (row.homepage as string) || undefined,
      license: (row.license as string) || undefined,
      versions,
      latest: versions[0] ?? "",
      downloads: Number(row.total_downloads ?? 0),
    };
  });
}

export async function updatePackage(
  name: string,
  updates: Record<string, string>
): Promise<boolean> {
  const sql = await withTables();
  const allowedFields = ["description", "homepage", "license"];

  for (const field of Object.keys(updates)) {
    if (!allowedFields.includes(field)) {
      throw new Error(`field ${field} cannot be updated`);
    }
  }

  // Build individual updates since neon tagged template doesn't support dynamic column names easily
  for (const [field, value] of Object.entries(updates)) {
    if (field === "description") {
      await sql`UPDATE packages SET description = ${value} WHERE name = ${name}`;
    } else if (field === "homepage") {
      await sql`UPDATE packages SET homepage = ${value} WHERE name = ${name}`;
    } else if (field === "license") {
      await sql`UPDATE packages SET license = ${value} WHERE name = ${name}`;
    }
  }

  return true;
}

export async function deletePackage(name: string): Promise<string[]> {
  const sql = await withTables();

  const rows = await sql`
    SELECT version FROM packages WHERE name = ${name}
  `;
  if (rows.length === 0) {
    throw new Error(`package ${name} not found`);
  }

  const versions = rows.map((r) => r.version as string);

  await sql`DELETE FROM packages WHERE name = ${name}`;
  await sql`DELETE FROM download_stats WHERE package_name = ${name}`;

  return versions;
}

export async function incrementDownload(
  packageName: string,
  version: string,
  ipAddress: string,
  userAgent: string
): Promise<void> {
  const sql = await withTables();
  await sql`
    UPDATE packages SET downloads = downloads + 1
    WHERE name = ${packageName} AND version = ${version}
  `;
  await sql`
    INSERT INTO download_stats (package_name, package_version, download_date, ip_address, user_agent)
    VALUES (${packageName}, ${version}, NOW(), ${ipAddress}, ${userAgent})
  `;
}

export async function getStats(): Promise<RegistryStats> {
  const sql = await withTables();
  const statsRow = await sql`
    SELECT COUNT(DISTINCT name)::int as total_packages,
           COALESCE(SUM(downloads), 0)::int as total_downloads
    FROM packages
  `;
  const versionRow = await sql`SELECT COUNT(*)::int as total_versions FROM packages`;

  return {
    total_packages: Number(statsRow[0]?.total_packages ?? 0),
    total_downloads: Number(statsRow[0]?.total_downloads ?? 0),
    total_versions: Number(versionRow[0]?.total_versions ?? 0),
  };
}

function rowToPackage(row: Record<string, unknown>): Package {
  return {
    name: row.name as string,
    version: row.version as string,
    description: row.description as string,
    homepage: (row.homepage as string) || undefined,
    license: (row.license as string) || undefined,
    dependencies: (row.dependencies as Record<string, string>) ?? {},
    files: (row.files as Record<string, string>) ?? {},
    checksum: row.checksum as string,
    size: Number(row.size),
    upload_date: (row.upload_date as Date)?.toISOString?.() ?? String(row.upload_date),
    download_url: row.download_url as string,
    downloads: Number(row.downloads ?? 0),
  };
}
