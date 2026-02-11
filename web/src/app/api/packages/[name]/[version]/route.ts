import { NextRequest, NextResponse } from "next/server";
import { getPackage } from "@/lib/db";

export async function GET(
  _request: NextRequest,
  { params }: { params: Promise<{ name: string; version: string }> }
) {
  const { name, version } = await params;

  try {
    const pkg = await getPackage(name, version);
    if (!pkg) {
      return NextResponse.json(
        { error: "Not Found", message: `Package '${name}' version '${version}' not found` },
        { status: 404 }
      );
    }
    return NextResponse.json(pkg);
  } catch (err) {
    return NextResponse.json(
      { error: "Internal Server Error", message: `Failed to get package: ${err}` },
      { status: 500 }
    );
  }
}
