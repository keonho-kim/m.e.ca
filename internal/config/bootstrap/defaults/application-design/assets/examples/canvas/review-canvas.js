// Purpose: Own bounded spatial review, selection, zoom and accessible local positioning.
// Pattern: Page-owned interaction state with pointer capture and revision-guarded undo.
// Usage: review-canvas.html offers pointer, keyboard and named-button alternatives without persistence.
// Related: internal/config/bootstrap/defaults/application-design/assets/examples/canvas/review-canvas.css, internal/config/bootstrap/defaults/application-design/assets/examples/shared/interaction.js.
(() => {
  const ui=ExampleUI,options={signal:ui.signal},plane=document.querySelector('[data-canvas-plane]');
  const viewport=document.querySelector('[data-canvas-viewport]'),extent=document.querySelector('[data-canvas-extent]');
  const cards=[...plane.querySelectorAll('[data-artifact]')],positions=new Map(),initial=new Map();
  let selected=cards[0].dataset.artifact,scale=1,drag=null,revision=0;
  const status=document.querySelector('[data-canvas-status]');
  for(const card of cards){const pos={x:parseFloat(card.style.getPropertyValue('--x')),y:parseFloat(card.style.getPropertyValue('--y'))};positions.set(card.dataset.artifact,pos);initial.set(card.dataset.artifact,{...pos});}
  function node(id){return cards.find(card=>card.dataset.artifact===id);}
  function inspect(){
    const card=node(selected),pos=positions.get(selected);
    document.querySelector('[data-inspector-title]').textContent=card.querySelector('[data-select]').textContent.trim();
    document.querySelector('[data-inspector-copy]').textContent=card.dataset.description;
    document.querySelector('[data-inspector-type]').textContent=card.dataset.kind;
    document.querySelector('[data-inspector-x]').textContent=Math.round(pos.x)+' px';document.querySelector('[data-inspector-y]').textContent=Math.round(pos.y)+' px';
    for(const item of cards){const active=item.dataset.artifact===selected;item.dataset.selected=String(active);item.querySelector('[data-select]').setAttribute('aria-pressed',String(active));}
  }
  function announce(text){if(status.textContent!==text)status.textContent=text;}
  function select(id){const changed=selected!==id;selected=id;inspect();if(changed)ui.motion(document.querySelector('.canvas-inspector'),'panel');}
  function bounds(id){const card=node(id);return {x:Math.max(0,1280-card.offsetWidth),y:Math.max(0,760-card.offsetHeight)};}
  function move(id,x,y,limit=bounds(id)){
    const pos={x:Math.max(0,Math.min(limit.x,x)),y:Math.max(0,Math.min(limit.y,y))};
    positions.set(id,pos);const card=node(id);card.style.setProperty('--x',pos.x+'px');card.style.setProperty('--y',pos.y+'px');revision++;inspect();
  }
  function stopDrag(cancel=false){
    if(!drag)return;const current=drag;drag=null;node(current.id).classList.remove('is-dragging');
    if(cancel)move(current.id,current.start.x,current.start.y,current.limit);
    if(current.handle.hasPointerCapture(current.pointer))current.handle.releasePointerCapture(current.pointer);
    const pos=positions.get(current.id);announce(`자료 위치 ${Math.round(pos.x)}, ${Math.round(pos.y)}. ${cancel?'이동을 취소했습니다.':'이 페이지에서만 유지됩니다.'}`);
  }
  function zoom(next){
    stopDrag();scale=Math.max(.5,Math.min(1.5,next));plane.style.transform=`scale(${scale})`;
    extent.style.width=1280*scale+'px';extent.style.height=760*scale+'px';
    document.querySelector('[data-zoom]').value=Math.round(scale*100);document.querySelector('[data-zoom-value]').textContent=Math.round(scale*100)+'%';
  }
  document.addEventListener('pointerdown',event=>{
    const handle=event.target.closest('[data-drag]');if(!handle||event.button!==0||drag)return;
    const id=handle.closest('[data-artifact]').dataset.artifact;select(id);event.preventDefault();handle.focus({preventScroll:true});
    drag={id,handle,pointer:event.pointerId,x:event.clientX,y:event.clientY,start:{...positions.get(id)},limit:bounds(id)};
    handle.setPointerCapture(event.pointerId);node(id).classList.add('is-dragging');
  },options);
  document.addEventListener('pointermove',event=>{if(drag&&event.pointerId===drag.pointer)move(drag.id,drag.start.x+(event.clientX-drag.x)/scale,drag.start.y+(event.clientY-drag.y)/scale,drag.limit);},options);
  document.addEventListener('pointerup',event=>{if(drag&&event.pointerId===drag.pointer)stopDrag();},options);
  document.addEventListener('pointercancel',event=>{if(drag&&event.pointerId===drag.pointer)stopDrag(true);},options);
  document.addEventListener('lostpointercapture',event=>{if(drag&&event.pointerId===drag.pointer)stopDrag(true);},options);
  document.addEventListener('keydown',event=>{
    if(event.key==='Escape'&&drag){event.preventDefault();stopDrag(true);return;}
    const card=event.target.closest('[data-artifact]');if(!card||!event.altKey||!['ArrowLeft','ArrowRight','ArrowUp','ArrowDown'].includes(event.key))return;
    event.preventDefault();stopDrag();select(card.dataset.artifact);const pos=positions.get(selected);
    move(selected,pos.x+(event.key==='ArrowLeft'?-10:event.key==='ArrowRight'?10:0),pos.y+(event.key==='ArrowUp'?-10:event.key==='ArrowDown'?10:0));
    announce('선택한 자료를 10px 이동했습니다.');
  },options);
  document.addEventListener('click',event=>{
    const card=event.target.closest('[data-artifact]');if(card)select(card.dataset.artifact);
    const button=event.target.closest('button');if(!button)return;
    if(button.dataset.nudge){stopDrag();const pos=positions.get(selected),[x,y]=button.dataset.nudge.split(',').map(Number);move(selected,pos.x+x,pos.y+y);announce('선택한 자료를 10px 이동했습니다.');}
    if(button.hasAttribute('data-fit')){zoom(Math.floor(Math.min(viewport.clientWidth/1280,viewport.clientHeight/760,1)*10)/10);viewport.scrollTo({top:0,left:0,behavior:'instant'});}
    if(button.hasAttribute('data-reset-layout')){
      stopDrag();const before=new Map([...positions].map(([id,pos])=>[id,{...pos}]));
      for(const [id,pos]of initial)move(id,pos.x,pos.y);
      const resetRevision=revision;announce('기본 배치로 돌아왔습니다.');
      ui.toast('이 페이지의 기본 배치로 돌아왔습니다.',()=>{if(revision!==resetRevision)throw new Error('이후 배치가 변경되어 이전 배치를 덮어쓰지 않았습니다.');for(const [id,pos]of before)move(id,pos.x,pos.y);announce('초기화 전 배치를 복원했습니다.');});
    }
  },options);
  document.querySelector('[data-zoom]').addEventListener('input',event=>zoom(Number(event.target.value)/100),options);
  window.addEventListener('resize',()=>stopDrag(true),options);ui.onLeave(()=>stopDrag(true));
  for(const image of plane.querySelectorAll('img'))image.draggable=false;
  document.querySelector('[data-artifact-count]').textContent=cards.length+'개 자료';inspect();zoom(1);
})();
