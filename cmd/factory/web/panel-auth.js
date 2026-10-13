// Read-only compatibility adapter for official CPAMC v1.26.1 saved auth.
// This is reversible obfuscation, not encryption. Never persist a key here.
(function (root) {
  "use strict";

  function readPanelAuth(win) {
    try {
      if (
        win.self === win.top ||
        win.parent.location.origin !== win.location.origin
      ) {
        return null;
      }

      // Try parent window first (typical iframe scenario), then fallback to local window
      let raw = null;
      try {
        if (win.parent && win.parent.localStorage) {
          raw = win.parent.localStorage.getItem("cli-proxy-auth");
        }
      } catch (_) {}

      if (!raw) {
        try {
          raw = win.localStorage.getItem("cli-proxy-auth");
        } catch (_) {}
      }

      if (!raw) return null;

      if (raw.startsWith("enc::v1::")) {
        const bytes = Uint8Array.from(atob(raw.slice(9)), (c) =>
          c.charCodeAt(0),
        );
        const salt = new TextEncoder().encode(
          "cli-proxy-api-webui::secure-storage|" +
            win.location.host +
            "|" +
            win.navigator.userAgent,
        );
        raw = new TextDecoder().decode(
          bytes.map((b, i) => b ^ salt[i % salt.length]),
        );
      }

      const parsed = JSON.parse(raw);
      const state = parsed.state || parsed;

      if (
        typeof state.managementKey !== "string" ||
        !state.managementKey.trim() ||
        typeof state.apiBase !== "string"
      ) {
        return null;
      }

      // Never forward the panel key to another origin or an arbitrary URL.
      if (
        new URL(state.apiBase, win.location.href).origin !== win.location.origin
      ) {
        return null;
      }

      return { key: state.managementKey.trim() };
    } catch (_) {
      return null;
    }
  }

  root.FactoryPanelAuth = { read: readPanelAuth };
})(globalThis);
