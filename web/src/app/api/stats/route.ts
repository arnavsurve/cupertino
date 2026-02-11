import { NextResponse } from "next/server";
import { getStats } from "@/lib/db";

export async function GET() {
  try {
    const stats = await getStats();
    return NextResponse.json(stats);
  } catch (err) {
    return NextResponse.json(
      { error: "Internal Server Error", message: `Failed to get stats: ${err}` },
      { status: 500 }
    );
  }
}
