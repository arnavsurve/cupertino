import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  async rewrites() {
    return [
      {
        // Backward-compatible download URLs matching the Go registry pattern
        source: "/packages/:slug.tar.gz",
        destination: "/api/download/:slug.tar.gz",
      },
    ];
  },
};

export default nextConfig;
