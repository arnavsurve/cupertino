import { NextResponse } from "next/server";
import { getStats } from "@/lib/db";

export async function GET() {
  const health: Record<string, unknown> = {
    status: "ok",
    timestamp: new Date().toISOString(),
    version: "1.0.0",
  };

  try {
    await getStats();
  } catch {
    health.status = "degraded";
    health.database = "error";
    return NextResponse.json(health, { status: 503 });
  }

  return NextResponse.json(health);
}
