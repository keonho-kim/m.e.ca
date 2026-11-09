// Purpose: Keep document headings and the slide filmstrip navigable within their workspaces.
// Pattern: Page-owned view interaction with native narrow-screen fallbacks.
// Usage: Document and slides load this after shared/interaction.js.
// Related: internal/config/bootstrap/defaults/ui-design/assets/examples/office/office-studio.css, internal/config/bootstrap/defaults/ui-design/assets/examples/shared/interaction.js.
(() => {
  if (!window.ExampleUI) return;
  const options = {signal:window.ExampleUI.signal};
  const stage = document.querySelector('.office-paper-stage');
  const outlines = document.querySelectorAll('.office-outline');
  function reveal(target) {
    if (!stage || !target || !stage.contains(target)) return false;
    for (const outline of outlines) {
      for (const link of outline.querySelectorAll('a')) {
        if (link.getAttribute('href') === '#' + target.id) link.setAttribute('aria-current','location');
        else link.removeAttribute('aria-current');
      }
    }
    if (getComputedStyle(stage).overflowY !== 'auto') return false;
    stage.scrollTo({top:stage.scrollTop + target.getBoundingClientRect().top - stage.getBoundingClientRect().top - 24,behavior:matchMedia('(prefers-reduced-motion: reduce)').matches?'auto':'smooth'});
    target.focus({preventScroll:true});
    return true;
  }
  for (const outline of outlines) {
    outline.addEventListener('click', event => {
      const link = event.target.closest('a[href^="#"]');
      if (!link || event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return;
      if (reveal(document.getElementById(link.getAttribute('href').slice(1)))) {
        event.preventDefault();
        history.replaceState(null,'',link.getAttribute('href'));
        window.scrollTo(0,0);
      }
    }, options);
  }
  window.addEventListener('load', () => {
    let id;
    try { id = decodeURIComponent(location.hash.slice(1)); }
    catch { return; }
    if (id && reveal(document.getElementById(id))) window.scrollTo(0,0);
  }, {...options,once:true});
  const filmstrip = document.querySelector('.slide-thumbnails');
  if (!filmstrip) return;
  const vertical = matchMedia('(min-width:601px)');
  const orient = () => filmstrip.setAttribute('aria-orientation',vertical.matches?'vertical':'horizontal');
  orient();
  vertical.addEventListener('change',orient,options);
  filmstrip.addEventListener('keydown', event => {
    if (!vertical.matches || event.altKey || event.ctrlKey || event.metaKey || !['ArrowUp','ArrowDown'].includes(event.key)) return;
    const tabs = [...filmstrip.querySelectorAll('[role=tab]')];
    const current = tabs.indexOf(event.target);
    if (current < 0) return;
    event.preventDefault();
    const next = tabs[(current + (event.key === 'ArrowUp'?-1:1) + tabs.length) % tabs.length];
    next.click();
    next.focus();
  },options);
})();
