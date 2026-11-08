// Purpose: Connect three contrasting reservation compositions to the same booking contract.
// Pattern: Native form presentation with explicit review and page-owned draft lifecycle.
// Usage: booking-explore.html, booking-schedule.html and booking-quick.html share this controller.
// Related: internal/config/bootstrap/defaults/application-design/assets/examples/morrow/booking-data.js, internal/config/bootstrap/defaults/application-design/assets/examples/booking/booking-studies.css, internal/config/bootstrap/defaults/application-design/assets/examples/shared/interaction.js.
(() => {
  const ui=ExampleUI,model=BookingDemo,options={signal:ui.signal};
  const form=document.getElementById('booking-study'),review=document.getElementById('study-review');
  const commitButton=document.querySelector('[data-study-commit]');
  let ready=false,submitted=false,reviewed=null,shownTotal=null;
  const values=()=>({space:form.elements.space.value,date:form.elements.date.value,start:form.elements.start.value,hours:form.elements.hours.value,people:form.elements.people.value,equipment:form.elements.equipment.checked,name:form.elements.name.value,note:form.elements.note.value});
  function hydrate(d){
    for(const [key,value]of Object.entries(d)){
      const control=form.elements.namedItem(key);if(!control)continue;
      if(control.type==='checkbox')control.checked=value;else control.value=value;
    }
  }
  function announce(text){
    const node=document.querySelector('[data-quote-status]');if(node.textContent!==text)node.textContent=text;
  }
  function render(){
    const d=values(),space=model.spaces[d.space],q=model.quote(d);
    const changed=shownTotal!==null&&shownTotal!==q.total;shownTotal=q.total;
    for(const node of document.querySelectorAll('[data-room-price]'))node.textContent=model.money(model.spaces[node.dataset.roomPrice].price);
    for(const key of ['base','equipment','tax','total'])for(const node of document.querySelectorAll(`[data-price-${key}]`))node.textContent=model.money(q[key]);
    if(changed)ui.motion(document.querySelector('[data-price-total]'),'amount');
    for(const node of document.querySelectorAll('[data-selected-space]'))node.textContent=space.name;
    for(const node of document.querySelectorAll('[data-selected-time]'))node.textContent=d.date+` / ${d.start}:00–${Number(d.start)+Number(d.hours)}:00`;
    document.querySelector('[data-base-label]').textContent=model.money(space.price)+' × '+d.hours+'시간';
    form.elements.people.max=space.people;document.querySelector('[data-capacity]').textContent='최대 '+space.people+'명';
    for(const choice of document.querySelectorAll('[data-space-choice]'))choice.dataset.selected=String(choice.dataset.spaceChoice===d.space);
    for(const button of document.querySelectorAll('[data-time-cell]')){
      const selected=button.dataset.space===d.space&&button.dataset.time===d.start;
      button.disabled=!model.canStart(button.dataset.time,d.hours);
      button.setAttribute('aria-pressed',String(selected));button.querySelector('span').textContent=button.disabled?'선택 불가':selected?'선택 중':'시작 선택';
    }
    for(const button of document.querySelectorAll('[data-date-choice]'))button.setAttribute('aria-pressed',String(button.dataset.dateChoice===d.date));
    const latest=[...model.read().reservations].reverse().find(item=>!item.cancelled);
    const repeat=document.querySelector('[data-reuse-booking]');if(repeat)repeat.disabled=!latest;
    announce(`${space.name} / ${d.hours}시간 / 총 ${model.money(q.total)}`);
  }
  function keepDraft(){if(ready&&!submitted)model.saveDraft(values());}
  function changed(){try{render();keepDraft();}catch(error){ui.error(error.message);}}
  function dateChoices(){
    const host=document.querySelector('[data-date-choices]');if(!host)return;
    const start=new Date(model.today()+'T12:00:00');
    for(let i=0;i<3;i++){
      const date=new Date(start);date.setDate(start.getDate()+i);
      const iso=`${date.getFullYear()}-${String(date.getMonth()+1).padStart(2,'0')}-${String(date.getDate()).padStart(2,'0')}`;
      const button=document.createElement('button');button.type='button';button.dataset.dateChoice=iso;button.setAttribute('aria-pressed','false');
      const day=document.createElement('strong'),label=document.createElement('span');day.textContent=String(date.getDate());label.textContent=date.toLocaleDateString('ko-KR',{month:'short',weekday:'short'});button.append(label,day);host.append(button);
    }
  }
  form.addEventListener('change',changed,options);
  document.addEventListener('click',event=>{
    const button=event.target.closest('button');if(!button)return;
    if(button.dataset.dateChoice){form.elements.date.value=button.dataset.dateChoice;changed();}
    if(button.hasAttribute('data-time-cell')){form.elements.space.value=button.dataset.space;form.elements.start.value=button.dataset.time;changed();}
    if(button.hasAttribute('data-reuse-booking')){
      const latest=[...model.read().reservations].reverse().find(item=>!item.cancelled);if(!latest)return;
      const d=values();hydrate({...d,space:latest.space,start:latest.start,hours:latest.hours,people:latest.people,equipment:latest.equipment});changed();ui.toast('최근 로컬 예약의 이용 조건을 적용했습니다.');
    }
  },options);
  form.addEventListener('submit',event=>{
    event.preventDefault();if(!ready||submitted||review.open)return;
    try{
      reviewed=model.check(values());keepDraft();
      review.querySelector('[data-review-space]').textContent=model.spaces[reviewed.space].name;
      review.querySelector('[data-review-time]').textContent=reviewed.date+` / ${reviewed.start}:00–${Number(reviewed.start)+Number(reviewed.hours)}:00 / ${reviewed.people}명`;
      review.querySelector('[data-review-total]').textContent=model.money(model.quote(reviewed).total);
      review.querySelector('[data-review-error]').hidden=true;commitButton.disabled=false;review.showModal();
    }catch(error){ui.error(error.message);}
  },options);
  review.addEventListener('close',()=>{if(!review.open)reviewed=null;},options);
  commitButton.addEventListener('click',()=>{
    if(submitted||!reviewed)return;
    try{const item=model.book(reviewed);submitted=true;commitButton.disabled=true;location.href='../morrow/atelier-confirmation.html?id='+encodeURIComponent(item.id);}
    catch(error){const node=review.querySelector('[data-review-error]');node.textContent=error.message;node.hidden=false;}
  },options);
  function boot(){
    ready=false;submitted=false;reviewed=null;shownTotal=null;if(review.open)review.close();
    hydrate(model.reload().draft);form.elements.date.min=model.today();render();ready=true;
    form.querySelector('[type=submit]').disabled=false;ui.enter(document.querySelector('main'));
  }
  ui.onLeave(keepDraft);dateChoices();
  try{boot();}catch(error){ui.error(error.message);}
  window.addEventListener('pageshow',event=>{if(event.persisted){try{boot();}catch(error){ui.error(error.message);}}},options);
})();
