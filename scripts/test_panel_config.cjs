const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const source = fs.readFileSync(require.resolve('../cmd/factory/web/quota.js'), 'utf8');
const html = fs.readFileSync(require.resolve('../cmd/factory/web/quota.html'), 'utf8');
function element() {
  return { value: '', checked: false, disabled: false, children: [], textContent: '', style: {},
    classList: { add() {}, remove() {} }, handlers: {},
    addEventListener(type, fn) { this.handlers[type] = fn; },
    append(...nodes) { this.children.push(...nodes); }, appendChild(node) { this.children.push(node); },
    replaceChildren(...nodes) { this.children = nodes; }, remove() {} };
}
const flush = () => new Promise(resolve => setImmediate(resolve));
async function fixture({ missing = false, readOnly = false } = {}) {
  const nodes = new Map(); const calls = []; let init;
  let stored = { enabled: false, catalog_cache_path: 'test-cache.json', future_option: 42,
    compatibility: { mode: 'factory', sensitive_words: ['Custom'], obfuscate_user_text: false } };
  const document = { getElementById(id) { if (!nodes.has(id)) nodes.set(id, element()); return nodes.get(id); },
    createElement: element, querySelectorAll: () => [] };
  const window = { addEventListener(event, fn) { if (event === 'DOMContentLoaded') init = fn; } };
  const ctx = { document, window, FormData, Blob, setTimeout: () => 0,
    FactoryPanelAuth: { read: () => ({ key: 'synthetic-management' }) },
    fetch: async (path, options) => {
      calls.push({ path, options });
      if (path.endsWith('/credentials')) return { ok: true, status: 200, json: async () => ({ files: [] }) };
      if (!options.body) {
        if (missing) return { ok: false, status: 500, json: async () => ({ error: 'read_failed' }) };
        return { ok: true, status: 200, json: async () => structuredClone(stored) };
      }
      if (readOnly) return { ok: false, status: 500, json: async () => ({ error: 'write_failed', message: 'read-only file system' }) };
      const payload = JSON.parse(options.body);
      if (options.method === 'PATCH') stored = { ...stored, ...payload, compatibility: { ...stored.compatibility, ...payload.compatibility } };
      else stored = payload;
      return { ok: true, status: 200, json: async () => ({ status: 'ok' }) };
    } };
  vm.runInNewContext(source, ctx); init(); await flush(); await flush();
  return { nodes, calls, get stored() { return stored; } };
}
(async () => {
  const modeValues = [...html.matchAll(/<option value="([^"]+)"/g)].map(m => m[1]);
  assert.deepEqual(modeValues, ['factory', 'passthrough'], 'UI advertises backend-invalid modes');
  const f = await fixture();
  assert.equal(f.nodes.get('select-compat-mode').value, 'factory');
  await f.nodes.get('btn-save-compat').handlers.click(); await flush();
  const write = f.calls.find(c => c.options.body);
  assert.equal(write.options.method, 'PATCH', 'save must not replace unrelated plugin settings');
  assert.equal(f.stored.enabled, false); assert.equal(f.stored.catalog_cache_path, 'test-cache.json'); assert.equal(f.stored.future_option, 42);
  assert.ok(f.calls.filter(c => !c.options.body && c.path.includes('/config/')).length >= 2, 'success must read back exact config');
  const p = await fixture(); p.nodes.get('select-compat-mode').value = 'passthrough'; p.nodes.get('check-obfuscate-user').checked = true;
  await p.nodes.get('btn-save-compat').handlers.click(); await flush();
  assert.deepEqual(p.stored.compatibility.sensitive_words, []); assert.equal(p.stored.compatibility.obfuscate_user_text, false);
  const m = await fixture({ missing: true }); assert.equal(m.nodes.get('btn-save-compat').disabled, true, 'failed load must block save');
  const r = await fixture({ readOnly: true }); await r.nodes.get('btn-save-compat').handlers.click(); await flush();
  const messages = r.nodes.get('toast-container').children.flatMap(t => t.children.map(c => c.textContent)).join(' ');
  assert.ok(messages.includes('只读')); assert.ok(!messages.includes('已保存'));
  console.log('Panel config: modes, partial patch, readback, passthrough, read failure and read-only failure passed');
})().catch(err => { console.error(err); process.exitCode = 1; });
