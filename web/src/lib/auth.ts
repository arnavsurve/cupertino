import { NextRequest, NextResponse } from "next/server";

export function requireAdmin(request: NextRequest): NextResponse | null {
  const adminApiKey = process.env.ADMIN_API_KEY;

  if (!adminApiKey) {
    return NextResponse.json(
      { error: "Service Unavailable", message: "Admin API key not configured" },
      { status: 503 }
    );
  }

  let apiKey = request.headers.get("x-api-key") ?? "";
  if (!apiKey) {
    const authHeader = request.headers.get("authorization") ?? "";
    if (authHeader.startsWith("Bearer ")) {
      apiKey = authHeader.slice(7);
    }
  }

  if (!apiKey) {
    return NextResponse.json(
      { error: "Unauthorized", message: "API key required" },
      { status: 401 }
    );
  }

  if (apiKey !== adminApiKey) {
    return NextResponse.json(
      { error: "Unauthorized", message: "Invalid API key" },
      { status: 401 }
    );
  }

  return null;
}
