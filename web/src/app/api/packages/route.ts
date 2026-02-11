import { NextRequest, NextResponse } from "next/server";
import { listPackages, addPackage } from "@/lib/db";
import { requireAdmin } from "@/lib/auth";
import { uploadPackageBlob } from "@/lib/blob";
import type { PackageUpload } from "@/lib/types";
import { createHash } from "crypto";

export async function GET(request: NextRequest) {
  const { searchParams } = request.nextUrl;

  let limit = Number(searchParams.get("limit")) || 50;
  if (limit <= 0 || limit > 100) limit = 50;

  let offset = Number(searchParams.get("offset")) || 0;
  if (offset < 0) offset = 0;

  try {
    const packages = await listPackages(limit, offset);
    return NextResponse.json(packages);
  } catch (err) {
    return NextResponse.json(
      { error: "Internal Server Error", message: `Failed to list packages: ${err}` },
      { status: 500 }
    );
  }
}

export async function POST(request: NextRequest) {
  const authError = requireAdmin(request);
  if (authError) return authError;

  let formData: FormData;
  try {
    formData = await request.formData();
  } catch {
    return NextResponse.json(
      { error: "Bad Request", message: "Failed to parse multipart form" },
      { status: 400 }
    );
  }

  const metadataStr = formData.get("metadata");
  if (!metadataStr || typeof metadataStr !== "string") {
    return NextResponse.json(
      { error: "Bad Request", message: "Metadata is required" },
      { status: 400 }
    );
  }

  let upload: PackageUpload;
  try {
    upload = JSON.parse(metadataStr);
  } catch {
    return NextResponse.json(
      { error: "Bad Request", message: "Invalid metadata JSON" },
      { status: 400 }
    );
  }

  if (!upload.name || !upload.version || !upload.description || !upload.files || Object.keys(upload.files).length === 0) {
    return NextResponse.json(
      { error: "Bad Request", message: "name, version, description, and files are required" },
      { status: 400 }
    );
  }

  const file = formData.get("file");
  if (!file || !(file instanceof Blob)) {
    return NextResponse.json(
      { error: "Bad Request", message: "File upload is required" },
      { status: 400 }
    );
  }

  const buffer = Buffer.from(await file.arrayBuffer());
  const checksum = createHash("sha256").update(buffer).digest("hex");
  const size = buffer.length;

  let blobUrl: string;
  try {
    blobUrl = await uploadPackageBlob(upload.name, upload.version, new Blob([buffer]));
  } catch (err) {
    return NextResponse.json(
      { error: "Internal Server Error", message: `Failed to upload file: ${err}` },
      { status: 500 }
    );
  }

  const baseUrl = process.env.BASE_URL ?? request.nextUrl.origin;
  const downloadUrl = `${baseUrl}/packages/${upload.name}-${upload.version}.tar.gz`;

  try {
    const pkg = await addPackage({ upload, checksum, size, downloadUrl });
    return NextResponse.json(
      { success: true, data: pkg, message: "Package uploaded successfully" },
      { status: 201 }
    );
  } catch (err) {
    const message = String(err);
    if (message.includes("already exists") || message.includes("unique") || message.includes("duplicate")) {
      return NextResponse.json(
        { error: "Conflict", message: `Package ${upload.name} version ${upload.version} already exists` },
        { status: 409 }
      );
    }
    return NextResponse.json(
      { error: "Internal Server Error", message: `Failed to add package: ${err}` },
      { status: 500 }
    );
  }
}
