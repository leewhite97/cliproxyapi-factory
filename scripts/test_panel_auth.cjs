const assert = require("node:assert/strict");
require("../cmd/factory/web/panel-auth.js");
const read = globalThis.FactoryPanelAuth.read;
function fixture(state, encoded = false) {
  const w = {
    location: {
      origin: "https://local.test",
      host: "local.test",
      href: "https://local.test/v0/resource/plugins/factory/quota",
    },
    navigator: { userAgent: "test-agent" },
    parent: { location: { origin: "https://local.test" } },
  };
  w.self = w;
  w.top = {};
  let raw = JSON.stringify({ state });
  if (encoded) {
    const key = Buffer.from(
      "cli-proxy-api-webui::secure-storage|local.test|test-agent",
    );
    raw =
      "enc::v1::" +
      Buffer.from(
        Buffer.from(raw).map((b, i) => b ^ key[i % key.length]),
      ).toString("base64");
  }
  w.localStorage = {
    getItem: () => raw,
    setItem: () => {
      throw Error("must never write storage");
    },
  };
  return w;
}
const state = { apiBase: "https://local.test", managementKey: "synthetic-key" };
assert.deepEqual(read(fixture(state)), { key: "synthetic-key" });
assert.deepEqual(read(fixture(state, true)), { key: "synthetic-key" });
assert.equal(
  read(fixture({ ...state, apiBase: "https://untrusted.test" })),
  null,
);
assert.equal(read(fixture({ apiBase: state.apiBase })), null);
let w = fixture(state);
w.top = w;
assert.equal(read(w), null);
w = fixture(state);
w.parent.location.origin = "https://untrusted.test";
assert.equal(read(w), null);
w = fixture(state);
w.localStorage.getItem = () => "{bad";
assert.equal(read(w), null);
w = fixture(state);
w.localStorage.getItem = () => {
  throw Error("blocked");
};
assert.equal(read(w), null);
console.log(
  "Panel auth: 8 cases passed; read-only, same-origin, malformed/missing state handled",
);
