import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Cupertino Package Registry",
  description: "A package registry for Cupertino",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body>
        <header>
          <h1>
            <a href="/" style={{ textDecoration: "none", color: "inherit" }}>
              Cupertino Package Registry
            </a>
          </h1>
          <nav>
            <a href="/">packages</a>
            <a href="/api/packages">api</a>
            <a href="/api/health">status</a>
          </nav>
        </header>

        <main>{children}</main>

        <footer>
          <a href="/">back to packages</a>
          <a href="/api/packages">api</a>
        </footer>
      </body>
    </html>
  );
}
