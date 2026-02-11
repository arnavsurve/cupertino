"use client";

import { useState } from "react";

export default function CopyButton({ command }: { command: string }) {
  const [label, setLabel] = useState("copy");

  async function handleCopy() {
    try {
      await navigator.clipboard.writeText(command);
      setLabel("copied");
    } catch {
      // fallback
      const textarea = document.createElement("textarea");
      textarea.value = command;
      textarea.style.position = "fixed";
      textarea.style.left = "-999999px";
      document.body.appendChild(textarea);
      textarea.select();
      try {
        document.execCommand("copy");
        setLabel("copied");
      } catch {
        setLabel("failed");
      }
      document.body.removeChild(textarea);
    }
    setTimeout(() => setLabel("copy"), 1000);
  }

  return (
    <button className="copy-btn" onClick={handleCopy}>
      {label}
    </button>
  );
}
