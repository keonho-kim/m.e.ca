// Purpose: Own motion-lab drawer, local loading and explicit presentation previews.
// Pattern: Native page-owned interaction examples.
// Usage: motion-lab.html; manual previews are labeled and never claim server completion.
// Related: internal/config/bootstrap/defaults/ui-design/assets/examples/shared/motion.css, internal/config/bootstrap/defaults/ui-design/assets/examples/motion/motion-source.json, internal/config/bootstrap/defaults/ui-design/assets/examples/motion/lab.css.
(() => {
  const ui=ExampleUI,options={signal:ui.signal};let pending=null;
  const drawer=document.getElementById('side-drawer'),toggle=document.querySelector('[data-side-toggle]');
  toggle.addEventListener('click',()=>{if(!drawer.open)drawer.showModal();toggle.setAttribute('aria-expanded','true');},options);
  document.querySelector('[data-side-close]').addEventListener('click',()=>drawer.close(),options);
  drawer.addEventListener('close',()=>toggle.setAttribute('aria-expanded','false'),options);
  const region=document.querySelector('[data-lab-region]'),list=document.querySelector('[data-lab-list]'),skeleton=document.querySelector('[data-lab-skeleton]'),status=document.querySelector('[data-lab-status]');
  function pendingView(){region.setAttribute('aria-busy','true');skeleton.hidden=false;list.hidden=true;status.textContent='대기 상태 미리보기 · 아직 자료를 요청하지 않았습니다.';}
  function stop(){pending?.abort();pending=null;region.setAttribute('aria-busy','false');skeleton.hidden=true;list.hidden=false;}
  document.querySelector('[data-preview-pending]').addEventListener('click',()=>{stop();pendingView();},options);
  document.querySelector('[data-skeleton-kind]').addEventListener('change',event=>{for(const node of skeleton.querySelectorAll('.skeleton')){node.classList.toggle('skeleton-pulse',event.target.value==='pulse');node.classList.toggle('skeleton-shimmer',event.target.value==='shimmer');}},options);
  document.querySelector('[data-load-local]').addEventListener('click',async()=>{
    stop();const task=new AbortController();pending=task;pendingView();status.textContent='로컬 자료를 읽고 있습니다.';
    try{
      const response=await fetch('motion-source.json',{signal:task.signal,cache:'no-store'});if(!response.ok)throw new Error('자료를 불러오지 못했습니다.');const rows=await response.json();
      if(task.signal.aborted||!region.isConnected)return;
      list.replaceChildren(...rows.map(row=>{const article=document.createElement('article'),title=document.createElement('strong'),copy=document.createElement('p');article.dataset.enter='';title.textContent=row.title;copy.textContent=row.detail;copy.className='small muted';copy.style.marginTop='7px';article.append(title,copy);return article;}));
      list.hidden=false;skeleton.hidden=true;ui.enter(region);status.textContent=`로컬 자료 ${rows.length}개를 읽었습니다. 빠른 응답은 지연시키지 않습니다.`;
    }catch(error){if(error.name!=='AbortError'){stop();status.textContent=error.message;}}
    finally{if(pending===task)pending=null;}
  },options);
  document.querySelector('[data-cancel-local]').addEventListener('click',()=>{stop();status.textContent='요청 또는 미리보기를 종료했습니다. 기존 내용은 유지합니다.';},options);
  document.querySelector('[data-expand-all]').addEventListener('click',event=>{const button=event.currentTarget,open=button.getAttribute('aria-expanded')!=='true';for(const item of document.querySelectorAll('[data-lab-accordion]'))item.open=open;button.setAttribute('aria-expanded',String(open));button.textContent=open?'모두 접기':'모두 펼치기';},options);
  let position=0;
  const screens=[['조건 입력','중요한 입력과 도움말을 같은 맥락에서 확인합니다.'],['내용 검토','바뀌는 내용과 영향을 확인한 뒤 결정합니다.'],['결과 확인','실제 결과와 다음 행동을 확인합니다. 이 패널은 화면 전환 미리보기입니다.']];
  document.querySelector('[data-next-screen]').addEventListener('click',()=>{
    position=(position+1)%screens.length;const old=document.querySelector('[data-lab-screen]'),next=old.cloneNode(true);next.dataset.transition=document.querySelector('[data-transition-kind]').value;next.querySelector('h3').textContent=screens[position][0];next.querySelector('p').textContent=screens[position][1];old.replaceWith(next);
    for(const [i,item]of [...document.querySelectorAll('[data-wizard-stage]')].entries()){item.setAttribute('aria-current',i===position?'step':'false');item.dataset.state=i<position?'done':i===position?'current':'queued';}
  },options);
  ui.onLeave(()=>{stop();if(drawer.open)drawer.close();});
})();
