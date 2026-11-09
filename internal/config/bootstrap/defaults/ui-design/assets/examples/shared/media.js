// Purpose: Reveal decoded original media over genuine derived previews with stable failures.
// Pattern: Image-owned progressive enhancement.
// Usage: Static and newly added image cards call enhance without external image services.
// Related: internal/config/bootstrap/defaults/ui-design/assets/examples/shared/motion.css, internal/config/bootstrap/defaults/ui-design/assets/examples/archive/archive.js.
(() => {
  const owner = new AbortController();
  const seen = new WeakSet();
  function enhance(root = document) {
    for (const image of root.querySelectorAll('img[data-blur-up]')) {
      if (seen.has(image) || !image.getAttribute('src')) continue;
      seen.add(image);
      const frame = image.closest('.media');
      const fallback = frame.querySelector('.fallback');
      const source = () => image.currentSrc || image.src;
      let cleanup=()=>{};
      const fail = () => {
        cleanup();
        if (owner.signal.aborted || !image.isConnected) return;
        frame.classList.remove('is-loading','is-ready'); frame.classList.add('is-error');
        if (fallback) fallback.hidden = false;
      };
      const ready = async () => {
        cleanup();
        const expected = source();
        try {
          await image.decode();
          if (owner.signal.aborted || !image.isConnected || source() !== expected) return;
          frame.classList.remove('is-loading','is-error'); frame.classList.add('is-ready');
          if (fallback) fallback.hidden = true;
        } catch { if (source() === expected) fail(); }
      };
      cleanup=()=>{image.removeEventListener('load',ready);image.removeEventListener('error',fail);};
      if (image.complete && image.naturalWidth > 0) { frame.classList.add('is-cached'); ready(); }
      else { frame.classList.add('is-loading'); image.addEventListener('load', ready, { signal: owner.signal }); }
      image.addEventListener('error', fail, { signal: owner.signal });
      if (image.complete && !image.naturalWidth) fail();
    }
  }
  window.addEventListener('pagehide', event => { if (!event.persisted) owner.abort(); }, { signal: owner.signal });
  window.ReferenceMedia = Object.freeze({ enhance });
  enhance();
})();
