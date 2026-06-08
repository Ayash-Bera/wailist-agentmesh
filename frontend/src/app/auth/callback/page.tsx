"use client";
import { useEffect } from "react";
import { useRouter } from "next/navigation";

// Landing point for backend OAuth: the Go server redirects here with the JWT in
// the URL fragment (#token=<jwt>). Fragments never reach the server, so the token
// stays out of access logs and Referer headers. We persist it the same way
// email/password login does, scrub it from history, then move on.
export default function AuthCallbackPage() {
  const router = useRouter();

  useEffect(() => {
    const token = new URLSearchParams(window.location.hash.slice(1)).get("token");
    // Drop the fragment from the URL/history before navigating away.
    window.history.replaceState({}, "", window.location.pathname);
    if (token) {
      localStorage.setItem("agentmesh_signed_in", "1");
      localStorage.setItem("agentmesh_token", token);
      router.replace("/workflows");
    } else {
      router.replace("/signin?error=oauth");
    }
  }, [router]);

  return (
    <div style={{
      height: "100vh", display: "flex", alignItems: "center", justifyContent: "center",
      background: "var(--bg)", color: "var(--fg-muted)", fontFamily: "var(--font-mono)", fontSize: 13,
    }}>
      Signing you in…
    </div>
  );
}
