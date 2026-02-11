import { NextRequest, NextResponse } from "next/server";
import { searchPackages } from "@/lib/db";

export async function GET(request: NextRequest) {
  const { searchParams } = request.nextUrl;
  const q = searchParams.get("q");

  if (!q) {
    return NextResponse.json(
      { error: "Bad Request", message: "Query parameter 'q' is required" },
      { status: 400 }
    );
  }

  let limit = Number(searchParams.get("limit")) || 20;
  if (limit <= 0 || limit > 100) limit = 20;

  let offset = Number(searchParams.get("offset")) || 0;
  if (offset < 0) offset = 0;

  try {
    const results = await searchPackages(q, limit, offset);
    return NextResponse.json(results);
  } catch (err) {
    return NextResponse.json(
      { error: "Internal Server Error", message: `Failed to search packages: ${err}` },
      { status: 500 }
    );
  }
}
