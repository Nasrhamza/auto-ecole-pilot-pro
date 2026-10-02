/* Presentation and local preferences only. Sales and stock stay server-controlled. */
const UX = (() => {
  const paths = {
    dashboard:'M3 3h7v7H3z M14 3h7v4h-7z M14 11h7v10h-7z M3 14h7v7H3z',
    analytics:'M3 20h18 M5 17V9 M10 17V4 M15 17v-6 M20 17V7 M4 8l6-5 5 7 6-5',
    caisse:'M3 4h2l3 12h10l3-8H6 M9 20h.01 M18 20h.01',
    products:'M12 3l9 5v9l-9 5-9-5V8z M3 8l9 5 9-5 M12 13v9 M7 5.8l9 5',
    stock:'M3 21V7l9-4 9 4v14 M7 21v-9h10v9 M7 16h10 M7 20h10',
    clients:'M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2 M9 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8 M18 8a3 3 0 0 1 0 6 M22 21v-2a4 4 0 0 0-3-3.8',
    purchases:'M3 8h12v10H3z M15 11h4l3 4v3h-7 M7 21a2 2 0 1 0 0-4 2 2 0 0 0 0 4 M18 21a2 2 0 1 0 0-4 2 2 0 0 0 0 4 M8 2v7 M5 6l3 3 3-3',
    suppliers:'M4 21V5h11v16 M15 11h5v10 M8 8h3 M8 12h3 M8 16h3 M3 21h18',
    history:'M4 4v6h6 M4.5 10a8 8 0 1 1 .5 6 M12 8v5l3 2',
    cash:'M3 7h18v13H3z M3 7V4h14v3 M16 11h5v5h-5z',
    movements:'M7 20V4 M3 8l4-4 4 4 M17 4v16 M13 16l4 4 4-4',
    settings:'M12 8a4 4 0 1 0 0 8 4 4 0 0 0 0-8 M12 2v3 M12 19v3 M2 12h3 M19 12h3 M5 5l2 2 M17 17l2 2 M5 19l2-2 M17 7l2-2',
    trend:'M3 17l6-6 4 3 8-9 M15 5h6v6',
    check:'M5 12l4 4L19 6',
    search:'M10 3a7 7 0 1 0 0 14 7 7 0 0 0 0-14 M15 15l6 6',
    cup:'M5 9h12v8a4 4 0 0 1-4 4H9a4 4 0 0 1-4-4z M17 10h2a3 3 0 0 1 0 6h-2 M8 3v3 M13 3v3',
    milk:'M8 2h8v4l3 4v12H5V10l3-4z M8 6h8 M5 10h14 M9 14h6',
    leaf:'M20 3C10 2 3 6 4 14c1 8 14 9 16-11z M4 21L15 10',
    bread:'M5 11a5 5 0 0 1-1-9h16a5 5 0 0 1-1 9v9H5z M9 7l2-3 M14 7l2-3',
    bottle:'M9 2h6v6l3 5v8H6v-8l3-5z M9 6h6 M6 14h12',
    snack:'M5 3l3 2 4-2 4 2 3-2v18l-3-2-4 2-4-2-3 2z M9 10h6 M9 14h6',
    snow:'M12 2v20 M3.3 7l17.4 10 M3.3 17L20.7 7 M9 4l3 3 3-3 M9 20l3-3 3 3',
    spark:'M12 3l2.5 6.5L21 12l-6.5 2.5L12 21l-2.5-6.5L3 12l6.5-2.5z'
  };
  const read = (key, fallback) => {try{return localStorage.getItem(key)??fallback}catch{return fallback}};
  const write = (key, value) => {try{localStorage.setItem(key,value)}catch{}};
  const media = matchMedia('(prefers-reduced-motion: reduce)');
  let enabled = read('superette_motion','on') !== 'off';
  let pinned = false, available = false, focusBeforeModal = null;
  const motion = () => enabled && !media.matches;
  const icon = name => `<svg class="ui-icon" viewBox="0 0 24 24" aria-hidden="true"><path d="${paths[name]||paths.products}"/></svg>`;
  const empty = (name,title,detail) => `<div class="empty-state"><span class="empty-icon">${icon(name)}</span><strong>${esc(title)}</strong><p>${esc(detail)}</p></div>`;
  const dateKey = d => `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,'0')}-${String(d.getDate()).padStart(2,'0')}`;

  function applyMotion() {
    document.documentElement.classList.toggle('motion-off',!motion());
    const toggle=document.querySelector('#motionEnabled');
    if(toggle){toggle.checked=enabled;toggle.disabled=media.matches}
    const hint=document.querySelector('#motionHint');
    if(hint)hint.textContent=media.matches?'Mouvements réduits activés dans Windows. Cette préférence est respectée.':'Préférence enregistrée sur ce PC.';
    if(!motion()){
      document.getAnimations().forEach(a=>{try{a.finish()}catch{a.cancel()}});
      document.querySelectorAll('.cart-fly').forEach(el=>el.remove());
    }
  }
  applyMotion();
  media.addEventListener('change',applyMotion);
  document.documentElement.dataset.density=read('superette_density','compact');

  function number(el,value) {
    if(!el)return;
    if(el._counter)cancelAnimationFrame(el._counter);
    el._counter=null;
    if(!motion()||!el.getClientRects().length){el.textContent=money(value);return}
    const start=performance.now();
    const tick=now=>{const progress=Math.min(1,(now-start)/440);el.textContent=money(value*(1-Math.pow(1-progress,3)));if(progress<1&&motion())el._counter=requestAnimationFrame(tick);else{el.textContent=money(value);el._counter=null}};
    el._counter=requestAnimationFrame(tick);
  }

  function dashboard() {
    const days=Array.from({length:7},(_,i)=>{const date=new Date();date.setDate(date.getDate()-6+i);return{key:dateKey(date),date,total:0,count:0}});
    const valid=DATA.sales.filter(s=>s.status!=='annulée');
    const recent=valid.filter(s=>days.some(d=>s.time.startsWith(d.key)));
    for(const s of recent){const d=days.find(d=>s.time.startsWith(d.key));d.total+=s.total;d.count++}
    const max=Math.max(...days.map(d=>d.total),1),weekTotal=days.reduce((sum,d)=>sum+d.total,0);
    const todaySales=valid.filter(s=>s.time.startsWith(todayKey()));
    const low=DATA.products.filter(p=>p.stock<=p.min_stock).length;
    const caption=[AUTH.username?`Bonjour ${AUTH.username}.`:'Bienvenue.',`${todaySales.length} vente${todaySales.length!==1?'s':''} aujourd’hui.`,low?`${low} produit${low>1?'s':''} à réapprovisionner.`:'Votre stock est à jour.'];
    $('#welcomeCaption').textContent=caption.join(' ');
    $('#dashboardDate').textContent=new Date().toLocaleDateString('fr-TN',{weekday:'long',day:'numeric',month:'long'});
    $('#weekRevenue').textContent=money(weekTotal);
    $('#weekSalesCount').textContent=`${recent.length} vente${recent.length!==1?'s':''}`;
    const describe=d=>`${d.date.toLocaleDateString('fr-TN',{weekday:'long',day:'numeric',month:'long'})} · ${money(d.total)} · ${d.count} vente${d.count!==1?'s':''}`;
    $('#salesBars').innerHTML=days.map((d,i)=>`<button type="button" class="sales-bar ${i===6?'today':''}" aria-label="${esc(describe(d))}" title="${esc(describe(d))}"><span class="bar-value">${d.total?d.total.toLocaleString('fr-TN',{maximumFractionDigits:1}):'—'}</span><span class="sales-bar-track"><span class="sales-bar-fill" style="--bar-height:${100*d.total/max}%;--delay:${i*35}ms"></span></span><span class="bar-day">${i===6?'Aujourd’hui':d.date.toLocaleDateString('fr-TN',{weekday:'short'})}</span></button>`).join('');
    const defaultCaption=recent.length?'Survolez un jour pour voir le détail. Montants en DT.':'Votre première vente fera apparaître l’activité de la journée.';
    $('#chartCaption').textContent=defaultCaption;
    $$('#salesBars .sales-bar').forEach((button,i)=>{button.onmouseenter=button.onfocus=()=>{$('#chartCaption').textContent=describe(days[i])};button.onmouseleave=button.onblur=()=>{$('#chartCaption').textContent=defaultCaption}});
    const totals=new Map();
    for(const s of recent)for(const item of s.items){if(!item.product_id)continue;const row=totals.get(item.product_id)||{name:item.name,qty:0};row.qty+=item.qty;totals.set(item.product_id,row)}
    const best=[...totals.values()].sort((a,b)=>b.qty-a.qty).slice(0,5),top=Math.max(best[0]?.qty||0,1);
    $('#bestSellers').innerHTML=best.length?best.map((p,i)=>`<div class="rank-row"><span class="rank-no">0${i+1}</span><div><span class="rank-title" title="${esc(p.name)}">${esc(p.name)}</span><div class="rank-track"><div class="rank-fill" style="--rank-width:${p.qty/top*100}%"></div></div></div><span class="rank-qty">${p.qty} vendu${p.qty>1?'s':''}</span></div>`).join(''):empty('trend','Vos futurs best-sellers','Ils apparaîtront ici à partir de vos ventes réelles.');
    animateMetrics();
  }

  function animateMetrics(){if(!$('#dashboard').classList.contains('active'))return;$$('[data-metric-value]').forEach(el=>number(el,Number(el.dataset.metricValue)))}

  function productIcon(category) {
    const c=(category||'').toLowerCase();
    if(/lait|fromage|yaourt/.test(c))return'milk';
    if(/boisson|eau|jus/.test(c))return'bottle';
    if(/fruit|légume/.test(c))return'leaf';
    if(/boulanger|pain/.test(c))return'bread';
    if(/snack|biscuit|confiserie/.test(c))return'snack';
    if(/surgelé|froid/.test(c))return'snow';
    if(/hygiène|entretien/.test(c))return'spark';
    return'products';
  }
  function matchesProduct(p){return(!pinned||p.pinned)&&(!available||p.stock>0)}
  function catalog(count){
    $('#catalogCount').textContent=`${count} / ${DATA.products.length} produit${DATA.products.length!==1?'s':''}`;
    $('#pinnedOnly').setAttribute('aria-pressed',String(pinned));
    $('#pinnedOnly').textContent=pinned?'★ Épinglés':'☆ Épinglés';
    $('#availableOnly').setAttribute('aria-pressed',String(available));
    const compact=document.documentElement.dataset.density!=='comfortable';
    $('#densityToggle').setAttribute('aria-pressed',String(compact));
    $('#densityToggle').textContent=compact?'Vue compacte':'Vue confort';
    $$('#productCatalog [data-add]').forEach(button=>{const p=productByID(+button.dataset.add),name=button.querySelector('.name');if(p&&name)name.insertAdjacentHTML('afterbegin',`<span class="prod-glyph">${icon(productIcon(p.category))}</span>`)});
  }
  function cartUpdated(){
    const qty=cart().items.reduce((sum,i)=>sum+i.qty,0);
    $('#cartQuantity').textContent=qty;
    $('#activeCartName').textContent=`Panier · Client ${activeSlot}`;
    for(const id of ['payCash','payCard','payCredit'])$('#'+id).disabled=qty===0;
  }
  function added(pid,rect){
    const total=$('#cartTotal'),badge=$('#cartQuantity');
    const row=$(`#cartList [data-remove="${pid}"]`)?.closest('.cart-row');
    if(!motion())return;
    total.getAnimations().forEach(a=>a.cancel());
    total.animate([{transform:'scale(1)'},{transform:'scale(1.035)',color:'var(--green-dark)'},{transform:'scale(1)'}],{duration:250,easing:'ease-out'});
    badge.getAnimations().forEach(a=>a.cancel());
    badge.animate([{transform:'scale(.8)'},{transform:'scale(1.12)'},{transform:'scale(1)'}],{duration:260,easing:'ease-out'});
    row?.animate([{background:'var(--green-soft)'},{background:'transparent'}],{duration:450});
    if(!rect||rect.width===0||document.querySelectorAll('.cart-fly').length>=4)return;
    const target=badge.getBoundingClientRect();
    if(target.top<0||target.top>innerHeight)return;
    const el=document.createElement('div');el.className='cart-fly';el.setAttribute('aria-hidden','true');el.textContent='+1';el.style.left=`${rect.left+rect.width/2}px`;el.style.top=`${rect.top+rect.height/2}px`;document.body.appendChild(el);
    const dx=target.left+target.width/2-rect.left-rect.width/2,dy=target.top+target.height/2-rect.top-rect.height/2;
    const animation=el.animate([{transform:'translate(-50%,-50%) scale(1)',opacity:.95},{transform:`translate(calc(-50% + ${dx*.45}px),calc(-50% + ${dy*.5-30}px)) scale(.9)`,opacity:.9,offset:.45},{transform:`translate(calc(-50% + ${dx}px),calc(-50% + ${dy}px)) scale(.4)`,opacity:0}],{duration:360,easing:'cubic-bezier(.2,.65,.35,1)'});
    animation.finished.catch(()=>{}).finally(()=>el.remove());
  }

  function modalOpened(modal){
    if(!focusBeforeModal)focusBeforeModal=document.activeElement;
    const first=modal.querySelector('input:not([type=hidden]):not([type=checkbox]),select,textarea,button');
    first?.focus({preventScroll:true});
  }
  function modalClosed(){if(focusBeforeModal?.isConnected)focusBeforeModal.focus({preventScroll:true});focusBeforeModal=null}

  function init(){
    $$('#nav [data-sec]').forEach(b=>{b.dataset.label=b.textContent.replace(b.querySelector('.nav-icon').textContent,'').trim();b.querySelector('.nav-icon').innerHTML=icon(b.dataset.sec)});
    $('.success-mark').innerHTML=icon('check');
    $('#shortcutsBtn').onclick=()=>{applyMotion();showModal('#shortcutsModal')};
    $('#motionEnabled').onchange=e=>{enabled=e.target.checked;write('superette_motion',enabled?'on':'off');applyMotion()};
    $('#pinnedOnly').onclick=()=>{pinned=!pinned;renderProductViews()};
    $('#availableOnly').onclick=()=>{available=!available;renderProductViews()};
    $('#densityToggle').onclick=()=>{const density=document.documentElement.dataset.density==='comfortable'?'compact':'comfortable';document.documentElement.dataset.density=density;write('superette_density',density);renderProductViews()};
    document.addEventListener('keydown',e=>{
      const modal=$('.modal.show');
      if(modal){
        if(e.key==='Tab'){
          const nodes=[...modal.querySelectorAll('button,a[href],input,select,textarea,[tabindex="0"]')].filter(n=>!n.disabled&&n.getClientRects().length);
          const first=nodes[0],last=nodes[nodes.length-1];
          if(e.shiftKey&&(document.activeElement===first||!modal.contains(document.activeElement))){e.preventDefault();last?.focus()}
          else if(!e.shiftKey&&document.activeElement===last){e.preventDefault();first?.focus()}
        }
        return;
      }
      if($('#app').classList.contains('hidden'))return;
      if(e.key==='F2'){e.preventDefault();goTo('caisse');$('#scanInput').focus();$('#scanInput').select()}
      else if(e.key==='F4'&&$('#caisse').classList.contains('active')){e.preventDefault();openPay('cash')}
      else if(e.altKey&&!e.ctrlKey&&/^[1-6]$/.test(e.key)&&$('#caisse').classList.contains('active')){e.preventDefault();$('#cartTabs [data-slot="'+e.key+'"]').click()}
    });
    applyMotion();
  }
  return{init,icon,empty,dashboard,animateMetrics,matchesProduct,catalog,cartUpdated,added,modalOpened,modalClosed};
})();
