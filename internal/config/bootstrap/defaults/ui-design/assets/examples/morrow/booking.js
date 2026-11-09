// Purpose: Present the original Morrow space, booking and confirmation journey.
// Pattern: Native page controller over the shared booking domain.
// Usage: Morrow space, booking and confirmation pages; no reservation or payment service is contacted.
// Related: internal/config/bootstrap/defaults/ui-design/assets/examples/morrow/atelier.css, internal/config/bootstrap/defaults/ui-design/assets/examples/morrow/booking-data.js, internal/config/bootstrap/defaults/ui-design/assets/examples/shared/media.js.
(() => {
  const ui=ExampleUI,options={signal:ui.signal},page=document.body.dataset.page;
  const model=BookingDemo,{spaces,today,money,quote}=model;
  let state,submitted=false,shownTotal=null;
  function fillPhoto(root,key){
    const frame=root.querySelector('.media');frame.style.viewTransitionName='morrow-room-'+key;
    let img=frame.querySelector('img.full'),preview=frame.querySelector('img.preview');
    if(!preview){
      preview=document.createElement('img');preview.className='preview';preview.alt='';
      preview.setAttribute('aria-hidden','true');preview.width=40;preview.height=30;frame.prepend(preview);
    }
    if(!img){
      img=document.createElement('img');img.className='full';img.dataset.blurUp='';
      img.width=1448;img.height=1086;img.decoding='async';preview.after(img);
    }
    img.src='../shared/studio-'+key+'.jpg';img.alt=spaces[key].copy;
    preview.src='../shared/'+key+'-preview.jpg';ReferenceMedia.enhance(root);
  }
  function renderSpace(){
    const key=new URLSearchParams(location.search).get('space')||'daylight',space=spaces[key];
    if(!space){ui.error('공간을 찾을 수 없습니다. 공간 목록에서 다시 선택해주세요.');document.querySelector('[data-space-content]').hidden=true;return;}
    document.querySelector('[data-space-name]').textContent=space.name;document.querySelector('[data-space-copy]').textContent=space.copy;
    document.querySelector('[data-space-people]').textContent='최대 '+space.people+'명';document.querySelector('[data-space-price]').textContent=money(space.price);
    document.querySelector('[data-space-book]').href='atelier-booking.html?space='+key;fillPhoto(document.querySelector('[data-space-photo]'),key);ui.enter(document.querySelector('main'));
  }
  const form=document.getElementById('booking-form');
  function values(){return {space:form.elements.space.value,date:form.elements.date.value,start:form.elements.start.value,hours:form.elements.hours.value,people:form.elements.people.value,equipment:form.elements.equipment.checked,name:form.elements.name.value.trim(),note:form.elements.note.value.trim()};}
  function draft(){if(form&&!submitted)model.saveDraft(values());}
  function renderQuote(){
    const d=values(),q=quote(d),space=spaces[d.space];
    const changed=shownTotal!==null&&shownTotal!==q.total;shownTotal=q.total;
    document.querySelector('[data-summary-name]').textContent=space.name;
    document.querySelector('[data-summary-time]').textContent=d.date+` / ${d.start}:00~${Number(d.start)+Number(d.hours)}:00`;
    for(const key of ['base','equipment','tax','total'])document.querySelector('[data-price-'+key+']').textContent=money(q[key]);
    document.querySelector('[data-base-label]').textContent=money(space.price)+' × '+d.hours+'시간';
    if(changed)ui.motion(document.querySelector('[data-price-total]'),'amount');
    form.elements.people.max=space.people;document.querySelector('[data-capacity]').textContent='최대 '+space.people+'명';
    for(const button of document.querySelectorAll('[data-slot]')){button.disabled=!model.canStart(button.dataset.slot,d.hours);button.setAttribute('aria-pressed',String(button.dataset.slot===d.start));}
    const frame=document.querySelector('[data-booking-photo]');
    if(frame.dataset.space!==d.space){frame.dataset.space=d.space;const full=frame.querySelector('img.full');if(full?.getAttribute('src')){const next=full.cloneNode(false);full.replaceWith(next);}fillPhoto(frame,d.space);}
  }
  function showReview(){
    try{const d=model.check(values());draft();const q=quote(d);document.querySelector('[data-review-space]').textContent=spaces[d.space].name;document.querySelector('[data-review-time]').textContent=d.date+` / ${d.start}:00~${Number(d.start)+Number(d.hours)}:00 / ${d.people}명`;document.querySelector('[data-review-total]').textContent=money(q.total);document.getElementById('booking-review').showModal();}catch(error){ui.error(error.message);}
  }
  function renderConfirmation(){
    const id=new URLSearchParams(location.search).get('id'),item=state.reservations.find(r=>r.id===id);
    if(!item){document.querySelector('[data-confirmation]').hidden=true;document.querySelector('[data-booking-missing]').hidden=false;return;}
    document.querySelector('[data-confirm-title]').textContent=item.cancelled?'로컬 예약을 취소했습니다':'예약 내용을 보관했습니다';
    document.querySelector('[data-booking-id]').textContent=item.id;document.querySelector('[data-confirm-space]').textContent=spaces[item.space].name;
    document.querySelector('[data-confirm-time]').textContent=item.date+` / ${item.start}:00~${Number(item.start)+Number(item.hours)}:00`;
    document.querySelector('[data-confirm-people]').textContent=item.people+'명 / '+item.name;document.querySelector('[data-confirm-total]').textContent=money(item.quote.total);
    document.querySelector('[data-cancel-booking]').hidden=item.cancelled;ui.enter(document.querySelector('main'));
  }
  if(form){
    form.addEventListener('change',()=>{try{renderQuote();draft();}catch(error){ui.error(error.message);}},options);
    form.addEventListener('submit',event=>{event.preventDefault();showReview();},options);ui.onLeave(draft);
    document.querySelector('[data-slots]').addEventListener('click',event=>{const button=event.target.closest('[data-slot]');if(button){form.elements.start.value=button.dataset.slot;try{renderQuote();draft();}catch(error){ui.error(error.message);}}},options);
    document.querySelector('[data-booking-commit]').addEventListener('click',()=>{
      if(submitted)return;
      try{const item=model.book(values()),id=item.id;submitted=true;location.href='atelier-confirmation.html?id='+id;}catch(error){document.getElementById('booking-review').close();ui.error(error.message);}
    },options);
  }
  document.querySelector('[data-cancel-booking]')?.addEventListener('click',async()=>{
    if(!await ui.confirm('로컬 예약을 취소할까요?','실제 예약 서비스에는 연결되지 않습니다. 이 탭에 보관한 데모 예약만 취소합니다.'))return;
    try{const id=new URLSearchParams(location.search).get('id');model.cancel(id);state=model.read();renderConfirmation();ui.toast('로컬 예약을 취소했습니다.');}catch(error){ui.error(error.message);}
  },options);
  function boot(){
    state=model.reload();
    if(page==='space')renderSpace();
    if(form){const key=new URLSearchParams(location.search).get('space');if(key&&!spaces[key])throw new Error('선택한 공간을 찾을 수 없습니다.');const d={...state.draft,...(key?{space:key}:{})};for(const [key,value]of Object.entries(d)){if(form.elements[key].type==='checkbox')form.elements[key].checked=value;else form.elements[key].value=value;}form.elements.date.min=today();renderQuote();ui.enter(document.querySelector('main'));}
    if(page==='confirmation')renderConfirmation();if(page==='home')ui.enter(document.querySelector('main'));
  }
  try{boot();}catch(error){ui.error(error.message);}
  window.addEventListener('pageshow',event=>{if(event.persisted){try{boot();}catch(error){ui.error(error.message);}}},options);
})();
