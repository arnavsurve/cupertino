export interface Package {
  id?: number;
  name: string;
  version: string;
  description: string;
  homepage?: string;
  license?: string;
  dependencies?: Record<string, string>;
  files: Record<string, string>;
  checksum: string;
  size: number;
  upload_date: string;
  download_url: string;
  downloads?: number;
}

export interface PackageInfo {
  name: string;
  description: string;
  homepage?: string;
  license?: string;
  versions: string[];
  latest: string;
  downloads: number;
}

export interface PackageUpload {
  name: string;
  version: string;
  description: string;
  homepage?: string;
  license?: string;
  dependencies?: Record<string, string>;
  files: Record<string, string>;
}

export interface RegistryStats {
  total_packages: number;
  total_downloads: number;
  total_versions: number;
}
