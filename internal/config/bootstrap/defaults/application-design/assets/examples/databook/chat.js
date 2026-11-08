// Purpose: Link each citation to the right evidence record without losing the current answer.
// Pattern: Page-owned evidence navigation with native scrolling and focus.
// Usage: databook/chat.html loads this after shared/interaction.js.
// Related: internal/config/bootstrap/defaults/application-design/assets/examples/databook/databook.css.
(() => {
  const region=document.querySelector('.db-evidence');
  const options={signal:ExampleUI.signal};
  const reduce=matchMedia('(prefers-reduced-motion: reduce)');
  function reveal(target,animate){
    if(!target||!region.contains(target))return;
    if(region.scrollHeight>region.clientHeight+1){
      region.scrollTo({top:region.scrollTop+target.getBoundingClientRect().top-region.getBoundingClientRect().top-20,behavior:reduce.matches?'auto':'smooth'});
      target.focus({preventScroll:true});
    }else{target.scrollIntoView({block:'nearest',behavior:reduce.matches?'auto':'smooth'});target.focus({preventScroll:true});}
    if(animate)ExampleUI.motion(target,'citation');
  }
  document.addEventListener('click',event=>{
    const link=event.target.closest('[data-citation]');
    if(!link||event.button!==0||event.metaKey||event.ctrlKey||event.shiftKey||event.altKey)return;
    event.preventDefault();const target=document.getElementById(link.getAttribute('href').slice(1));
    reveal(target,true);history.replaceState(null,'',link.getAttribute('href'));
  },options);
  window.addEventListener('load',()=>{
    let id;try{id=decodeURIComponent(location.hash.slice(1));}catch{return;}
    if(id)reveal(document.getElementById(id),false);
  },{...options,once:true});
})();
