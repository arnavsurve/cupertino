import { put, del } from "@vercel/blob";

export async function uploadPackageBlob(
  name: string,
  version: string,
  file: Blob
): Promise<string> {
  const pathname = `packages/${name}-${version}.tar.gz`;
  const blob = await put(pathname, file, {
    access: "public",
    contentType: "application/gzip",
  });
  return blob.url;
}

export async function deletePackageBlobs(
  name: string,
  versions: string[]
): Promise<void> {
  const urls = versions.map((v) => `packages/${name}-${v}.tar.gz`);
  await Promise.allSettled(urls.map((url) => del(url)));
}
