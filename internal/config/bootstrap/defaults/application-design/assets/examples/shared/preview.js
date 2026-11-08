// Purpose: Own native tab selection, scripts-off panels and progressive deep links.
// Pattern: Progressive enhancement with page-owned event cleanup.
// Usage: Only screens with real tab groups load this after interaction.js; without scripts all panels remain readable.
// Related: internal/config/bootstrap/defaults/application-design/assets/examples/shared/interaction.js.
if (window.ExampleUI) {
  const ui=window.ExampleUI,options={signal:ui.signal};
  for(const tabs of document.querySelectorAll('[data-tabs]')){
    const buttons=[...tabs.querySelectorAll('[role=tab]')];
    let current=buttons.find(tab=>tab.getAttribute('aria-selected')==='true')||buttons[0];
    const activate=(selected,animate)=>{
      const previous=current;current=selected;
      for(const tab of buttons){
        const active=tab===selected,panel=document.getElementById(tab.getAttribute('aria-controls'));
        tab.setAttribute('aria-selected',String(active));tab.tabIndex=active?0:-1;panel.hidden=!active;
        if(!active)for(const animation of panel.getAnimations?.()||[])animation.cancel();
      }
      if(animate&&previous!==selected)ui.motion(document.getElementById(selected.getAttribute('aria-controls')),tabs.dataset.panelMotion||'panel',buttons.indexOf(selected)-buttons.indexOf(previous));
    };
    for(const button of buttons){
      button.addEventListener('click',event=>activate(button,event.isTrusted),options);
      button.addEventListener('keydown',event=>{
        if(event.altKey||event.ctrlKey||event.metaKey||!['ArrowLeft','ArrowRight','Home','End'].includes(event.key))return;
        event.preventDefault();const i=buttons.indexOf(button),next=event.key==='Home'?0:event.key==='End'?buttons.length-1:(i+(event.key==='ArrowLeft'?-1:1)+buttons.length)%buttons.length;
        activate(buttons[next],true);buttons[next].focus();
      },options);
    }
  }
  for (const group of document.querySelectorAll('[data-tabs]')) {
    group.querySelector('[role=tab][aria-selected=true]')?.click();
  }
  const revealFragment = () => {
    let id;
    try { id = decodeURIComponent(location.hash.slice(1)); }
    catch { return; }
    const target = document.getElementById(id);
    const panel = target?.closest('[role=tabpanel]');
    if (!panel) return;
    for (const tab of document.querySelectorAll('[data-tabs] [role=tab]')) {
      if (tab.getAttribute('aria-controls') === panel.id) {
        tab.click();
        target.scrollIntoView({block:'nearest'});
        break;
      }
    }
  };
  window.addEventListener('hashchange', revealFragment, {signal:window.ExampleUI.signal});
  revealFragment();
}
