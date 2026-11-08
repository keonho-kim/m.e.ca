// Purpose: Keep explicitly local demo state consistent across related example pages.
// Pattern: Session-storage I/O boundary with caller-owned data validation.
// Usage: Domain modules and page controllers commit their own current-tab namespaces.
// Related: internal/config/bootstrap/defaults/application-design/assets/examples/morrow/booking-data.js, internal/config/bootstrap/defaults/application-design/assets/examples/archive/archive.js.
(() => {
  const scopes = new Set(['booking','archive']);
  function key(scope) {
    if (!scopes.has(scope)) throw new Error('지원하지 않는 데모 저장 범위입니다.');
    return 'meca-application-example:' + scope;
  }
  function read(scope, initial, validate) {
    let raw;
    try { raw = sessionStorage.getItem(key(scope)); }
    catch { throw new Error('이 브라우저에서 세션 저장을 사용할 수 없습니다. 로컬 HTTP 미리보기로 열어 주세요.'); }
    if (raw === null) return structuredClone(initial);
    let value;
    try { value = JSON.parse(raw); }
    catch { throw new Error('저장된 데모 자료를 읽지 못했습니다. 자동으로 덮어쓰지 않았습니다.'); }
    if (!value || value.version !== 1 || !validate(value)) throw new Error('지원하지 않는 데모 자료 형식입니다. 자동 변환하거나 삭제하지 않았습니다.');
    return value;
  }
  function write(scope, value) {
    try { sessionStorage.setItem(key(scope), JSON.stringify(value)); }
    catch { throw new Error('데모를 저장하지 못했습니다. 입력은 현재 화면에 남아 있습니다. 저장 공간과 브라우저 설정을 확인해 주세요.'); }
  }
  window.ExampleState = Object.freeze({ read, write });
})();
