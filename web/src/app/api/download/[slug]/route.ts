import { NextRequest, NextResponse } from "next/server";
import { getPackage, incrementDownload } from "@/lib/db";

export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ slug: string }> }
) {
  const { slug } = await params;

  // slug format: "name-version.tar.gz" or "name-version"
  const cleaned = slug.replace(/\.tar\.gz$/, "");
  const lastDash = cleaned.lastIndexOf("-");
  if (lastDash === -1) {
    return NextResponse.json(
      { error: "Bad Request", message: "Invalid download path" },
      { status: 400 }
    );
  }

  const name = cleaned.slice(0, lastDash);
  const version = cleaned.slice(lastDash + 1);

  const pkg = await getPackage(name, version);
  if (!pkg) {
    return NextResponse.json(
      { error: "Not Found", message: "Package not found" },
      { status: 404 }
    );
  }

  // Record download stats in the background
  const clientIp =
    request.headers.get("x-forwarded-for")?.split(",")[0]?.trim() ??
    request.headers.get("x-real-ip") ??
    "unknown";
  const userAgent = request.headers.get("user-agent") ?? "";

  // Fire and forget
  incrementDownload(name, version, clientIp, userAgent).catch(() => {});

  // Redirect to the blob URL (the actual file location)
  return NextResponse.redirect(pkg.download_url, 302);
}
