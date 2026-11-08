// Purpose: Own one bounded local booking contract across different visual compositions.
// Pattern: Domain module over the session-storage boundary.
// Usage: booking.js and booking-views.js share drafts, validation, quotes and confirmed local records.
// Related: internal/config/bootstrap/defaults/application-design/assets/examples/shared/local-state.js, internal/config/bootstrap/defaults/application-design/assets/examples/morrow/booking.js, internal/config/bootstrap/defaults/application-design/assets/examples/booking/booking-views.js.
(() => {
  const spaces=Object.freeze({
    daylight:Object.freeze({name:'Daylight studio',copy:'넓은 테이블과 자연광이 있는 공동 작업실',people:8,price:35000}),
    warm:Object.freeze({name:'Warm meeting room',copy:'대화에 집중할 수 있는 조용한 독립 회의실',people:4,price:22000})
  });
  const starts=Object.freeze(['10','11','12','14','15','16','17','18']);
  const durations=[2,3,4,5,6];
  const fields=['space','date','start','hours','people','name','note'];
  const formatter=new Intl.NumberFormat('ko-KR');
  const dateText=d=>`${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,'0')}-${String(d.getDate()).padStart(2,'0')}`;
  const today=()=>dateText(new Date());
  const initial={version:1,nextId:1001,draft:{space:'daylight',date:today(),start:'10',hours:'2',people:'2',equipment:false,name:'김민서',note:''},reservations:[]};
  let state;
  function validDraft(d){
    return d&&typeof d==='object'&&!Array.isArray(d)&&fields.every(key=>typeof d[key]==='string')&&
      Object.hasOwn(spaces,d.space)&&d.date.length<=10&&d.start.length<=2&&d.hours.length<=1&&d.people.length<=32&&
      typeof d.equipment==='boolean'&&d.name.length<=40&&d.note.length<=500;
  }
  function quote(d){
    if(!d||!Object.hasOwn(spaces,d.space)||!durations.includes(Number(d.hours))||typeof d.equipment!=='boolean')throw new Error('공간과 이용 시간을 확인해주세요.');
    const base=spaces[d.space].price*Number(d.hours),equipment=d.equipment?8000:0,tax=Math.round((base+equipment)*.1);
    return {base,equipment,tax,total:base+equipment+tax};
  }
  function canStart(start,hours){return starts.includes(start)&&durations.includes(Number(hours))&&Number(start)+Number(hours)<=20;}
  function parse(d,allowPast=false){
    if(!validDraft(d))throw new Error('이용 정보의 형식을 확인해주세요.');
    if(!/^\d{4}-\d{2}-\d{2}$/.test(d.date))throw new Error('오늘 이후의 이용 날짜를 선택해주세요.');
    const [year,month,day]=d.date.split('-').map(Number),date=new Date(year,month-1,day);
    if(dateText(date)!==d.date||!allowPast&&d.date<today())throw new Error('오늘 이후의 이용 날짜를 선택해주세요.');
    if(!canStart(d.start,d.hours))throw new Error('이용 시간에 맞는 시작 시간을 선택해주세요.');
    if(!/^\d+(?:\.\d+)?(?:[eE][+-]?\d+)?$/.test(d.people)||!Number.isInteger(Number(d.people))||Number(d.people)<1||Number(d.people)>spaces[d.space].people)throw new Error('이용 시간과 인원을 확인해주세요.');
    if(!d.name.trim())throw new Error('담당자 이름을 확인해주세요.');
    return {...Object.fromEntries(fields.map(key=>[key,d[key]])),people:String(Number(d.people)),name:d.name.trim(),note:d.note.trim(),equipment:d.equipment};
  }
  function valid(s){
    if(!s||!validDraft(s.draft)||!Number.isSafeInteger(s.nextId)||s.nextId<1001||s.nextId>Number.MAX_SAFE_INTEGER-8||!Array.isArray(s.reservations)||s.reservations.length>8)return false;
    const ids=new Set();
    try{
      for(const item of s.reservations){
        parse(item,true);
        if(!/^MR-\d+$/.test(item.id)||Number(item.id.slice(3))>=s.nextId||ids.has(item.id)||typeof item.cancelled!=='boolean')return false;
        ids.add(item.id);const expected=quote(item);
        if(!item.quote||Object.keys(expected).some(key=>item.quote[key]!==expected[key]))return false;
      }
      return true;
    }catch{return false;}
  }
  function reload(){state=ExampleState.read('booking',initial,valid);return state;}
  function read(){return state||reload();}
  function commit(next){ExampleState.write('booking',next);state=next;}
  function saveDraft(d){
    if(!validDraft(d))throw new Error('초안의 공간과 입력 형식을 확인해주세요.');
    commit({...read(),draft:{...Object.fromEntries(fields.map(key=>[key,d[key]])),equipment:d.equipment}});
  }
  function check(d){
    const value=parse(d);
    if(read().reservations.length>=8)throw new Error('이 탭에는 예약 8개까지 보관할 수 있습니다. 새 탭에서 데모를 시작해주세요.');
    return value;
  }
  function book(d){
    const value=check(d),current=read(),id='MR-'+current.nextId;
    const item={...value,id,quote:quote(value),cancelled:false};
    commit({...current,nextId:current.nextId+1,draft:value,reservations:[...current.reservations,item]});return item;
  }
  function cancel(id){
    const current=read();
    if(!current.reservations.some(item=>item.id===id))throw new Error('로컬 예약을 찾을 수 없습니다.');
    commit({...current,reservations:current.reservations.map(item=>item.id===id?{...item,cancelled:true}:item)});
  }
  window.BookingDemo=Object.freeze({spaces,canStart,today,read,reload,quote,saveDraft,check,book,cancel,money:n=>formatter.format(n)+'원'});
})();
