// Purpose: Own contextual menus, native dialogs, cancelable presentation motion and bounded feedback.
// Pattern: Page-owned progressive enhancement using delegated input.
// Usage: Practical subpages enhance static and newly rendered controls without framework dependencies.
// Related: internal/config/bootstrap/defaults/application-design/assets/examples/shared/components.css, internal/config/bootstrap/defaults/application-design/assets/examples/shared/motion.css.
(() => {
  const owner=new AbortController(), options={signal:owner.signal};
  const leave=new Set(),timers=new Set(),toasts=new Map(),entered=new WeakSet(),reduced=matchMedia('(prefers-reduced-motion: reduce)');
  const stack=document.querySelector('[data-toasts]');
  let menu=null;
  const motions=new Map();
  function stopMotion(){for(const animation of motions.values())animation.cancel();motions.clear();}
  function motion(region,kind="panel",direction=1){
    if(!region||reduced.matches||!region.animate||(region.contains(document.activeElement)&&kind!=="citation"))return;
    motions.get(region)?.cancel();
    const offset=kind==="slides"?`translateX(${Math.sign(direction||1)*24}px)`:kind==="amount"?"translateY(3px)":"translateY(6px)";
    const frames=kind==="citation"?[{backgroundColor:getComputedStyle(region).getPropertyValue("--accent-soft").trim()},{backgroundColor:"transparent"}]:kind==="save"?[{transform:"scale(.8)"},{transform:"scale(1.12)",offset:.65},{transform:"scale(1)"}]:kind==="filter"?[{opacity:.55,clipPath:"inset(0 0 3% 0)"},{opacity:1,clipPath:"inset(0)"}]:[{opacity:.45,transform:offset},{opacity:1,transform:"none"}];
    const animation=region.animate(frames,{duration:kind==="citation"?650:kind==="slides"?220:kind==="filter"?150:170,easing:"cubic-bezier(.16,1,.3,1)"});
    motions.set(region,animation);
    const release=()=>{if(motions.get(region)===animation)motions.delete(region);};
    animation.onfinish=release;animation.oncancel=release;
  }
  reduced.addEventListener("change",()=>{if(reduced.matches)stopMotion();},options);
  document.addEventListener("visibilitychange",()=>{if(document.hidden)stopMotion();},options);
  function menus(root=document){
    for(const details of root.querySelectorAll('.stagger-menu')){
      const anchor=details.querySelector('summary'),panel=details.querySelector('.menu-panel');
      anchor.setAttribute('aria-haspopup','menu');anchor.setAttribute('aria-expanded',String(details.open));panel.setAttribute('role','menu');
      [...panel.querySelectorAll('a,button')].forEach((item,i)=>{item.setAttribute('role','menuitem');item.style.setProperty('--i',Math.min(i,4));});
    }
  }
  function closeMenu(focus=false){
    if(!menu)return;
    const {details,panel,anchor}=menu;menu=null;
    panel.classList.remove('menu-visible');details.append(panel);details.open=false;anchor.setAttribute('aria-expanded','false');
    if(focus&&anchor.isConnected)anchor.focus({preventScroll:true});
  }
  document.addEventListener('toggle',event=>{
    const details=event.target;
    if(!(details instanceof HTMLDetailsElement)||!details.classList.contains('stagger-menu'))return;
    if(!details.open){if(menu?.details===details)closeMenu(false);return;}
    if(menu?.details===details)return;
    closeMenu(false);menus(details.parentElement);
    const anchor=details.querySelector('summary'),panel=details.querySelector('.menu-panel');menu={details,panel,anchor};
    document.body.append(panel);
    const rect=anchor.getBoundingClientRect(),width=panel.offsetWidth,height=panel.offsetHeight;
    panel.style.left=`${Math.max(10,Math.min(rect.right-width,innerWidth-width-10))}px`;
    panel.style.top=`${rect.bottom+height+8<innerHeight?rect.bottom+8:Math.max(10,rect.top-height-8)}px`;
    panel.classList.add('menu-visible');anchor.setAttribute('aria-expanded','true');panel.querySelector('a,button')?.focus({preventScroll:true});
  },{capture:true,signal:owner.signal});
  document.addEventListener('click',event=>{
    if(menu&&!menu.details.contains(event.target)&&!menu.panel.contains(event.target))closeMenu(false);
    if(menu&&event.target.closest('[role=menuitem]'))closeMenu(!event.target.closest('a[href^="#"]'));
    const opener=event.target.closest('[data-open]');
    if(opener){const dialog=document.getElementById(opener.dataset.open);if(dialog&&!dialog.open)dialog.showModal();}
    const closer=event.target.closest('[data-close-dialog]');if(closer)closer.closest('dialog').close();
  },options);
  document.addEventListener('keydown',event=>{
    if(menu&&event.key==='Tab')closeMenu(true);
    if(menu&&event.key==='Escape'){event.preventDefault();closeMenu(true);return;}
    const summary=event.target.closest('summary');
    if(summary?.parentElement.classList.contains('stagger-menu')&&event.key==='ArrowDown'){
      event.preventDefault();summary.parentElement.open=true;if(menu)menu.panel.querySelector('a,button')?.focus();return;
    }
    if(menu&&menu.panel.contains(event.target)&&!event.altKey&&!event.ctrlKey&&!event.metaKey&&['ArrowDown','ArrowUp','Home','End'].includes(event.key)){
      event.preventDefault();const items=[...menu.panel.querySelectorAll('a,button')].filter(el=>!el.disabled),i=items.indexOf(document.activeElement);
      const next=event.key==='Home'?0:event.key==='End'?items.length-1:(i+(event.key==='ArrowUp'?-1:1)+items.length)%items.length;items[next]?.focus();
    }
    if((event.metaKey||event.ctrlKey)&&event.key.toLowerCase()==='k'){
      const dialog=document.getElementById('command-dialog');if(dialog){event.preventDefault();closeMenu(false);if(!dialog.open)dialog.showModal();}
    }
  },options);
  const command=document.querySelector('[data-command-query]');
  if(command)command.addEventListener('input',()=>{const q=command.value.trim().toLocaleLowerCase();for(const a of document.querySelectorAll('[data-command-link]'))a.hidden=!a.textContent.toLocaleLowerCase().includes(q);},options);
  function enter(root){
    root.setAttribute('aria-busy','false');for(const node of root.querySelectorAll('[data-skeleton]'))node.hidden=true;
    if(reduced.matches||entered.has(root)||!root.closest('[data-app="motion"]'))return;entered.add(root);document.documentElement.classList.add('motion-ready');
    [...root.querySelectorAll('[data-enter]')].slice(0,10).forEach((el,i)=>{el.style.setProperty('--i',Math.min(i,5));el.classList.add('stagger-item');});
  }
  function toast(message,undo){
    if(!stack)return;while(stack.children.length>=3){const stale=[...stack.children].find(el=>!el.contains(document.activeElement));toasts.get(stale||stack.firstElementChild)?.();}
    const node=document.createElement('div');node.className='toast';node.setAttribute('role','status');
    const text=document.createElement('span');text.textContent=message;node.append(text);
    const button=document.createElement('button');button.type='button';button.textContent=undo?'되돌리기':'닫기';
    let timer=null;
    const dispose=()=>{button.removeEventListener('click',act);if(timer!==null){clearTimeout(timer);timers.delete(timer);}toasts.delete(node);node.remove();};
    const act=()=>{try{if(undo)undo();dispose();}catch(cause){error(cause.message);}};
    button.addEventListener('click',act);node.append(button);stack.append(node);toasts.set(node,dispose);
    if(!undo){timer=setTimeout(()=>{timers.delete(timer);if(!node.contains(document.activeElement))dispose();},5000);timers.add(timer);}
  }
  function error(message){
    closeMenu(false);const node=document.querySelector('[data-page-error]');if(node){node.textContent=message;node.hidden=false;}
    for(const element of document.querySelectorAll('[aria-busy=true]'))enter(element);
  }
  async function confirm(title,copy){
    const dialog=document.getElementById('confirm-dialog');if(!dialog||dialog.open)return false;
    dialog.querySelector('h2').textContent=title;dialog.querySelector('p').textContent=copy;dialog.returnValue='';dialog.showModal();
    return new Promise(resolve=>{const abort=()=>resolve(false);owner.signal.addEventListener('abort',abort,{once:true});dialog.addEventListener('close',()=>{owner.signal.removeEventListener('abort',abort);resolve(dialog.returnValue==='confirm');},{once:true,signal:owner.signal});});
  }
  window.addEventListener('resize',()=>closeMenu(!!menu?.panel.contains(document.activeElement)),options);
  document.addEventListener('scroll',()=>closeMenu(!!menu?.panel.contains(document.activeElement)),{capture:true,passive:true,signal:owner.signal});
  document.addEventListener('error',event=>{if(event.target instanceof HTMLScriptElement&&event.target.dataset.required)error('화면 자료를 불러오지 못했습니다. 페이지를 다시 열어주세요.');},{capture:true,signal:owner.signal});
  window.addEventListener('pagehide',event=>{
    for(const callback of leave){try{callback();}catch{/* A failed local write must not claim success. */}}
    stopMotion();closeMenu(false);for(const dispose of toasts.values())dispose();for(const timer of timers)clearTimeout(timer);timers.clear();stack?.replaceChildren();
    if(event.persisted)return;leave.clear();owner.abort();
  },options);
  menus();window.ExampleUI=Object.freeze({signal:owner.signal,motion,enter,toast,error,confirm,menus,onLeave:callback=>leave.add(callback)});
})();
