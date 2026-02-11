import { NextRequest, NextResponse } from "next/server";
import { getPackageInfo, updatePackage, deletePackage } from "@/lib/db";
import { deletePackageBlobs } from "@/lib/blob";
import { requireAdmin } from "@/lib/auth";

export async function GET(
  _request: NextRequest,
  { params }: { params: Promise<{ name: string }> }
) {
  const { name } = await params;

  try {
    const info = await getPackageInfo(name);
    if (!info) {
      return NextResponse.json(
        { error: "Not Found", message: `Package '${name}' not found` },
        { status: 404 }
      );
    }
    return NextResponse.json(info);
  } catch (err) {
    return NextResponse.json(
      { error: "Internal Server Error", message: `Failed to get package info: ${err}` },
      { status: 500 }
    );
  }
}

export async function PUT(
  request: NextRequest,
  { params }: { params: Promise<{ name: string }> }
) {
  const authError = requireAdmin(request);
  if (authError) return authError;

  const { name } = await params;

  let updates: Record<string, string>;
  try {
    updates = await request.json();
  } catch {
    return NextResponse.json(
      { error: "Bad Request", message: "Invalid JSON" },
      { status: 400 }
    );
  }

  try {
    await updatePackage(name, updates);
    return NextResponse.json({ success: true, message: "Package updated successfully" });
  } catch (err) {
    const message = String(err);
    if (message.includes("not found")) {
      return NextResponse.json(
        { error: "Not Found", message },
        { status: 404 }
      );
    }
    if (message.includes("cannot be updated")) {
      return NextResponse.json(
        { error: "Bad Request", message },
        { status: 400 }
      );
    }
    return NextResponse.json(
      { error: "Internal Server Error", message: `Failed to update package: ${err}` },
      { status: 500 }
    );
  }
}

export async function DELETE(
  request: NextRequest,
  { params }: { params: Promise<{ name: string }> }
) {
  const authError = requireAdmin(request);
  if (authError) return authError;

  const { name } = await params;

  try {
    const versions = await deletePackage(name);
    // Best-effort blob cleanup
    try {
      await deletePackageBlobs(name, versions);
    } catch {
      // non-fatal
    }
    return NextResponse.json({ success: true, message: "Package deleted successfully" });
  } catch (err) {
    const message = String(err);
    if (message.includes("not found")) {
      return NextResponse.json(
        { error: "Not Found", message },
        { status: 404 }
      );
    }
    return NextResponse.json(
      { error: "Internal Server Error", message: `Failed to delete package: ${err}` },
      { status: 500 }
    );
  }
}
