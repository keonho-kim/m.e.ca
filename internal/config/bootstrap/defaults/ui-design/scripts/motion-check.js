// Purpose: Observe bounded on-screen geometry and report possible abrupt motion.
// Pattern: Playwright CLI function expression with page-owned recording state.
// Usage: Start with run-code --filename; stop through the recorded stop function.
// Related: internal/config/bootstrap/defaults/ui-design/references/browser-tools.md.
async page => page.evaluate(() => {
  const key = '__mecaUIDesignMotion';
  if (window[key]) return { status: 'inconclusive', reason: 'a motion recording already exists' };
  const maxDuration = 15000;
  const maxFrames = 900;
  const maxVisible = 120;
  const maxScanned = 3000;
  const maxFindings = 20;
  const started = performance.now();
  const identifiers = new WeakMap();
  let nextIdentifier = 1;
  let samples = [];
  let raf;
  let expiry;
  let limited = false;
  let iframeSeen = false;
  let finished;
  const state = { frames: 0, stop };
  window[key] = state;

  function label(element) {
    const tag = element.tagName.toLowerCase();
    return element.id ? `${tag}#${CSS.escape(element.id)}` : tag;
  }
  function id(element) {
    if (!identifiers.has(element)) identifiers.set(element, nextIdentifier++);
    return identifiers.get(element);
  }
  function visible(element) {
    const rect = element.getBoundingClientRect();
    if (rect.width < 1 || rect.height < 1 || rect.bottom <= 0 || rect.right <= 0 ||
        rect.top >= innerHeight || rect.left >= innerWidth) return false;
    for (let parent = element; parent; parent = parent.parentElement) {
      const style = getComputedStyle(parent);
      if (style.display === 'none' || style.visibility === 'hidden' ||
          style.visibility === 'collapse' || Number(style.opacity) === 0) return false;
    }
    return true;
  }
  function sample(time) {
    if (window[key] !== state || finished) return;
    const boxes = new Map();
    let scanned = 0;
    const walker = document.createTreeWalker(document.body || document.documentElement, NodeFilter.SHOW_ELEMENT);
    for (let element = walker.currentNode; element; (element = walker.nextNode())) {
      if (++scanned > maxScanned) { limited = true; break; }
      if (element.tagName === 'IFRAME' && !iframeSeen && visible(element)) iframeSeen = true;
      if (!element.matches('button, a[href], input, select, textarea, [role], [id], [class]')) continue;
      if (!visible(element)) continue;
      if (boxes.size >= maxVisible) { limited = true; break; }
      const rect = element.getBoundingClientRect();
      boxes.set(id(element), { target: label(element), x: rect.x, y: rect.y, width: rect.width, height: rect.height });
    }
    samples.push({ time, boxes, scrollX, scrollY, width: innerWidth, height: innerHeight });
    state.frames = samples.length;
    if (samples.length >= maxFrames) finish('frame limit reached');
    else raf = requestAnimationFrame(sample);
  }
  function inspect() {
    const findings = [];
    const reported = new Set();
    const intervals = samples.slice(1).map((frame, index) => frame.time - samples[index].time);
    const ordered = intervals.filter(value => value > 0).sort((a, b) => a - b);
    const typical = ordered[Math.floor(ordered.length / 2)] || 0;
    const maxGap = Math.max(60, typical * 2.5);
    const gaps = intervals.filter(value => value <= 0 || value > maxGap).length;
    const viewportChanged = samples.some(frame => frame.width !== samples[0].width || frame.height !== samples[0].height);
    const scrolled = samples.some(frame => frame.scrollX !== samples[0].scrollX || frame.scrollY !== samples[0].scrollY);
    let moved = false;
    for (let index = 2; index < samples.length - 1; index++) {
      const before = samples[index - 2];
      const previous = samples[index - 1];
      const current = samples[index];
      const after = samples[index + 1];
      if (current.time - previous.time > maxGap || previous.time - before.time > maxGap ||
          after.time - current.time > maxGap || current.scrollX !== previous.scrollX ||
          current.scrollY !== previous.scrollY || current.width !== previous.width ||
          current.height !== previous.height) continue;
      for (const [node, box] of current.boxes) {
        const a = before.boxes.get(node);
        const b = previous.boxes.get(node);
        const c = after.boxes.get(node);
        if (!a || !b || !c) continue;
        for (const axis of ['x', 'y', 'width', 'height']) {
          const delta = box[axis] - b[axis];
          if (Math.abs(delta) > 0.5) moved = true;
          const nearby = Math.max(Math.abs(b[axis] - a[axis]), Math.abs(c[axis] - box[axis]), 0.5);
          const findingKey = `${node}:${axis}`;
          if (Math.abs(delta) > 2 && Math.abs(delta) > nearby * 3 && !reported.has(findingKey)) {
            reported.add(findingKey);
            if (findings.length < maxFindings) {
              findings.push({ kind: 'jump', target: box.target, axis, delta: Math.round(delta * 100) / 100,
                atMs: Math.round(current.time - started) });
            }
          }
        }
      }
    }
    const uninspected = [];
    if (samples.length < 4) uninspected.push('fewer than four animation frames were recorded');
    if (!samples.some(frame => frame.boxes.size)) uninspected.push('no visible elements were sampled');
    if (!moved) uninspected.push('no sampled geometry changed');
    if (gaps) uninspected.push(`${gaps} sampling gap(s) could hide motion`);
    if (typical > 40) uninspected.push('animation frames were sampled too sparsely');
    if (scrolled || viewportChanged) uninspected.push('scrolling or viewport changes were not judged as motion');
    if (iframeSeen) uninspected.push('visible iframes require separate inspection');
    if (limited) uninspected.push('visible-element or scan limit was reached');
    if (reported.size > maxFindings) uninspected.push('additional findings were omitted');
    return { findings, uninspected };
  }
  function finish(reason) {
    if (finished) return finished;
    cancelAnimationFrame(raf);
    clearTimeout(deadline);
    const result = inspect();
    if (reason) result.uninspected.push(reason);
    finished = {
      status: result.uninspected.length ? 'inconclusive' : result.findings.length ? 'issues' : 'no-issues-detected',
      frames: samples.length,
      durationMs: samples.length > 1 ? Math.round(samples.at(-1).time - samples[0].time) : 0,
      findings: result.findings,
      uninspected: result.uninspected,
    };
    samples = [];
    if (reason) expiry = setTimeout(() => { if (window[key] === state) delete window[key]; }, maxDuration);
    return finished;
  }
  function stop() {
    const result = finish();
    clearTimeout(expiry);
    if (window[key] === state) delete window[key];
    return result;
  }
  const deadline = setTimeout(() => finish('recording time limit reached'), maxDuration);
  raf = requestAnimationFrame(sample);
  return { status: 'recording', maxDurationMs: maxDuration, maxFrames, maxVisible };
})
