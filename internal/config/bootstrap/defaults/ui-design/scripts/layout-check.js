// Purpose: Inspect visible button content for constrained layout geometry.
// Pattern: Playwright CLI function expression with browser-side measurements.
// Usage: Run in an existing session with playwright-cli run-code --filename.
// Related: internal/config/bootstrap/defaults/ui-design/references/browser-tools.md.
async page => page.evaluate(() => {
  const findings = [];
  const uninspected = [];
  const limit = 200;
  const scanLimit = 3000;
  const findingLimit = 20;
  const tolerance = 1;
  let omitted = 0;
  const add = finding => {
    if (findings.length < findingLimit) findings.push(finding);
    else omitted++;
  };
  const number = value => Number.parseFloat(value) || 0;
  const rounded = value => Math.round(value * 100) / 100;
  const name = element => {
    const tag = element.tagName.toLowerCase();
    return element.id ? `${tag}#${CSS.escape(element.id)}` : tag;
  };
  const visible = element => {
    const rect = element.getBoundingClientRect();
    if (rect.width < 1 || rect.height < 1 || rect.bottom <= 0 || rect.right <= 0 ||
        rect.top >= innerHeight || rect.left >= innerWidth) return false;
    for (let parent = element; parent; parent = parent.parentElement) {
      const style = getComputedStyle(parent);
      if (style.display === 'none' || style.visibility === 'hidden' ||
          style.visibility === 'collapse' || Number(style.opacity) === 0) return false;
    }
    return true;
  };
  const animations = document.getAnimations().filter(animation => animation.playState === 'running');
  const moving = element => animations.some(animation => {
    const target = animation.effect?.target;
    if (!(target instanceof Element) || !(target === element || target.contains(element))) return false;
    const geometry = /^(transform|translate|rotate|scale|width|height|top|right|bottom|left|padding.*|margin.*)$/;
    return animation.effect.getKeyframes().some(frame => Object.keys(frame).some(key => geometry.test(key)));
  });
  let checked = 0;
  let skipped = 0;
  let limited = false;
  let iframeSeen = false;
  let scanned = 0;
  const walker = document.createTreeWalker(document.body || document.documentElement, NodeFilter.SHOW_ELEMENT);
  for (let element = walker.currentNode; element; (element = walker.nextNode())) {
    if (++scanned > scanLimit) { limited = true; break; }
    if (element.tagName === 'IFRAME' && visible(element)) iframeSeen = true;
    if (!element.matches('button, [role="button"]')) continue;
    if (!visible(element)) continue;
    if (checked + skipped >= limit) { limited = true; break; }
    if (moving(element)) { skipped++; continue; }
    const style = getComputedStyle(element);
    if (style.writingMode !== 'horizontal-tb') { skipped++; continue; }
    const padding = number(style.paddingTop) + number(style.paddingBottom);
    const border = number(style.borderTopWidth) + number(style.borderBottomWidth);
    const cssHeight = number(style.height) + (style.boxSizing === 'border-box' ? 0 : padding + border);
    const rect = element.getBoundingClientRect();
    if (!cssHeight || !rect.height) { skipped++; continue; }
    const scale = rect.height / cssHeight;
    let top = Infinity;
    let bottom = -Infinity;
    let textNodes = 0;
    let textLength = 0;
    let lines = 0;
    let textLimited = false;
    const textWalker = document.createTreeWalker(element, NodeFilter.SHOW_TEXT);
    for (let text; (text = textWalker.nextNode());) {
      textLength += text.length;
      if (++textNodes > 128 || textLength > 16384) { textLimited = true; break; }
      if (!text.textContent.trim() || !visible(text.parentElement)) continue;
      const lineHeight = number(getComputedStyle(text.parentElement).lineHeight);
      const range = document.createRange();
      range.selectNodeContents(text);
      for (const line of range.getClientRects()) {
        if (++lines > 256) { textLimited = true; break; }
        if (!line.width || !line.height) continue;
        const center = (line.top + line.bottom) / (2 * scale);
        const height = Math.max(line.height / scale, lineHeight);
        top = Math.min(top, center - height / 2);
        bottom = Math.max(bottom, center + height / 2);
      }
      if (textLimited) break;
    }
    if (textLimited || !Number.isFinite(top)) { skipped++; continue; }
    checked++;
    const needed = bottom - top + padding + border;
    if (needed > cssHeight + tolerance) {
      add({ kind: 'control-fit', target: name(element), actual: rounded(cssHeight), required: rounded(needed) });
    }
    if (!['auto', 'scroll'].includes(style.overflowX) && element.scrollWidth > element.clientWidth + tolerance) {
      add({ kind: 'horizontal-overflow', target: name(element), actual: element.clientWidth, required: element.scrollWidth });
    }
  }
  if (skipped) uninspected.push(`${skipped} visible button(s) were moving or had unsupported content or writing mode`);
  if (limited) uninspected.push(`scan stopped at ${limit} visible controls or ${scanLimit} elements`);
  if (omitted) uninspected.push(`${omitted} additional finding(s) were omitted`);
  if (iframeSeen) uninspected.push('visible iframes require separate inspection');
  if (!checked) uninspected.push('no supported visible buttons were checked');
  return {
    status: uninspected.length ? 'inconclusive' : findings.length ? 'issues' : 'no-issues-detected',
    checked,
    findings,
    uninspected,
  };
})
