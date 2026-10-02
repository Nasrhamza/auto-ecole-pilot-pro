/* Daily workflow for one instructor. No external dependencies. */
(() => {
  'use strict';
  const q = s => document.querySelector(s), qa = s => [...document.querySelectorAll(s)];
  const labels = {dashboard:'Vue d’ensemble',students:'Mes élèves',schedule:'Mon planning',exams:'Examens',finance:'Paiements & factures',fleet:'Mon véhicule',settings:'Paramètres'};
  Object.assign(PAGE_TITLES, labels);
  q('.side .brand small').textContent='MONITEUR · 12.09.2026';
  tag = value => {
    const text=String(value||'—');
    const state=/échec|annul|hors|incomplet|absence|expir/i.test(text)?'bad':/^(réussi|actif|disponible|terminée|complet|confirmée|payée)$/i.test(text)?'ok':/prévu|planifi|attente|à payer/i.test(text)?'warn':'';
    return '<span class="tag '+state+'">'+esc(text)+'</span>';
  };
  qa('#nav [data-page]').forEach(b => {
    if (!labels[b.dataset.page]) return;
    [...b.childNodes].filter(n => n.nodeType === 3).forEach(n => n.remove());
    b.append(document.createTextNode(labels[b.dataset.page]));
  });
  q('#studentSearch').placeholder = 'Rechercher un élève : nom, CIN, téléphone';
  q('#students .sectionhead h2').textContent = 'Mes élèves';
  q('#students .sectionhead p').textContent = 'Saisissez une recherche, ouvrez le dossier ou planifiez la prochaine séance.';
  q('#dashboard .hero h2').textContent = 'Votre journée, en un coup d’œil';
  q('#dashboard .hero .eyebrow').textContent = 'ESPACE MONITEUR';
  q('#scheduleDate').classList.remove('hidden');
  q('#scheduleDate').setAttribute('aria-label', 'Choisir une date du planning');
  q('#weekPrev').title = q('#weekPrev').ariaLabel = 'Semaine précédente';
  q('#weekNext').title = q('#weekNext').ariaLabel = 'Semaine suivante';
  let planningView='week';
  const views=document.createElement('div');views.className='planning-views';
  views.innerHTML='<button type="button" class="btn ghost" data-planning-view="week" aria-pressed="true">Semaine</button><button type="button" class="btn ghost" data-planning-view="month" aria-pressed="false">Mois</button>';
  q('#schedule .planningbar').before(views);
  views.onclick=e=>{const b=e.target.closest('[data-planning-view]');if(!b)return;planningView=b.dataset.planningView;renderPlanning();};
  function shiftWeek(delta) {
    let d = new Date((q('#scheduleDate').value || todayKey())+'T12:00:00');
    if(planningView==='month'){const requested=d.getDate();d.setDate(1);d.setMonth(d.getMonth()+Math.sign(delta));d.setDate(Math.min(requested,new Date(d.getFullYear(),d.getMonth()+1,0).getDate()));}
    else d=addDays(d,delta);
    q('#scheduleDate').value = localISO(d); WEEK_ANCHOR = d; renderPlanning();
  }
  q('#weekPrev').onclick = () => shiftWeek(-7);
  q('#weekNext').onclick = () => shiftWeek(7);
  q('#scheduleDate').onchange = () => { WEEK_ANCHOR = new Date(q('#scheduleDate').value+'T12:00:00'); renderPlanning(); };
  const dateTime = d => localISO(d)+'T'+String(d.getHours()).padStart(2,'0')+':'+String(d.getMinutes()).padStart(2,'0');
  const clock = d => d.toLocaleTimeString('fr-FR',{hour:'2-digit',minute:'2-digit'});
  const hours = n => Number(n).toLocaleString('fr-FR',{maximumFractionDigits:1})+' h';
  const liveLesson = l => l.status !== 'Annulée';

  function labelTables() {
    qa('table').forEach(table => {
      const titles = [...table.querySelectorAll('thead th')].map(x => x.textContent.trim());
      table.querySelectorAll('tbody tr').forEach(tr => [...tr.children].forEach((td,i) => td.dataset.label = td.colSpan > 1 ? '' : titles[i] || 'Actions'));
    });
  }
  function applyLogo() {
    qa('.brandmark').forEach(mark => {
      let img = mark.querySelector('img');
      if (!img) { img = document.createElement('img'); img.alt = 'Logo'; mark.append(img); }
      const src = BRAND.logo_url || '/api/branding/logo';
      if (img.getAttribute('src') !== src) img.src = src;
    });
  }
  const oldBrand = applyBrand;
  function actionColor(color) {
    if(!/^#[0-9a-f]{6}$/i.test(color))color='#173f58';
    const rgb=[1,3,5].map(i=>parseInt(color.slice(i,i+2),16)/255).map(v=>v<=.04045?v/12.92:((v+.055)/1.055)**2.4);
    const luminance=rgb[0]*.2126+rgb[1]*.7152+rgb[2]*.0722;
    document.documentElement.style.setProperty('--action-bg',color);
    document.documentElement.style.setProperty('--action-ink',luminance>.179?'#102c3c':'#ffffff');
  }
  applyBrand = function() { oldBrand(); applyLogo(); actionColor(DATA.settings?.theme||BRAND.theme||'#173f58'); };
  q('#schoolTheme').addEventListener('input',e=>actionColor(e.target.value));

  // Persist the order with the data file; random desktop ports no longer reset it.
  queueOrder = () => DATA.driving_config?.queue_order || [];
  let queueWrite = Promise.resolve();
  saveQueueOrder = ids => {
    DATA.driving_config.queue_order = [...ids];
    queueWrite = queueWrite.then(async () => {
      try { await api('/api/driving/queue',{method:'POST',body:JSON.stringify(ids)}); }
      catch(e) { toast('Ordre non enregistré : '+e.message,true); }
    });
  };
  const originalOrdered = orderedCoachStudents;
  orderedCoachStudents = function() {
    const rows = originalOrdered(), saved = queueOrder();
    return rows.sort((a,b) => {
      const ai=saved.indexOf(a.x.id), bi=saved.indexOf(b.x.id);
      if(ai>=0 || bi>=0) return ai<0?1:bi<0?-1:ai-bi;
      if(a.exam || b.exam) { if(!a.exam)return 1;if(!b.exam)return -1;const d=a.exam.date.localeCompare(b.exam.date);if(d)return d; }
      if(!!a.next !== !!b.next) return a.next?1:-1;
      return b.remaining-a.remaining || a.x.name.localeCompare(b.x.name,'fr');
    });
  };
  const oldQueue = renderCoachQueue;
  renderCoachQueue = function() {
    oldQueue();
    const box=q('#coachQueue'); if(!box)return;
    if(!q('#resetQueue')) {
      const b=document.createElement('button');b.id='resetQueue';b.className='btn ghost';b.textContent='Ordre automatique';
      b.onclick=()=>{saveQueueOrder([]);renderCoachQueue();};box.querySelector('.panelhead').append(b);
    }
    qa('#coachQueue [data-queue-id]').forEach((row,i,all) => {
      row.draggable=false;
      const arrows=row.querySelectorAll('[data-queue-move]');if(arrows[0])arrows[0].disabled=i===0;if(arrows[1])arrows[1].disabled=i===all.length-1;
      const id=+row.dataset.queueId, s=DATA.students.find(x=>x.id===id);
      const remaining=Math.max(0,(+s.driving_target||0)-coachDone(id)), scheduled=coachPlanned(id);
      row.querySelector('.queuehours small').textContent=hours(coachDone(id))+' réalisées · '+hours(scheduled)+' déjà prévues';
      const button=row.querySelector('[data-queue-student]');
      button.title=hours(Math.max(0,remaining-scheduled))+' à placer dans le planning';
    });
    box.querySelector('.panelhead p').textContent='Utilisez les flèches pour définir l’ordre de passage. Ouvrez « Planifier » pour choisir un créneau.';
  };
  const oldPlanning=renderPlanning;
  renderPlanning=function() {
    oldPlanning();
    const date=q('#scheduleDate').value;
    const chosen=new Date(date+'T12:00:00'),isMonth=planningView==='month';
    const periodStart=isMonth?new Date(chosen.getFullYear(),chosen.getMonth(),1,12):monday(chosen);
    const periodEnd=isMonth?new Date(chosen.getFullYear(),chosen.getMonth()+1,1,12):addDays(periodStart,7);
    const cells=[];
    if(isMonth)for(let i=0;i<(periodStart.getDay()+6)%7;i++)cells.push('<span class="calendar-blank" aria-hidden="true"></span>');
    for(let d=new Date(periodStart);d<periodEnd;d=addDays(d,1)){
      const key=localISO(d),lessons=DATA.lessons.filter(l=>day(l.start)===key&&liveLesson(l)&&(!+q('#scheduleInstructor').value||l.instructor_id===+q('#scheduleInstructor').value)).sort((a,b)=>a.start.localeCompare(b.start));
      cells.push('<button type="button" class="daypick '+(key===date?'selected ':'')+(key===todayKey()?'today':'')+'" data-day="'+key+'" aria-pressed="'+(key===date)+'" aria-label="'+d.toLocaleDateString('fr-FR',{weekday:'long',day:'numeric',month:'long',year:'numeric'})+'"><span>'+d.toLocaleDateString('fr-FR',{weekday:'short'})+'</span><b>'+d.getDate()+'</b><small>'+lessons.length+' séance'+(lessons.length!==1?'s':'')+'</small>'+lessons.slice(0,isMonth?1:3).map(l=>'<span class="calendar-lesson" title="'+esc(nameOf(l.student_id))+'">'+clock(new Date(l.start))+' · '+esc(nameOf(l.student_id))+'</span>').join('')+(lessons.length>(isMonth?1:3)?'<small>+'+(lessons.length-(isMonth?1:3))+' autre(s)</small>':'')+'</button>');
    }
    q('#weekGrid').innerHTML=cells.join('');q('#weekGrid').classList.toggle('month-calendar',isMonth);
    if(isMonth)q('#weekTitle').textContent=chosen.toLocaleDateString('fr-FR',{month:'long',year:'numeric'});
    qa('[data-planning-view]').forEach(b=>b.setAttribute('aria-pressed',String(b.dataset.planningView===planningView)));
    q('#weekPrev').title=q('#weekPrev').ariaLabel=isMonth?'Mois précédent':'Semaine précédente';
    q('#weekNext').title=q('#weekNext').ariaLabel=isMonth?'Mois suivant':'Semaine suivante';
    const rows=DATA.lessons.filter(l=>day(l.start)===date&&(!+q('#scheduleInstructor').value||l.instructor_id===+q('#scheduleInstructor').value)).sort((a,b)=>a.start.localeCompare(b.start));
    qa('#dayAgenda .agenda-item').forEach((item,i) => {
      const l=rows[i];if(!l)return;
      const start=new Date(l.start),end=new Date(start.getTime()+l.duration*60000);
      item.querySelector('.agenda-time').innerHTML='<span class="agenda-start">'+clock(start)+'</span><small>jusqu’à '+clock(end)+'</small>';
      item.dataset.finished=l.status==='Terminée';item.dataset.canceled=l.status==='Annulée';
      item.querySelector('.typeicon').title=l.kind;
      item.querySelectorAll('[data-lesson-status]').forEach(b=>{
        b.textContent=b.dataset.lessonStatus.endsWith('Terminée')?'Terminée':'Absence';
        b.disabled=l.status===b.dataset.lessonStatus.split(':')[1] || l.status==='Annulée';
      });
      item.querySelector('[data-edit]').textContent='Modifier';
      const pdf=item.querySelector('[title="Bon rendez-vous"]');if(pdf)pdf.textContent='Bon RDV';
      const wa=item.querySelector('[title="Rappel WhatsApp"]');
      if(wa){wa.textContent='WhatsApp';if(!DATA.students.find(s=>s.id===l.student_id)?.phone)wa.remove();}
    });
    const active=rows.filter(liveLesson),done=active.filter(l=>l.status==='Terminée'),pending=active.filter(l=>['Planifiée','Confirmée'].includes(l.status));
    q('#planningSummary').innerHTML='<h3 class="overview-title">Bilan de la journée</h3><div class="overview-grid">'+[
      [new Set(active.map(l=>l.student_id)).size,'ÉLÈVES'],[hours(done.reduce((n,l)=>n+l.duration/60,0)),'RÉALISÉES'],[hours(pending.reduce((n,l)=>n+l.duration/60,0)),'À EFFECTUER']
    ].map(([n,label])=>'<div><b>'+n+'</b><small>'+label+'</small></div>').join('')+'</div><p class="coachhint">Seules les séances « Terminée » comptent dans les heures réalisées. Une absence ne déduit pas d’heures.</p>';
  };
  const oldCoach=applyCoachMode;
  applyCoachMode=function(){oldCoach();renderPlanning();labelTables();};
  q('#studentSearch').oninput=q('#studentStatus').onchange=()=>{render();applyCoachMode();};
  const oldRender=render;
  render=function(){
    oldRender();labelTables();applyLogo();
    const now=new Date(),next=DATA.lessons.filter(l=>['Planifiée','Confirmée'].includes(l.status)&&new Date(new Date(l.start).getTime()+l.duration*60000)>now).sort((a,b)=>a.start.localeCompare(b.start))[0];
    q('#nextLesson').textContent=next?(new Date(next.start)<=now?'En cours · ':'Prochaine séance · ')+fmt(next.start)+' · '+nameOf(next.student_id):'Aucune prochaine séance planifiée';
    q('#pageTitle').textContent=labels[q('#nav .active')?.dataset.page]||q('#pageTitle').textContent;
  };

  // Three settings destinations; existing forms, saved values and submit handlers stay intact.
  const settings=q('#settings'), splits=[...settings.children].filter(x=>x.classList.contains('split'));
  const tabs=document.createElement('div');tabs.className='settings-tabs';tabs.setAttribute('role','tablist');tabs.setAttribute('aria-label','Catégories de paramètres');
  settings.querySelector('.sectionhead').after(tabs);
  ['Identité & logo','Horaires & documents','Sauvegarde & historique','Utilisateurs'].forEach((label,i)=>{
    const b=document.createElement('button');b.className='settings-tab';b.textContent=label;b.type='button';b.setAttribute('role','tab');b.setAttribute('aria-controls',i===0?'settingsForm':i===1?'configForm':'settingsBackup');
    b.onclick=()=>{
      q('#settingsForm').classList.toggle('settings-hidden',i!==0);q('#configForm').classList.toggle('settings-hidden',i!==1);
      splits[0].classList.toggle('settings-hidden',i>1);splits[1].classList.toggle('settings-hidden',i!==2);splits[2].classList.toggle('settings-hidden',i!==3);
      [...tabs.children].forEach((tab,n)=>tab.setAttribute('aria-selected',String(n===i)));
    };tabs.append(b);
  });
  splits[1].id='settingsBackup';tabs.firstChild.click();
  q('#configForm .sectionhead h2').textContent='Horaires, rappels et textes des documents';
  q('#cfgDuration').max='480';q('#cfgDuration').step='15';
  qa('#settings .field').forEach(field=>{const control=field.querySelector('input,textarea,select'),label=field.querySelector('label');if(control?.id&&label)label.htmlFor=control.id;});

  // Dossier displays every attachment, including several files of the same category.
  const oldProfile=openProfile;
  openProfile=function(id){
    oldProfile(id);const s=DATA.students.find(x=>x.id==id);if(!s)return;
    q('#profileDocCount').textContent=DATA.driving_documents.filter(d=>d.student_id==id).length+' fichier(s)';
    const docs=DATA.driving_documents.filter(d=>d.student_id==id), grid=q('#profileDocs');
    grid.innerHTML=['CIN','Photo','Certificat médical','Contrat','Justificatif',...new Set(docs.map(d=>d.category).filter(c=>!['CIN','Photo','Certificat médical','Contrat','Justificatif'].includes(c)))].map(cat=>{
      const files=docs.filter(d=>d.category===cat);
      const latest=files[files.length-1];
      return '<article class="doccard '+(files.length?'ready':'')+'"><h4>'+esc(cat)+'</h4><small>'+files.length+' fichier(s)</small><div class="profile-doc-list">'+files.map(d=>'<div><a class="btn ghost" target="_blank" rel="noopener" href="/api/driving/document/download?id='+d.id+'&inline=1">'+esc(d.name)+'</a><small>'+(d.expiry?'Validité : '+esc(fmt(d.expiry)):'Sans date d’expiration')+'</small><button class="btn danger" title="Supprimer ce fichier" data-delete-doc="'+d.id+'">Supprimer</button></div>').join('')+'</div><button class="btn primary" data-add-doc="'+esc(cat)+'"'+(latest?' data-replace-doc="'+latest.id+'"':'')+'>'+(latest?'Remplacer le fichier':'Ajouter un fichier')+'</button></article>';
    }).join('');
    let extra=q('#profileProgress');if(!extra){extra=document.createElement('section');extra.id='profileProgress';extra.className='profile-extra';grid.before(extra);}
    const done=coachDone(+id),target=+s.driving_target||0,planned=coachPlanned(+id);
    extra.innerHTML='<h3>Suivi des heures de conduite</h3><div class="coachhours">'+[[hours(done),'Réalisées'],[hours(Math.max(0,target-done)),'Restantes'],[hours(planned),'Déjà planifiées']].map(([n,t])=>'<div class="coachhour"><b>'+n+'</b><small>'+t+'</small></div>').join('')+'</div><p class="field-hint">Forfait : '+hours(target)+' · Encore à placer : '+hours(Math.max(0,target-done-planned))+'</p><div class="actions"><button class="btn primary" data-profile-plan="'+id+'">Planifier une séance</button><button class="btn ghost" data-profile-payment="'+id+'">Enregistrer un paiement</button></div>';
  };
  document.addEventListener('click',e=>{
    const b=e.target.closest('[data-profile-plan],[data-profile-payment]');if(!b)return;
    const id=b.dataset.profilePlan||b.dataset.profilePayment;
    q('#profileClose').click();openForm(b.dataset.profilePlan?'lesson':'payment');q('#formFields [name=student_id]').value=id;
  });

  function lessonStarts(form) {
    const start=new Date(form.elements.namedItem('start').value),mode=form.elements.namedItem('repeat_mode')?.value||'once';
    if(Number.isNaN(start.getTime()))throw Error('Choisissez une date et une heure.');
    if(mode==='once')return [dateTime(start)];
    if(mode==='weekly')return Array.from({length:Math.max(1,Math.min(52,+form.elements.namedItem('repeat_weeks').value||1))},(_,i)=>dateTime(addDays(start,7*i)));
    let end;
    if(mode==='week')end=addDays(monday(start),6);
    else if(mode==='month')end=new Date(start.getFullYear(),start.getMonth()+1,0,12);
    else end=new Date(form.elements.namedItem('repeat_until').value+'T12:00:00');
    if(Number.isNaN(end.getTime())||localISO(end)<localISO(start))throw Error('La fin de période doit être après ou le jour de la première séance.');
    const weekdays=qa('#repeatDays input:checked').map(n=>+n.value),result=[];
    if(!weekdays.length)throw Error('Cochez au moins un jour de la semaine.');
    for(let d=new Date(start),i=0;localISO(d)<=localISO(end);d=addDays(d,1),i++){
      if(i>365)throw Error('Choisissez une période de 12 mois au maximum.');
      if(weekdays.includes(d.getDay()))result.push(dateTime(d));
    }
    if(!result.length)throw Error('Aucun des jours cochés ne se trouve dans cette période.');
    return result;
  }
  let returnFocus=null,requestID='';
  const oldOpen=openForm;
  openForm=function(type,id=0){
    returnFocus=document.activeElement;requestID=Array.from(crypto.getRandomValues(new Uint8Array(16)),n=>n.toString(16).padStart(2,'0')).join('');oldOpen(type,id);
    const form=q('#entityForm');let feedback=q('#formFeedback');
    if(!feedback){feedback=document.createElement('div');feedback.id='formFeedback';feedback.className='form-feedback';feedback.setAttribute('role','alert');q('#formFields').before(feedback);}feedback.textContent='';
    qa('#formFields .field').forEach((field,i)=>{const c=field.querySelector('input,select,textarea'),label=field.querySelector('label');if(!c)return;c.id='entity-field-'+i;if(label)label.htmlFor=c.id;});
    const required={student:['name'],lesson:['student_id','start','duration'],exam:['student_id','date'],payment:['student_id','time','amount'],invoice:['student_id','date','label','amount'],vehicle:['brand','plate'],expense:['label','amount']};
    (required[type]||[]).forEach(n=>{const c=form.elements.namedItem(n);if(c)c.required=true;});
    for(const n of ['driving_target','theory_target','theory_hours','driving_hours']){const c=form.elements.namedItem(n);if(c){c.step='.5';c.min='0';}}
    if(type==='lesson'){
      const duration=form.elements.namedItem('duration');duration.min='15';duration.max='480';duration.step='1';
      const repeat=form.elements.namedItem('repeat_weeks');repeat.min='1';repeat.max='52';repeat.step='1';repeat.closest('.field').classList.add('hidden');
      if(!id){
        const recurrence=document.createElement('fieldset');recurrence.className='lesson-recurrence span2';
        recurrence.innerHTML='<legend>Programmer plusieurs séances</legend><label for="repeatMode">Période</label><select class="input" id="repeatMode" name="repeat_mode"><option value="once">Une seule séance</option><option value="weekly">Même jour, chaque semaine</option><option value="week">Plusieurs jours de cette semaine</option><option value="month">Plusieurs jours de ce mois</option><option value="custom">Jusqu’à une date choisie</option></select><div id="repeatDays" class="repeat-days hidden">'+[[1,'Lun'],[2,'Mar'],[3,'Mer'],[4,'Jeu'],[5,'Ven'],[6,'Sam'],[0,'Dim']].map(([n,label])=>'<label><input type="checkbox" value="'+n+'"> '+label+'</label>').join('')+'</div><div id="repeatUntilWrap" class="hidden"><label for="repeatUntil">Dernier jour</label><input class="input" id="repeatUntil" name="repeat_until" type="date"></div><p class="field-hint">Même élève, même heure et même durée. La période commence à la date choisie ci-dessus.</p><p id="repeatPreview" class="field-hint" role="status"></p>';
        q('#formFields').append(recurrence);
        const weekday=new Date(form.elements.namedItem('start').value).getDay();recurrence.querySelector('input[value="'+weekday+'"]')?.setAttribute('checked','');
        const preview=()=>{
          const mode=q('#repeatMode').value,multi=['week','month','custom'].includes(mode);
          repeat.closest('.field').classList.toggle('hidden',mode!=='weekly');q('#repeatDays').classList.toggle('hidden',!multi);q('#repeatUntilWrap').classList.toggle('hidden',mode!=='custom');
          try{const dates=lessonStarts(form);q('#repeatPreview').textContent=dates.length+' séance(s) · '+dates.map(s=>new Date(s).toLocaleDateString('fr-FR',{day:'numeric',month:'short'})).join(' · ');}catch(e){q('#repeatPreview').textContent=e.message;}
        };recurrence.oninput=preview;repeat.oninput=preview;form.elements.namedItem('start').addEventListener('input',preview);preview();
      }
      const suggestion=document.createElement('button');suggestion.type='button';suggestion.className='btn ghost';suggestion.textContent='Trouver le prochain créneau libre';
      suggestion.onclick=()=>{
        const startField=form.elements.namedItem('start'),minutes=+duration.value||60,key=(startField.value||q('#scheduleDate').value||todayKey()).slice(0,10),cfg=DATA.driving_config;
        let d=new Date(key+'T'+(cfg.open_time||'08:00'));const close=new Date(key+'T'+(cfg.close_time||'18:00'));
        if(key===todayKey()&&d<new Date()){d=new Date();d.setSeconds(0,0);d.setMinutes(Math.ceil(d.getMinutes()/15)*15);}
        for(;d.getTime()+minutes*60000<=close.getTime();d=new Date(d.getTime()+15*60000)){
          if(!DATA.lessons.some(l=>l.id!==id&&liveLesson(l)&&d<new Date(new Date(l.start).getTime()+l.duration*60000)&&new Date(l.start)<new Date(d.getTime()+minutes*60000))){startField.value=dateTime(d);feedback.textContent='';return;}
        }feedback.textContent='Aucun créneau assez long ce jour. Choisissez une autre date.';
      };form.elements.namedItem('start').after(suggestion);
    }
    if(type==='exam'){
      const decision=form.elements.namedItem('failure_action');if(decision)decision.value='record';
      form.elements.namedItem('result').dispatchEvent(new Event('change'));
    }
    requestAnimationFrame(()=>qa('#formFields input,#formFields select,#formFields textarea').find(c=>c.getClientRects().length&&!c.disabled)?.focus({preventScroll:true}));
  };

  q('#entityForm').onsubmit=async function(e){
    e.preventDefault();if(this.dataset.saving==='true'||!this.reportValidity())return;
    const type=q('#entityType').value,form=new FormData(this),obj={id:+q('#entityID').value},feedback=q('#formFeedback');
    const numeric=new Set(['invoice_id','student_id','instructor_id','vehicle_id','package_id','duration','mileage','maintenance_due','theory_target','driving_target','next_due','repeat_weeks','year','oil_change_due','amount','package','price','cost','theory_hours','driving_hours','hourly_rate','monthly_salary','purchase_price']);
    form.forEach((v,k)=>{if(!['failure_action','remedial_hours','remedial_price','reexam_date','repeat_mode','repeat_until'].includes(k))obj[k]=numeric.has(k)?+v:v;});
    const buttons=[...this.querySelectorAll('button')];this.dataset.saving='true';buttons.forEach(b=>b.disabled=true);feedback.textContent='';let saved=0,batch=[];
    try{
      if(type==='exam'&&obj.result==='Échec'){
        const action=(obj.type==='Code')?'record':form.get('failure_action')||'record';
        obj.request_id=requestID;
        await api('/api/driving/exam-followup',{method:'POST',body:JSON.stringify({exam:obj,action,hours:+form.get('remedial_hours')||0,price:+form.get('remedial_price')||0,next_date:form.get('reexam_date')||''})});
      }else if(type==='document'){
        const r=await fetch('/api/driving/document',{method:'POST',body:form}),j=await r.json();if(!r.ok||!j.ok)throw Error(j.error||'Impossible d’enregistrer le fichier');
      }else{
        const starts=type==='lesson'?(obj.id?[obj.start]:lessonStarts(this)):[];delete obj.repeat_weeks;
        const endpoint={package_sale:'package-sale',vehicle_event:'vehicle-event'}[type]||type;
        batch=type==='lesson'?starts.map(start=>({...obj,start})):[{...obj}];
        if(type==='lesson'&&!['Annulée','Absence'].includes(obj.status)){
          const pupil=DATA.students.find(s=>s.id===obj.student_id),drive=['Conduite','Créneau'].includes(obj.kind);
          if(!pupil)throw Error('Choisissez un candidat.');
          const used=DATA.lessons.filter(l=>l.student_id===obj.student_id&&l.id!==obj.id&&!['Annulée','Absence'].includes(l.status)&&['Conduite','Créneau'].includes(l.kind)===drive).reduce((n,l)=>n+l.duration/60,0);
          const target=+(drive?pupil.driving_target:pupil.theory_target)||0,requested=batch.reduce((n,l)=>n+l.duration/60,0);
          if(used+requested>target+.0001)throw Error('Cette programmation demande '+requested.toFixed(1)+' h, mais il reste '+Math.max(0,target-used).toFixed(1)+' h disponibles. Ajoutez un forfait ou réduisez le nombre de séances. Aucune séance créée.');
          const vehicle=DATA.vehicles.find(v=>v.id===obj.vehicle_id);
          if(drive&&vehicle&&batch.some(l=>(vehicle.insurance_expiry&&vehicle.insurance_expiry<day(l.start))||(vehicle.visit_expiry&&vehicle.visit_expiry<day(l.start))))throw Error('Assurance ou visite technique expirée pendant la période choisie. Aucune séance créée.');
        }
        if(type==='lesson'&&obj.status!=='Annulée'){
          for(const l of batch){const start=new Date(l.start),end=new Date(start.getTime()+l.duration*60000),cfg=DATA.driving_config;if(dateTime(start).slice(11)<(cfg.open_time||'00:00')||dateTime(end).slice(11)>(cfg.close_time||'23:59')||day(dateTime(start))!==day(dateTime(end)))throw Error('Séance en dehors des horaires le '+fmt(l.start));if(DATA.lessons.some(x=>x.id!==obj.id&&liveLesson(x)&&new Date(x.start)<end&&start<new Date(new Date(x.start).getTime()+x.duration*60000)))throw Error('Créneau déjà occupé le '+fmt(l.start)+'. Aucune séance créée : choisissez une autre heure.');}
        }
        for(const item of batch){await api('/api/driving/'+endpoint,{method:'POST',body:JSON.stringify(item)});saved++;}
      }
      q('#modal').classList.remove('show');toast('Enregistrement réussi');await reload();
    }catch(err){
      feedback.textContent=(saved?saved+' séance(s) enregistrée(s). Les suivantes n’ont pas été enregistrées. ':'')+err.message;
      if(saved&&type==='lesson'&&batch[saved]){this.elements.namedItem('start').value=batch[saved].start;this.elements.namedItem('repeat_weeks').value=batch.length-saved;this.elements.namedItem('start').dispatchEvent(new Event('input'));try{await reload();}catch{}}
      feedback.scrollIntoView({block:'nearest'});
    }finally{delete this.dataset.saving;buttons.forEach(b=>b.disabled=false);}
  };

  // Native scrolling, keyboard focus containment, and no wheel/touch interception.
  const modals=qa('#modal,#profileModal');
  const observer=new MutationObserver(()=>{
    const opened=modals.some(m=>m.classList.contains('show'));document.body.classList.toggle('has-dialog',opened);
    if(!opened&&returnFocus?.isConnected){returnFocus.focus({preventScroll:true});returnFocus=null;}
  });modals.forEach(m=>observer.observe(m,{attributes:true,attributeFilter:['class']}));
  document.addEventListener('keydown',e=>{
    const modal=modals.find(m=>m.classList.contains('show'));if(!modal)return;
    if(e.key==='Escape'&&q('#entityForm').dataset.saving!=='true'){e.preventDefault();if(modal.id==='profileModal')q('#profileClose').click();else q('#cancelModal').click();}
    if(e.key==='Tab'){
      const nodes=[...modal.querySelectorAll('button,input,select,textarea,a[href]')].filter(n=>!n.disabled&&n.getClientRects().length),first=nodes[0],last=nodes[nodes.length-1];
      if(e.shiftKey&&(document.activeElement===first||!modal.contains(document.activeElement))){e.preventDefault();last?.focus();}
      else if(!e.shiftKey&&document.activeElement===last){e.preventDefault();first?.focus();}
    }
  });
  q('#nav').addEventListener('click',e=>{const b=e.target.closest('[data-page]');if(!b)return;requestAnimationFrame(()=>{const h=q('.top h1');if(labels[b.dataset.page])h.textContent=labels[b.dataset.page];window.scrollTo({top:0});qa('#nav [data-page]').forEach(x=>x.setAttribute('aria-current',x===b?'page':'false'));});});
  applyLogo();
})();

/* Presentation only: keep original forms, handlers, API calls and permissions. */
(() => {
  const q=s=>document.querySelector(s),qa=s=>[...document.querySelectorAll(s)];
  const titles={dashboard:'Tableau de bord',students:'Candidats',schedule:'Planning',exams:'Examens',finance:'Paiements',fleet:'Équipe & véhicules',operations:'Dossiers & opérations',alertsPage:'Alertes',reports:'Rapports',settings:'Administration'};
  const subtitles={dashboard:'Votre activité, aujourd’hui',students:'Dossiers, progression et prochaines étapes',schedule:'Séances, disponibilités et ordre de passage',exams:'Épreuves, résultats et nouvelles tentatives',finance:'Encaissements, dépenses et factures',fleet:'Disponibilité et suivi du matériel',operations:'Documents et suivi administratif',alertsPage:'Priorités et échéances à traiter',reports:'Activité et résultats sur la période',settings:'Identité, préférences et accès'};
  const paths={dashboard:'M3 3h7v7H3z M14 3h7v7h-7z M3 14h7v7H3z M14 14h7v7h-7z',students:'M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2 M16 3a4 4 0 0 1 0 8 M22 21v-2a4 4 0 0 0-3-3.9 M13 7a4 4 0 1 1-8 0 4 4 0 0 1 8 0',schedule:'M4 5h16v16H4z M4 10h16 M8 3v4 M16 3v4 M8 14h2 M14 14h2 M8 17h2',exams:'M9 3h6v4H9z M9 5H5v16h14V5h-4 M8 14l3 3 5-6',finance:'M3 5h18v14H3z M3 9h18 M7 15h3 M15 14h3',fleet:'M3 16V9l3-5h12l3 5v7 M3 10h18 M5 16h14 M5 16v4 M19 16v4 M6 13h2 M16 13h2',operations:'M3 6h7l2 3h9v11H3z M3 6V4h7l2 2h6v3',alertsPage:'M12 3L2 21h20L12 3z M12 9v5 M12 17h.01',reports:'M4 3v18h17 M8 17v-4 M13 17V9 M18 17V5',settings:'M4 7h16 M4 17h16 M8 4v6 M16 14v6'};
  const iconAssets={dashboard:'nav-dashboard-ui.png',students:'nav-candidates-ui.png',schedule:'nav-planning-ui.png',exams:'nav-exams-ui.png',finance:'nav-payments-ui.png',fleet:'nav-fleet-ui.png',operations:'nav-dossiers-ui.png',alertsPage:'nav-alerts-ui.png',reports:'nav-reports-ui.png',settings:'nav-settings-ui.png'};
  const icon=k=>'<img class="pilot-custom-icon" src="/icons/'+iconAssets[k]+'" alt="" aria-hidden="true" draggable="false">';
  const groups=[['Opérations',['dashboard','students','schedule','exams']],['Gestion',['finance','fleet','operations']],['Analyse',['alertsPage','reports']],['Système',['settings']]];
  groups.forEach(([label,keys])=>{
    const h=document.createElement('div');h.className='nav-group';h.textContent=label;q('#nav').append(h);
    keys.forEach(key=>{const b=q('#nav [data-page="'+key+'"]');b.querySelector('.ico').innerHTML=icon(key);[...b.childNodes].filter(n=>n.nodeType===3).forEach(n=>n.remove());b.insertBefore(document.createTextNode(titles[key]),b.querySelector('.nav-count'));q('#nav').append(b);});
  });
  q('.side .brand small').textContent='PILOT PRO / V1';
  const user=document.createElement('div');user.className='pilot-user';q('#logout').innerHTML='<img class="pilot-custom-icon pilot-action-picture" src="/icons/action-logout-ui.png" alt="" aria-hidden="true" draggable="false"><span>Déconnexion</span>';user.append(q('#userChip'),q('#logout'));q('.side').append(user);
  q('.sidefoot').innerHTML='<b>POSTE DE PILOTAGE</b>Données locales · Sauvegarde automatique';
  const head=q('.top>div');head.insertAdjacentHTML('afterbegin','<span class="command-eyebrow">PILOT PRO / OPÉRATIONS</span>');
  const date=document.createElement('span');date.className='command-date';date.textContent=new Date().toLocaleDateString('fr-FR',{day:'numeric',month:'long'});q('.top .actions').append(date);
  const search=document.createElement('input');search.className='input command-search';search.placeholder='Rechercher un candidat…';search.setAttribute('aria-label','Recherche rapide candidat');
  search.oninput=()=>{q('#studentSearch').value=search.value;q('#nav [data-page="students"]').click();q('#studentSearch').dispatchEvent(new Event('input'));};q('.top .actions').append(search);
  const newLesson=document.createElement('button');newLesson.className='btn primary';newLesson.dataset.open='lesson';newLesson.textContent='+ Séance';q('.top .actions').append(newLesson);
  function command(){const page=q('#nav .active')?.dataset.page||'dashboard';q('#pageTitle').textContent=titles[page];q('#today').textContent=subtitles[page];Object.assign(PAGE_TITLES,titles);}
  q('#nav').addEventListener('click',()=>requestAnimationFrame(command));
  q('#students .sectionhead h2').textContent='Dossiers candidats';
  q('#dashboard .hero .eyebrow').textContent='SITUATION DU JOUR';
  q('#dashboard .hero h2').textContent='La journée en mouvement';
  q('#authForm .eyebrow').textContent='ACCÈS À VOTRE ESPACE';
  q('#username').autocomplete='username';q('#password').autocomplete='current-password';q('#password').removeAttribute('minlength');
  qa('.auth .field').forEach(f=>{const c=f.querySelector('input,textarea'),l=f.querySelector('label');if(c&&l)l.htmlFor=c.id;});
  q('#modal').setAttribute('role','dialog');q('#modal').setAttribute('aria-modal','true');q('#modal').setAttribute('aria-labelledby','modalTitle');
  q('#profileModal').setAttribute('role','dialog');q('#profileModal').setAttribute('aria-modal','true');q('#profileModal').setAttribute('aria-labelledby','profileName');
  q('#profileClose').setAttribute('aria-label','Fermer le dossier');q('#closeModal').setAttribute('aria-label','Fermer le formulaire');
  const baseToast=toast;toast=function(text,bad=false){baseToast(text,bad);q('#toast').dataset.kind=bad?'error':'success';q('#toast').setAttribute('role',bad?'alert':'status');};
  function palette(accent,primary){
    const style=document.documentElement.style,valid=c=>/^#[0-9a-f]{6}$/i.test(c||'');
    accent=valid(accent)?accent:'#316c76';primary=valid(primary)?primary:'#26343b';
    style.setProperty('--pilot-accent',accent);style.setProperty('--pilot-graphite',primary);
    style.setProperty('--pilot-soft','color-mix(in srgb,'+accent+' 8%,white)');
    const contrast=c=>{const rgb=[1,3,5].map(i=>parseInt(c.slice(i,i+2),16)/255).map(v=>v<=.04045?v/12.92:((v+.055)/1.055)**2.4);return rgb[0]*.2126+rgb[1]*.7152+rgb[2]*.0722>.179?'#17232a':'#ffffff';};
    style.setProperty('--pilot-action-ink',contrast(accent));style.setProperty('--pilot-nav-ink',contrast(primary));
  }
  let appearance='administrative';
  try{appearance=localStorage.getItem('pilot-appearance-v1')||appearance;}catch{}
  const preset=document.createElement('div');preset.className='field span2';
  preset.innerHTML='<label for="pilotAppearance">Style de l’interface</label><select class="input" id="pilotAppearance"><option value="administrative">Administratif — bleu Bootstrap et blanc</option><option value="custom">Couleurs personnalisées</option></select>';
  q('#schoolTheme').closest('.field').before(preset);q('#pilotAppearance').value=appearance;
  try{if(localStorage.getItem('pilot-palette-version')!=='vivid-blue-v2'){appearance='administrative';localStorage.setItem('pilot-appearance-v1',appearance);localStorage.setItem('pilot-palette-version','vivid-blue-v2');}}catch{}
  const usePalette=()=>appearance==='administrative'?palette('#448aff','#2979ff'):palette(DATA.settings?.theme||BRAND.theme,DATA.settings?.business?.invoice_primary||BRAND.business?.invoice_primary);
  const baseBrand=applyBrand;applyBrand=function(){baseBrand();usePalette();};
  q('#pilotAppearance').onchange=e=>{appearance=e.target.value;try{localStorage.setItem('pilot-appearance-v1',appearance);}catch{}usePalette();};
  const previewPalette=()=>{appearance='custom';q('#pilotAppearance').value=appearance;try{localStorage.setItem('pilot-appearance-v1',appearance);}catch{}palette(q('#schoolTheme').value,q('#schoolPrimary').value);};
  q('#schoolTheme').addEventListener('input',previewPalette);q('#schoolPrimary').addEventListener('input',previewPalette);
  q('#schoolPrimary').closest('.field').querySelector('label').textContent='Couleur navigation & entête PDF';
  q('#schoolTheme').closest('.field').querySelector('label').textContent='Couleur des actions & documents';
  const resetColors=document.createElement('button');resetColors.type='button';resetColors.className='btn ghost';resetColors.id='resetPilotColors';
  resetColors.innerHTML='<svg class="pilot-action-icon" viewBox="0 0 24 24" aria-hidden="true"><path d="M4 4v6h6M5.5 15a7 7 0 1 0 .5-7"/></svg><span>Restaurer les couleurs par défaut</span>';
  q('#settingsForm>.actions').prepend(resetColors);
  resetColors.onclick=()=>{appearance='administrative';q('#pilotAppearance').value=appearance;q('#schoolTheme').value='#448aff';q('#schoolPrimary').value='#2979ff';try{localStorage.setItem('pilot-appearance-v1',appearance);localStorage.setItem('pilot-palette-version','vivid-blue-v2');}catch{}palette('#448aff','#2979ff');q('#settingsForm').requestSubmit();};
  function workspace(section,panels,labels){
    const tabs=document.createElement('div');tabs.className='workspace-tabs';tabs.setAttribute('role','tablist');tabs.setAttribute('aria-label',titles[section.id]);
    panels[0].parentElement.before(tabs);
    labels.forEach((label,i)=>{const b=document.createElement('button');b.className='btn ghost';b.textContent=label;b.type='button';b.setAttribute('role','tab');panels[i].id=panels[i].id||section.id+'-panel-'+i;b.setAttribute('aria-controls',panels[i].id);b.onclick=()=>{panels.forEach((p,n)=>p.classList.toggle('pilot-workspace-hidden',n!==i));panels.forEach(p=>p.parentElement.classList.toggle('pilot-workspace-hidden',[...p.parentElement.children].every(x=>x.classList.contains('pilot-workspace-hidden'))));[...tabs.children].forEach((t,n)=>t.setAttribute('aria-selected',String(n===i)));};tabs.append(b);});tabs.firstChild.click();
  }
  workspace(q('#fleet'),qa('#fleet .split>article'),['Véhicules','Moniteurs'].reverse());
  q('#fleet .workspace-tabs').lastChild.click();
  workspace(q('#operations'),qa('#operations .split>article'),['Forfaits','Documents','Entretien véhicules','Absences']);
  const intervention=q('#operations [data-vehicle-event]');intervention.removeAttribute('data-vehicle-event');intervention.dataset.open='vehicle_event';
  function more(actions){
    if(!actions||actions.querySelector('details'))return;const extra=[...actions.children].slice(2);if(!extra.length)return;
    const d=document.createElement('details');d.className='pilot-more';d.innerHTML='<summary class="btn ghost" aria-label="Autres actions">•••</summary><div class="pilot-menu"></div>';extra.forEach(x=>d.lastChild.append(x));actions.append(d);
  }
  function cleanText(root){
    const nodes=document.createTreeWalker(root,NodeFilter.SHOW_TEXT);let n;while(n=nodes.nextNode()){if(n.parentElement.closest('input,textarea,script,style,.studentidentity,.queueidentity,.agenda-info,.profilehero,td'))continue;n.textContent=n.textContent.replace(/[🚘🚗👤📁💳📅📘⏱☎]/gu,'').trimStart();}
  }
  function polish(){
    command();
    qa('.studentcard').forEach(card=>{const id=+card.querySelector('[data-profile]').dataset.profile;more(card.querySelector('.studentactions'));const wa=card.querySelector('[title=WhatsApp]');if(wa)wa.innerHTML='<img class="pilot-custom-icon pilot-action-picture" src="/icons/action-message-ui.png" alt="" aria-hidden="true" draggable="false"><span>WhatsApp</span>';const del=card.querySelector('[data-delete-student]');if(del)del.textContent='Supprimer';cleanText(card);const next=coachNext(id);card.querySelector('.nextslot').title=next?fmt(next.start):'Planifier depuis le dossier';});
    cleanText(q('#studentQuickStats'));
    qa('#todayLessons .event').forEach((el,i)=>{const rows=DATA.lessons.filter(x=>day(x.start)===todayKey()).sort((a,b)=>a.start.localeCompare(b.start));const x=rows[i];if(!x)return;el.classList.toggle('pilot-next',['Planifiée','Confirmée'].includes(x.status)&&new Date(x.start)>=new Date());el.title=vehicleOf(x.vehicle_id);});
    const rail=q('#alerts').parentElement;if(!rail.querySelector('.pilot-rail-link')){const b=document.createElement('button');b.className='btn ghost pilot-rail-link';b.textContent='Ouvrir le centre d’alertes →';b.onclick=()=>q('#nav [data-page=alertsPage]').click();rail.append(b);}
    let activity=q('#pilotActivity');if(!activity){activity=document.createElement('div');activity.id='pilotActivity';activity.className='pilot-activity';q('#dashboard').append(activity);}
    const active=DATA.students.filter(s=>s.status==='Actif');activity.innerHTML='<button data-pilot-go="students"><b>'+active.length+'</b> candidats actifs</button><button data-pilot-go="schedule"><b>'+active.filter(s=>!coachNext(s.id)).length+'</b> sans rendez-vous</button><button data-pilot-go="alertsPage"><b>'+computeAlerts().filter(a=>a.severity==='urgent').length+'</b> priorités urgentes</button>';
    qa('#fleet .vehiclecard h3').forEach(h=>h.textContent=h.textContent.replace('🚘','').trim());
    qa('#fleet .vehiclecard').forEach((card,i)=>{const v=DATA.vehicles[i];card.querySelectorAll('.vehiclefact').forEach((f,n)=>f.dataset.risk=String(n===1?!!v.insurance_expiry&&v.insurance_expiry<todayKey():n===2?!!v.visit_expiry&&v.visit_expiry<todayKey():n===3?v.maintenance_due>0&&v.mileage>=v.maintenance_due:false));});
    const fleet=q('#fleet .split>article:last-child');fleet.querySelector('h2').textContent='Parc véhicules';const add=fleet.querySelector('.sectionhead button');add.removeAttribute('data-edit');add.dataset.open='vehicle';add.textContent='+ Véhicule';
  }
  const baseRender=render;render=function(){baseRender();polish();};
  const baseCoach=applyCoachMode;applyCoachMode=function(){baseCoach();polish();};
  const baseAlerts=renderAlertCenter;renderAlertCenter=function(){baseAlerts();let last='';qa('#alertList .alertrow').forEach(row=>{const s=row.querySelector('.severity').textContent;if(s===last)return;const h=document.createElement('div');h.className='pilot-alert-group';h.textContent=s==='URGENT'?'01 / URGENT':s==='INFO'?'03 / INFORMATION':'02 / À TRAITER';row.before(h);last=s;});};
  const baseOpen=openForm;openForm=function(type,id=0){baseOpen(type,id);q('#modal').dataset.entity=type;const field=q('#formFields [name=instructor_id]');if(field&&DATA.instructors.length>1)field.closest('.field').classList.remove('hidden');const sections={student:{name:'Identité du candidat',package:'Formation & règlement',status:'Suivi'},vehicle:{brand:'Identification du véhicule',insurance_expiry:'Échéances & conformité',mileage:'Suivi kilométrique',purchase_date:'Acquisition'},lesson:{student_id:'Affectation',start:'Créneau',kind:'Séance'},exam:{student_id:'Candidat & épreuve',result:'Résultat'},payment:{student_id:'Encaissement',amount:'Règlement'},invoice:{student_id:'Destinataire',label:'Facturation'}};
    Object.entries(sections[type]||{}).forEach(([name,label])=>{const field=q('#formFields [name="'+name+'"]')?.closest('.field');if(!field||field.classList.contains('hidden'))return;const h=document.createElement('div');h.className='form-section-title';h.textContent=label;field.before(h);});
    q('.dialog').scrollTop=0;
  };
  const baseProfile=openProfile;openProfile=function(id){baseProfile(id);let history=q('#pilotHistory');if(!history){history=document.createElement('section');history.id='pilotHistory';history.className='pilot-history';q('.profilebody').append(history);}
    const rows=[...DATA.lessons.filter(x=>x.student_id==id).map(x=>({date:x.start,type:'Séance',text:x.kind+' · '+x.duration+' min · '+x.status})),...DATA.exams.filter(x=>x.student_id==id).map(x=>({date:x.date,type:'Examen',text:x.type+' · '+x.result})),...DATA.driving_payments.filter(x=>x.student_id==id).map(x=>({date:x.time,type:'Paiement',text:money(x.amount)+' · '+x.method}))].sort((a,b)=>b.date.localeCompare(a.date));
    history.innerHTML='<h3>Historique de formation & règlements</h3>'+rows.slice(0,20).map(x=>'<div class="pilot-history-row"><small>'+esc(fmt(x.date))+'</small><b>'+esc(x.type)+'</b><span>'+esc(x.text)+'</span></div>').join('')+(rows.length?'':'<p class="empty">Aucune séance ni règlement enregistré.</p>');
  };
  document.addEventListener('click',e=>{const b=e.target.closest('[data-pilot-go]');if(b)q('#nav [data-page="'+b.dataset.pilotGo+'"]').click();qa('.pilot-more[open]').forEach(d=>{if(!d.contains(e.target))d.open=false;});});
  command();
})();

/* Single-instructor workflow, candidate finances and presentation preferences. */
(() => {
 const q=s=>document.querySelector(s),qa=s=>[...document.querySelectorAll(s)];
 const preference=(k,fallback)=>{try{return localStorage.getItem(k)||fallback}catch{return fallback}};
 const savePreference=(k,v)=>{try{localStorage.setItem(k,v)}catch{}};
 // The main workflow follows the order in which an instructor works.
 const nav=q('#nav'),secondary=document.createElement('div');
 secondary.className='workflow-secondary';secondary.innerHTML='<div class="workflow-secondary-title">Gestion & outils</div>';
 qa('#nav .nav-group').forEach(n=>n.remove());
 ['dashboard','students','schedule','finance','exams'].forEach(p=>nav.append(q('#nav [data-page="'+p+'"]')));
 ['alertsPage','fleet','operations','reports','settings'].forEach(p=>secondary.append(q('#nav [data-page="'+p+'"]')));
 nav.append(secondary);
 const archived=document.createElement('option');archived.value='Archivé';archived.textContent='Archivés';q('#studentStatus').append(archived);
 q('#studentStatus').value='Actif';
 fields.student.find(f=>f[0]==='status')[2]='select:Actif|Suspendu|Terminé|Archivé';
 fields.student.splice(3,0,['gender','Genre','select:Homme|Femme']);
 fields.package_sale.push(['mode','Type d’affectation','select:initial|additional']);
 fields.payment.push(['invoice_id','Facture liée (facultatif)','number']);
 // Replace destructive candidate actions at capture phase, preserving history.
 document.addEventListener('click',async e=>{
  const b=e.target.closest('[data-delete-student]');if(!b)return;
  e.preventDefault();e.stopImmediatePropagation();const id=+b.dataset.deleteStudent,s=DATA.students.find(s=>s.id===id);
  if(!s||!confirm('Archiver '+s.name+' ? Ses documents, factures et paiements sont conservés. Ses séances à venir seront annulées.'))return;
  try{await api('/api/driving/student?id='+id,{method:'DELETE'});toast('Dossier archivé, historique conservé');await reload();}catch(err){toast(err.message,true);}
 },true);
 function creditUI(){
  qa('.studentcard').forEach(card=>{
   const id=+card.querySelector('[data-profile]').dataset.profile,s=DATA.students.find(s=>s.id===id);if(!s)return;
   const net=(+s.package||0)-paid(id),value=card.querySelector('.studentbalance strong'),label=value?.previousElementSibling;
   if(value){value.textContent=money(Math.abs(net));value.classList.toggle('due',net>.0005);if(label)label.textContent=net<-.0005?'Crédit disponible':'Reste à payer';}
   const archive=card.querySelector('[data-delete-student]');if(archive){archive.textContent='Archiver';archive.title='Archiver le dossier';archive.disabled=s.status==='Archivé';}
   if(s.status==='Archivé'&&!card.querySelector('[data-reactivate]')){const button=document.createElement('button');button.className='btn ghost';button.dataset.reactivate=String(id);button.textContent='Réactiver';card.querySelector('.studentactions').append(button);}
  });
 }
 const oldRender=render;render=function(){oldRender();creditUI();};
 const oldCoach=applyCoachMode;applyCoachMode=function(){oldCoach();creditUI();};
 const oldForm=openForm;openForm=function(type,id=0){
  oldForm(type,id);const form=q('#entityForm'),f=n=>form.elements.namedItem(n);
  if(type==='student'){
   f('phone').required=true;
   if(!id){f('driving_target').value=0;f('theory_target').value=0;f('package').value=0;}
   const hint=document.createElement('p');hint.className='field-hint span2';hint.textContent='Indiquez les heures et le prix convenus, ou affectez un forfait initial depuis le dossier. Aucun ajout automatique de 20 heures.';q('#formFields').append(hint);
  }
  if(type==='package_sale'){
   const mode=f('mode');mode.options[0].textContent='Forfait initial — remplace les objectifs saisis';mode.options[1].textContent='Heures supplémentaires — s’ajoutent au forfait';
   const update=()=>{const has=DATA.package_sales.some(s=>s.student_id===+f('student_id').value);mode.options[0].disabled=has;mode.value=has?'additional':'initial';};f('student_id').addEventListener('change',update);update();
  }
  if(type==='payment'){
   const old=f('invoice_id'),select=document.createElement('select');select.className='input';select.name='invoice_id';select.id=old.id;old.replaceWith(select);
   const initial=DATA.driving_payments.find(p=>p.id===id)?.invoice_id||0;
   const update=()=>{
    const chosen=+select.value||initial,student=+f('student_id').value;
    select.innerHTML='<option value="">Sans facture / avance sur forfait</option>'+DATA.driving_invoices.filter(x=>x.student_id===student&&x.status!=='Annulée').map(x=>'<option value="'+x.id+'">#'+x.id+' · '+esc(x.label)+' · '+money(x.amount)+'</option>').join('');
    select.value=String(chosen);if(select.selectedIndex<0)select.value='';
    let info=q('#paymentBalance');if(!info){info=document.createElement('p');info.id='paymentBalance';info.className='field-hint span2';q('#formFields').append(info);}
    const s=DATA.students.find(s=>s.id===student),net=s?(+s.package||0)-paid(student):0;
    info.textContent=s?(net<0?'Crédit disponible : '+money(-net):'Reste du forfait : '+money(net))+' · Les paiements liés règlent la facture sans doubler les recettes.':'Choisissez un candidat.';
   };
   f('student_id').addEventListener('change',update);update();
   if(!id)f('time').value=localISO(new Date())+'T'+new Date().toTimeString().slice(0,5);
   // Legacy shortcuts assign the candidate after opening the form.
   setTimeout(update,60);
  }
  if(type==='invoice'){
   const status=f('status'),existing=DATA.driving_invoices.find(x=>x.id===id);
   status.innerHTML='<option value="À payer">Suivi automatique des paiements</option><option value="Annulée">Annulée</option>';status.value=existing?.status==='Annulée'?'Annulée':'À payer';
   const hint=document.createElement('p');hint.className='field-hint span2';hint.textContent='Payée / partiellement payée se calcule depuis les paiements liés. Cette facture décrit le forfait : elle n’ajoute pas une deuxième dette. Pour une prestation supplémentaire, augmentez d’abord le forfait.';q('#formFields').append(hint);
  }
  if(type==='lesson'){
   const field=f('instructor_id');if(field)field.closest('.field').classList.add('hidden');
   const note=document.createElement('p');note.className='field-hint span2';
   const update=()=>{const s=DATA.students.find(s=>s.id===+f('student_id').value);note.textContent=s?'Conduite : '+Math.max(0,(+s.driving_target||0)-coachDone(s.id)-coachPlanned(s.id)).toFixed(1)+' h encore à planifier. Le système contrôle le forfait et les échéances du véhicule.':'Choisissez un candidat pour voir ses heures.';};
   q('#formFields').append(note);f('student_id').addEventListener('change',update);update();setTimeout(update,60);
  }
 };
 const oldProfile=openProfile;openProfile=function(id){
  oldProfile(id);const s=DATA.students.find(s=>s.id==id);if(!s)return;
  q('#profileTabs')?.remove();q('#profileOverview')?.remove();
  const body=q('.profilebody'),overview=document.createElement('section');overview.id='profileOverview';
  const done=coachDone(+id),planned=coachPlanned(+id),net=(+s.package||0)-paid(id),next=coachNext(+id);
  overview.innerHTML='<h3>Situation du candidat</h3><div class="profilestats">'+[[done.toFixed(1)+' h','Réalisées'],[Math.max(0,(+s.driving_target||0)-done).toFixed(1)+' h','Restantes'],[planned.toFixed(1)+' h','Réservées'],[money(Math.abs(net)),net<0?'Crédit disponible':'Reste à payer']].map(([v,l])=>'<div class="profilestat"><b>'+esc(v)+'</b><small>'+l+'</small></div>').join('')+'</div><div class="nextslot">'+(next?'Prochain rendez-vous : '+esc(fmt(next.start)):'Aucun prochain rendez-vous')+'</div><p class="field-hint">Forfait : '+money(s.package)+' · Total payé : '+money(paid(id))+'</p><div class="actions"><button class="btn primary" data-profile-plan="'+id+'">Planifier une séance</button><button class="btn ghost" data-profile-payment="'+id+'">Enregistrer un paiement</button><button class="btn ghost" data-assign-package="'+id+'">Affecter un forfait</button><button class="btn ghost" data-profile-edit="'+id+'">Modifier</button></div>';
  const tabs=document.createElement('div');tabs.id='profileTabs';tabs.className='workspace-tabs';tabs.setAttribute('role','tablist');
  body.prepend(tabs,overview);
  const docs=q('#profileDocs'),history=q('#pilotHistory'),progress=q('#profileProgress'),stats=q('#profileStats'),docHead=body.querySelector('.panelhead');
  progress?.remove();
  const panels=[[overview],[docs,docHead],[history]];
  ['Situation & heures','Pièces du dossier','Historique'].forEach((label,i)=>{const b=document.createElement('button');b.type='button';b.className='btn ghost';b.textContent=label;b.setAttribute('role','tab');b.onclick=()=>{panels.forEach((nodes,n)=>nodes.filter(Boolean).forEach(p=>p.classList.toggle('workflow-hidden',n!==i)));stats.classList.add('workflow-hidden');[...tabs.children].forEach((t,n)=>t.setAttribute('aria-selected',String(n===i)));};tabs.append(b);});tabs.firstChild.click();
 };
 document.addEventListener('click',async e=>{
  const restore=e.target.closest('[data-reactivate]');
  if(restore){const student=DATA.students.find(s=>s.id===+restore.dataset.reactivate);if(!student)return;restore.disabled=true;try{await api('/api/driving/student',{method:'POST',body:JSON.stringify({...student,status:'Actif'})});toast('Candidat réactivé. Les séances annulées restent annulées.');q('#studentStatus').value='Actif';await reload();}catch(err){toast(err.message,true);}finally{restore.disabled=false;}return;}
  const b=e.target.closest('[data-assign-package]');if(!b)return;openForm('package_sale');q('#formFields [name=student_id]').value=b.dataset.assignPackage;q('#formFields [name=student_id]').dispatchEvent(new Event('change'));
 });
 // Compact, persisted language and display controls. Data and option values remain unchanged.
 const controls=document.createElement('div');controls.className='display-controls';
 controls.innerHTML='<button type="button" class="btn ghost" id="pilotLanguage">تونسي</button><button type="button" class="btn ghost" id="pilotDark">Mode sombre</button>';
 q('.top .actions').prepend(controls);
 let lang=preference('pilot-language','fr'),dark=preference('pilot-dark','false')==='true';
 const dictionary={
 'Réactiver':'رجّعو ناشط','Tableau de bord':'نهاري','Candidats':'تلامذتي','Planning':'البرنامج','Paiements':'الخلاص','Examens':'الامتحانات','Gestion & outils':'التصرّف والأدوات','Alertes':'التنبيهات','Équipe & véhicules':'المونيتور والكرهبة','Dossiers & opérations':'الدوسيات والخدمات','Rapports':'التقارير','Administration':'الإعدادات','Déconnexion':'خروج',
 'Votre activité, aujourd’hui':'خدمتك اليوم','Dossiers, progression et prochaines étapes':'الدوسيات، التقدّم والخطوة الجاية','Séances, disponibilités et ordre de passage':'السوايع، الأوقات الفارغة وترتيب التلامذة','Encaissements, dépenses et factures':'الخلاص، المصاريف والفواتير','Situation du candidat':'وضعية التلميذ','Situation & heures':'الوضعية والسوايع','Pièces du dossier':'أوراق الدوسي','Historique':'التاريخ',
 'Planifier une séance':'برمج ساعة','Enregistrer un paiement':'سجّل خلاص','Affecter un forfait':'اختار فورفي','Modifier':'بدّل','Archiver':'حطّ في الأرشيف','Annuler':'إلغاء','Enregistrer':'سجّل','Ajouter':'زيد','Dossier':'الدوسي','Payer':'خلّص','Supprimer':'فسّخ','Fermer le dossier':'سكّر الدوسي','Fermer le formulaire':'سكّر الفورميلار',
 'Semaine':'جمعة','Mois':'شهر','Aujourd’hui':'اليوم','Réalisées':'قراهم','Restantes':'مازالوا','Réservées':'مبرمجين','Reste à payer':'باقي يخلّص','Crédit disponible':'عندو تسبقة','RÉALISÉES':'قراهم','RESTANTES':'مازالوا','PLANIFIÉES':'مبرمجين','À EFFECTUER':'مازالوا','ÉLÈVES':'تلامذة','Terminée':'كملت','Planifiée':'مبرمجة','Confirmée':'متأكّدة','Absence':'غايب','Annulée':'تلغات','Actif':'ناشط','Suspendu':'موقوف','Terminé':'كمّل','Archivé':'في الأرشيف','Archivés':'الأرشيف','Tous':'الكل','Complet':'كامل','Incomplet':'ناقص','Payée':'خالصة','Partiellement payée':'خالص منها شطر','À payer':'مازالت ما خلصتش','Réussi':'ناجح','Échec':'طاح','Prévu':'مبرمج',
 'Nom complet':'الإسم واللقب','Téléphone':'التليفون','CIN':'بطاقة التعريف','Date de naissance':'تاريخ الولادة','Adresse':'العنوان','Catégorie permis':'صنف الرخصة','Date inscription':'تاريخ التسجيل','Forfait total (DT)':'سوم الفورفي (د.ت)','État':'الحالة','Notes':'ملاحظات','Objectif heures conduite':'سوايع السياقة المتفاهم عليهم','Objectif heures code':'سوايع الكود','Date et heure':'النهار والوقت','Durée (minutes)':'المدّة بالدقايق','Type':'النوع','Conduite':'سياقة','Code':'كود','Créneau':'كريّنو','Candidat':'التلميذ','Moniteur':'المونيتور','Véhicule':'الكرهبة',
 'Montant':'المبلغ','Mode':'طريقة الخلاص','Espèces':'كاش','Carte':'كارطة','Virement':'تحويل','Chèque':'شيك','Facture liée (facultatif)':'الفاتورة المرتبطة (اختياري)','Sans facture / avance sur forfait':'بلا فاتورة / تسبقة على الفورفي','Suivi automatique des paiements':'تتبّع الخلاص أوتوماتيك','Bon RDV':'وصل موعد','Dossiers candidats':'دوسيات التلامذة',
 'SITUATION DU JOUR':'وضعية اليوم','La journée en mouvement':'نهارك منظّم','Candidats actifs':'التلامذة الناشطين','Séances du jour':'سوايع اليوم','Recettes du mois':'مداخيل الشهر','Soldes à recevoir':'فلوس مازالت ما خلصتش','Planning du jour':'برنامج اليوم','Alertes à traiter':'حاجات تستحق تدخّل','Planning libre pour aujourd’hui':'اليوم ما فماش سوايع','Aucune prochaine séance planifiée':'ما فماش ساعة جاية مبرمجة','Aucun prochain rendez-vous':'ما فماش موعد جاي',
 'Paramètres':'الإعدادات','Identité & logo':'المعلومات واللوغو','Horaires & documents':'الأوقات والوثائق','Sauvegarde & historique':'نسخة احتياطية والتاريخ','Utilisateurs':'المستعملين','Sauvegarde':'نسخة احتياطية','Télécharger sauvegarde':'هبّط نسخة احتياطية','Restaurer':'رجّع النسخة','Style de l’interface':'ستايل الواجهة','Couleurs personnalisées':'ألوان على ذوقك','Chaleureux — ivoire, brun et cuivre':'دافي — بيج وبني ونحاسي',
 'Paiements & factures':'الخلاص والفواتير','Finances':'الحسابات','Derniers paiements':'آخر الخلاصات','Dépenses':'المصاريف','Factures clients':'فواتير التلامذة','Total encaissé':'مجموع المقبوض','Ce mois':'الشهر هذا','Dépenses du mois':'مصاريف الشهر','Résultat du mois':'نتيجة الشهر','Date':'التاريخ','Centre':'المركز','Résultat':'النتيجة','Bon réception':'وصل خلاص','Aucun paiement':'ما فماش خلاصات','Aucune dépense':'ما فماش مصاريف','Aucune facture':'ما فماش فواتير',
 'Forfait initial — remplace les objectifs saisis':'الفورفي الأول — يعوّض السوايع والسوم المكتوبين','Heures supplémentaires — s’ajoutent au forfait':'سوايع زيادة — يتزادوا على الفورفي','Type d’affectation':'نوع الفورفي','Formation & règlement':'التكوين والخلاص','Identité du candidat':'معلومات التلميذ','Suivi':'المتابعة','Encaissement':'قبض الفلوس','Règlement':'الخلاص',
 'Trouver le prochain créneau libre':'لوّج على أقرب وقت فارغ','Programmer plusieurs séances':'برمج برشا سوايع','Une seule séance':'ساعة واحدة','Même jour, chaque semaine':'نفس النهار كل جمعة','Plusieurs jours de cette semaine':'برشا نهارات في الجمعة هاذي','Plusieurs jours de ce mois':'برشا نهارات في الشهر هذا','Jusqu’à une date choisie':'لين نهار تختارو','Période':'المدّة','Dernier jour':'آخر نهار','Ordre automatique':'ترتيب أوتوماتيك','Planifier':'برمج','Bilan de la journée':'حصيلة النهار','Ordre des élèves à planifier':'شكون قبل شكون',
 'Identité de l’auto-école':'معلومات المونيتور','Parc véhicules':'الكراهب','Véhicules':'الكراهب','Moniteurs':'المونيتورات','Disponible':'موجودة','Entretien':'صيانة','Hors service':'ما تخدمش','Fiche complète':'المعلومات الكل','Connexion':'دخول','Connexion utilisateur':'دخول للحساب','Utilisateur':'المستعمل','Mot de passe':'كلمة السر','Se connecter':'ادخل','Confirmer':'عاود أكّد','Activer le logiciel':'فعّل البرنامج','Clé produit':'مفتاح التفعيل','Code machine':'كود الجهاز',
 'Ouvrir le centre d’alertes →':'شوف التنبيهات الكل ←','Nouveau candidat':'تلميذ جديد','Nouvelle séance':'ساعة جديدة','Séance':'ساعة','Paiement':'خلاص','Dépense':'مصروف','Facture client':'فاتورة تلميذ','Nouvel examen':'امتحان جديد','Document':'وثيقة','Ajouter un fichier':'زيد ملف','Remplacer le fichier':'بدّل الملف','Enregistrement réussi':'تسجّل بنجاح','Dossier archivé, historique conservé':'الدوسي تحطّ في الأرشيف والتاريخ محفوظ'
 };
 Object.assign(dictionary,{
 'Dossier candidat':'دوسي التلميذ','Encaisser un paiement':'اقبض خلاص','Enregistrer une dépense':'سجّل مصروف','Forfait de formation':'فورفي تكوين','Ajouter un document':'زيد وثيقة','Intervention véhicule':'خدمة للكرهبة','Absence moniteur':'غياب المونيتور','Affecter un forfait':'اختار فورفي','Nom du forfait':'إسم الفورفي','Heures de code':'سوايع الكود','Heures de conduite':'سوايع السياقة','Prix (DT)':'السوم بالدينار','Catégorie':'الصنف','Motif':'السبب','Montant total (DT)':'المبلغ الكل بالدينار','Désignation / service':'الخدمة','Référence / note':'مرجع / ملاحظة','Fin validité certificat médical':'آخر صلاحية للشهادة الطبية','État du dossier':'وضعية الدوسي',
 'Type de document':'نوع الوثيقة','Date d’expiration':'آخر صلاحية','Fichier PDF / image':'ملف PDF ولا تصويرة','Photo':'تصويرة','Certificat médical':'شهادة طبية','Contrat':'عقد','Justificatif':'وثيقة إثبات','Autre':'حاجة أخرى','Voir':'شوف','Télécharger':'هبّط','Ouvrir':'حلّ','Forfaits':'الفورفيات','Documents':'الوثائق','Entretien véhicules':'صيانة الكراهب','Absences':'الغيابات','Marque':'الماركة','Modèle':'الموديل','Matricule':'النمرة','Année':'العام','Couleur':'اللون','Carburant':'الوقود','Boîte':'الفيتاس','Manuelle':'مانيال','Automatique':'أوتوماتيك','Fin assurance':'آخر نهار تأمين','Visite technique':'الزيارة الفنية','Kilométrage actuel':'الكيلومتراج توا','Prochain entretien (km)':'الصيانة الجاية (كم)','Compagnie assurance':'شركة التأمين','N° police assurance':'رقم التأمين','Notes véhicule':'ملاحظات على الكرهبة','Coût (DT)':'التكلفة بالدينار','Prestataire':'شكون عمل الخدمة','Du':'من','Au':'لين','Suivi des examens':'متابعة الامتحانات','Épreuve':'الإمتحان','Résultats examens':'نتائج الامتحانات','Recettes':'المداخيل','Résultat net':'الصافي','Heures réalisées':'السوايع المقروءة','Actualiser':'جدّد','Imprimer':'اطبع','Exporter Excel/CSV':'هبّط Excel/CSV','Rattrapage après un échec':'سوايع تدارك بعد الطيحان','Décision':'القرار','Ajouter des heures de rattrapage':'زيد سوايع تدارك','Reprogrammer directement l’examen':'عاود برمج الامتحان','Enregistrer seulement':'سجّل برك','Heures supplémentaires':'سوايع زيادة','Prix supplémentaire (DT)':'السوم الزايد بالدينار','Nouvelle date d’examen':'النهار الجديد للامتحان','Aucun candidat trouvé':'ما لقيناش تلميذ','Aucun examen':'ما فماش امتحانات','Aucun document':'ما فماش وثائق','Aucun véhicule':'ما فماش كرهبة','Finances':'الحسابات','Centre d’alertes':'التنبيهات','Tout est en ordre : aucune alerte.':'الأمور واضحة، ما فماش تنبيهات.','Paramètres du moniteur':'إعدادات المونيتور','Suivi des heures de conduite':'متابعة سوايع السياقة','Historique de formation & règlements':'تاريخ السوايع والخلاص','Aucune séance ni règlement enregistré.':'ما فماش سوايع ولا خلاص متسجّل.',
 'Nom':'الإسم','Genre':'الجنس','Homme':'راجل','Femme':'مرأة','Supprimer définitivement':'فسّخ نهائي','Supprimer la séance':'فسّخ الساعة','Supprimer définitivement le candidat':'فسّخ التلميذ نهائي','Email':'الإيميل','Matricule fiscal':'المعرّف الجبائي','Couleur des actions & documents':'لون الأزرار والوثائق','Couleur navigation & entête PDF':'لون القائمة وراس الوثيقة','Restaurer les couleurs par défaut':'رجّع الألوان الأصلية','Logo personnalisé (PNG/JPG, maximum 2 Mo)':'لوغو على ذوقك (PNG/JPG، أقصى حجم 2Mo)','Horaires, rappels et textes des documents':'الأوقات، التذكير ونصوص الوثائق','Durée séance par défaut':'مدّة الساعة العادية','Ouverture':'بداية الخدمة','Fermeture':'نهاية الخدمة','Note facture':'ملاحظة الفاتورة','Note reçu de paiement':'ملاحظة وصل الخلاص','Note bon de rendez-vous':'ملاحظة وصل الموعد','Introduction certificat':'مقدّمة الشهادة','Conclusion certificat':'خاتمة الشهادة'
 });
 const originals=new WeakMap();
 function translated(text){
  const raw=text.trim(),prefix=raw.match(/^[＋+✎✓]\s*/)?.[0]||'',key=prefix?raw.slice(prefix.length):raw;
  if(dictionary[raw])return text.replace(raw,dictionary[raw]);
  if(dictionary[key])return text.replace(raw,prefix+dictionary[key]);
  return text;
 }
 let pending=false;
 function localize(){
  pending=false;
  const walker=document.createTreeWalker(document.body,NodeFilter.SHOW_TEXT);let node;
  while(node=walker.nextNode()){
   if(!node.parentElement||node.parentElement.closest('script,style,textarea,[data-name],#profileName,#profileMeta,.studentidentity,.queueidentity,.agenda-info,.licensecode,#userChip,#pilotLanguage,#pilotDark,select[name=student_id],select[name=instructor_id],select[name=vehicle_id],select[name=package_id],select[name=invoice_id]'))continue;
   const saved=originals.get(node),source=saved&&node.nodeValue===saved.output?saved.source:node.nodeValue;
   const parent=node.parentElement;if(parent.tagName==='OPTION'&&!parent.hasAttribute('value'))parent.value=source;
   const output=lang==='tn'?translated(source):source;originals.set(node,{source,output});if(node.nodeValue!==output)node.nodeValue=output;
  }
  const languageLabel=lang==='tn'?'Français':'تونسي',darkLabel=dark?(lang==='tn'?'الوضع الفاتح':'Mode clair'):(lang==='tn'?'الوضع الليلي':'Mode sombre');
  if(q('#pilotLanguage').textContent!==languageLabel)q('#pilotLanguage').textContent=languageLabel;
  if(q('#pilotDark').textContent!==darkLabel)q('#pilotDark').textContent=darkLabel;
  document.documentElement.lang=lang==='tn'?'ar-TN':'fr';document.documentElement.dataset.pilotLanguage=lang;
  qa('input[placeholder]').forEach(input=>{if(!input.dataset.frPlaceholder)input.dataset.frPlaceholder=input.placeholder;input.placeholder=lang==='tn'&&/Rechercher|Nom, CIN/.test(input.dataset.frPlaceholder)?'لوّج بالإسم، بطاقة التعريف ولا التليفون…':input.dataset.frPlaceholder;});
 }
 const schedule=()=>{if(pending)return;pending=true;requestAnimationFrame(localize);};
 new MutationObserver(schedule).observe(document.body,{childList:true,subtree:true,characterData:true});
 q('#pilotLanguage').onclick=()=>{lang=lang==='tn'?'fr':'tn';savePreference('pilot-language',lang);localize();};
 const applyDark=()=>{document.documentElement.dataset.pilotDark=String(dark);q('#pilotDark').setAttribute('aria-pressed',String(dark));schedule();};
 q('#pilotDark').onclick=()=>{dark=!dark;savePreference('pilot-dark',String(dark));applyDark();};
 applyDark();schedule();
})();

/* Professional interaction pass: clear actions, vehicle costs and maintenance alerts. */
(() => {
 const q=s=>document.querySelector(s),qa=s=>[...document.querySelectorAll(s)];
 document.addEventListener('click',e=>{const b=e.target.closest('[data-replace-doc]');if(!b)return;setTimeout(()=>{const form=q('#entityForm');if(!form||q('#entityType').value!=='document')return;let hidden=form.querySelector('[name=replace_id]');if(!hidden){hidden=document.createElement('input');hidden.type='hidden';hidden.name='replace_id';form.append(hidden);}hidden.value=b.dataset.replaceDoc;},0);},true);
 const svg=path=>'<svg class="pilot-action-icon" viewBox="0 0 24 24" aria-hidden="true"><path d="'+path+'"/></svg>';
 const icons={check:'M5 12l4 4L19 6',absent:'M6 6l12 12M18 6L6 18',cancel:'M5 5l14 14M19 5L5 19',trash:'M4 7h16M9 7V4h6v3M7 7l1 13h8l1-13M10 11v5M14 11v5',edit:'M4 20h4L19 9l-4-4L4 16v4M13 7l4 4',pdf:'M6 3h9l4 4v14H6zM14 3v5h5M9 13h6M9 17h6',car:'M3 14l2-6h14l2 6v5h-3v-2H6v2H3zM6 14h.01M18 14h.01',plus:'M12 5v14M5 12h14',dots:'M5 12h.01M12 12h.01M19 12h.01'};
 const avatarSets={Homme:['/icons/avatar-man-1.png','/icons/avatar-man-2.png','/icons/avatar-man-3.png','/icons/avatar-man-4.png','/icons/avatar-man-5.png'],Femme:['/icons/avatar-woman-1.png','/icons/avatar-woman-2.png']};
 const avatarFor=student=>{const gender=student?.gender==='Femme'?'Femme':'Homme',pool=avatarSets[gender],seed=String(student?.id||student?.name||'1').split('').reduce((n,c)=>(n*31+c.charCodeAt(0))>>>0,7);return pool[seed%pool.length];};
 const paintAvatar=(element,student)=>{if(!element||!student)return;const src=avatarFor(student)+'?v=3';element.classList.add('has-default-avatar');element.style.removeProperty('--student-avatar');element.innerHTML='<img src="'+src+'" alt="Photo de '+esc(student.name)+'" draggable="false">';};
 const decorateStudentAvatars=()=>qa('#studentsGrid .studentcard').forEach(card=>{const id=+(card.querySelector('[data-profile]')?.dataset.profile||0),student=(DATA.students||[]).find(x=>x.id===id);paintAvatar(card.querySelector('.studentavatar'),student);});
 let permanentDelete=q('#profileDeletePermanent');if(!permanentDelete){permanentDelete=document.createElement('button');permanentDelete.id='profileDeletePermanent';permanentDelete.type='button';permanentDelete.className='btn profile-delete';permanentDelete.innerHTML=svg(icons.trash)+'<span>Supprimer définitivement le candidat</span>';q('#profileClose').before(permanentDelete);}
 const originalOpenProfileAvatar=openProfile;openProfile=function(id){originalOpenProfileAvatar(id);const student=DATA.students.find(x=>x.id===+id);paintAvatar(q('#profileAvatar'),student);permanentDelete.dataset.permanentStudent=String(id);};
 const iconize=(el,kind,label)=>{if(!el)return;el.innerHTML=svg(icons[kind])+ '<span>'+label+'</span>';};
 const isVehicleExpense=x=>/^Véhicule ·/.test(x.category||'')||['Carburant','Entretien','Assurance'].includes(x.category);
 const vehicleExpenses=()=>(DATA.expenses||[]).filter(isVehicleExpense);
 const collapse=(actions,keep=1)=>{
  if(!actions||actions.querySelector(':scope > details.pilot-more'))return;
  const extra=[...actions.children].slice(keep);if(!extra.length)return;
  const d=document.createElement('details');d.className='pilot-more';d.innerHTML='<summary class="btn ghost" aria-label="Autres actions">'+svg(icons.dots)+'<span>Autres</span></summary><div class="pilot-menu"></div>';
  extra.forEach(x=>d.lastElementChild.append(x));actions.append(d);
 };
 const expand=(actions)=>{const d=actions?.querySelector(':scope > details.pilot-more'),menu=d?.querySelector('.pilot-menu');if(!d||!menu)return;[...menu.children].forEach(button=>actions.insertBefore(button,d));d.remove();};
 const decorateAgenda=()=>{
  const rows=qa('#dayAgenda .agenda-item'),now=Date.now();let next=null;
  rows.forEach(row=>{
   const marker=row.querySelector('[data-lesson-status]'),id=+(marker?.dataset.lessonStatus.split(':')[0]||0),lesson=DATA.lessons.find(x=>x.id===id);
   if(!lesson)return;
   const start=new Date(lesson.start).getTime(),end=start+(+lesson.duration||60)*60000;
   row.dataset.state=lesson.status;row.classList.toggle('pilot-current',start<=now&&now<end&&!/Terminée|Annulée|Absence/.test(lesson.status));
   if(start>now&&!/Terminée|Annulée|Absence/.test(lesson.status)&&(!next||start<next.start))next={row,start};
   const info=row.querySelector('.agenda-info small'),student=DATA.students.find(x=>x.id===lesson.student_id),remaining=student?Math.max(0,(+student.driving_target||0)-coachDone(student.id)):0;
   if(info)info.innerHTML='<span class="agenda-vehicle"><b>Véhicule</b> '+esc(vehicleOf(lesson.vehicle_id)||'Non affecté')+'</span><span class="agenda-hours"><b>'+remaining.toFixed(1)+' h</b> restantes</span>';
   const type=row.querySelector('.typeicon');if(type){type.innerHTML=svg(lesson.kind==='Conduite'?icons.car:lesson.kind==='Code'?'M5 4h14v16H5zM8 8h8M8 12h8M8 16h5':'M12 5v14M5 12h14');type.setAttribute('aria-label',lesson.kind);}
   const actions=row.querySelector('.agenda-actions'),done=actions?.querySelector('[data-lesson-status$=":Terminée"]'),absent=actions?.querySelector('[data-lesson-status$=":Absence"]');
   if(done){done.className='btn action-complete';iconize(done,'check','Terminée');}
   if(absent){absent.className='btn action-absent';iconize(absent,'absent','Absence');}
   const edit=actions?.querySelector('[data-edit]');if(edit){edit.className='btn action-edit';iconize(edit,'edit','Modifier');}
   const pdf=actions?.querySelector('a[href*="appointment.pdf"]');if(pdf){pdf.className='btn action-rdv';iconize(pdf,'pdf','Bon RDV');}
   const whatsapp=actions?.querySelector('a[title="Rappel WhatsApp"]');let remove=actions?.querySelector('[data-lesson-delete]');
   if(!remove&&actions){remove=document.createElement('button');remove.type='button';remove.className='btn action-delete';remove.dataset.lessonDelete=id;remove.title='Supprimer uniquement cette séance';iconize(remove,'trash','Supprimer');if(whatsapp)whatsapp.replaceWith(remove);else actions.append(remove);}
   if(actions&&!/Terminée|Annulée/.test(lesson.status)&&!actions.querySelector('[data-lesson-cancel]')){const b=document.createElement('button');b.type='button';b.className='btn action-cancel';b.dataset.lessonCancel=id;iconize(b,'cancel','Annuler');actions.append(b);}
   if(actions){expand(actions);const primary=done||edit;if(primary)actions.prepend(primary);}
  });
  if(next&&!next.row.classList.contains('pilot-current'))next.row.classList.add('pilot-next-session');
 };
 const emptyAction=(selector,text,label,type)=>{const el=q(selector);if(!el||el.querySelector('.empty-action'))return;el.innerHTML='<strong>'+text+'</strong><span>Utilisez le bouton ci-dessous pour commencer.</span><button type="button" class="btn primary empty-action" data-open="'+type+'">'+svg(icons.plus)+'<span>'+label+'</span></button>';};
 const enhanceFinance=()=>{
  const rows=[...vehicleExpenses()].reverse();q('#expensesTable').innerHTML=rows.map(x=>'<tr><td>'+fmt(x.time)+'</td><td>'+esc(x.category.replace(/^Véhicule ·\s*/,''))+'</td><td>'+esc(x.label)+'</td><td class="balance">'+money(x.amount)+'</td></tr>').join('')||'<tr><td colspan="4" class="empty"><strong>Aucun frais véhicule enregistré</strong><span>Ajoutez l’essence, une vidange ou une réparation depuis la fiche véhicule.</span><button class="btn primary empty-action" data-open="vehicle_event">'+svg(icons.car)+'<span>Ajouter un frais véhicule</span></button></td></tr>';
  const month=todayKey().slice(0,7),cost=vehicleExpenses().filter(x=>day(x.time).startsWith(month)).reduce((a,x)=>a+(+x.amount||0),0),paidMonth=(DATA.driving_payments||[]).filter(x=>day(x.time).startsWith(month)).reduce((a,x)=>a+(+x.amount||0),0),cards=qa('#financeCards .financecard');
  if(cards[2]){cards[2].querySelector('span').textContent='Frais véhicules du mois';cards[2].querySelector('b').textContent=money(cost);}if(cards[3])cards[3].querySelector('b').textContent=money(paidMonth-cost);
 };
 const enhanceReports=()=>{
  const from=q('#reportFrom')?.value||'',to=q('#reportTo')?.value||'9999-12-31',costs=vehicleExpenses().filter(x=>day(x.time)>=from&&day(x.time)<=to),cost=costs.reduce((a,x)=>a+(+x.amount||0),0),income=(DATA.driving_payments||[]).filter(x=>day(x.time)>=from&&day(x.time)<=to).reduce((a,x)=>a+(+x.amount||0),0),cards=qa('#reportCards .financecard');
  if(cards[1]){cards[1].querySelector('span').textContent='Frais véhicules';cards[1].querySelector('b').textContent=money(cost);}if(cards[2])cards[2].querySelector('b').textContent=money(income-cost);
  let box=q('#vehicleCostBreakdown');if(!box){box=document.createElement('section');box.id='vehicleCostBreakdown';box.className='card vehicle-cost-summary';q('#reportCards').after(box);}
  const groups={};costs.forEach(x=>{const k=(x.category||'Autre').replace(/^Véhicule ·\s*/,'');groups[k]=(groups[k]||0)+(+x.amount||0);});
  box.innerHTML='<div><strong>Détail des frais véhicules</strong><span>La période choisie pilote automatiquement ce bilan.</span></div><div class="cost-pills">'+(Object.entries(groups).map(([k,v])=>'<span><b>'+esc(k)+'</b>'+money(v)+'</span>').join('')||'<span>Aucun frais véhicule sur cette période</span>')+'</div>';
 };
  const enhance=()=>{
  decorateAgenda();enhanceFinance();enhanceReports();decorateStudentAvatars();
  qa('.studentcard').forEach(card=>{card.tabIndex=0;card.setAttribute('role','group');});qa('.daypick').forEach(b=>b.setAttribute('aria-label',b.innerText.replace(/\s+/g,' ')));
  const students=q('#studentsGrid .empty');if(students)emptyAction('#studentsGrid .empty','Aucun candidat dans cette vue','Ajouter un candidat','student');
  const vehicles=q('#vehiclesGrid .empty');if(vehicles)emptyAction('#vehiclesGrid .empty','Aucun véhicule enregistré','Ajouter un véhicule','vehicle');
  qa('a[href$=".pdf"]').forEach(a=>{if(!a.querySelector('.pilot-action-icon')&&/PDF|Certificat|Bon/.test(a.textContent))iconize(a,'pdf',a.textContent.trim());});
  qa('a[title="WhatsApp"],a[title="Rappel WhatsApp"]').forEach(a=>{if(!a.querySelector('.pilot-action-picture'))a.innerHTML='<img class="pilot-custom-icon pilot-action-picture" src="/icons/action-message-ui.png" alt="" aria-hidden="true" draggable="false"><span>WhatsApp</span>';});
  };
 const oldCompute=computeAlerts;computeAlerts=function(){
  const rows=oldCompute(),rank={urgent:0,warning:1,info:2};DATA.vehicles.forEach(v=>{
   const add=(severity,title,detail)=>{if(!rows.some(x=>x.title===title))rows.push({severity,type:'Entretien véhicule',title,detail,date:''});};
   const maintenance=(+v.maintenance_due||0)-(+v.mileage||0),oil=(+v.oil_change_due||0)-(+v.mileage||0),name=v.plate||[v.brand,v.model].filter(Boolean).join(' ');
   if(v.maintenance_due>0&&maintenance>0&&maintenance<=500)add('warning','Entretien bientôt requis · '+name,'Dans '+maintenance.toLocaleString('fr-FR')+' km');
   if(v.oil_change_due>0&&oil<=0)add('urgent','Vidange dépassée · '+name,Math.abs(oil).toLocaleString('fr-FR')+' km de dépassement');
   else if(v.oil_change_due>0&&oil<=500)add('warning','Vidange bientôt requise · '+name,'Dans '+oil.toLocaleString('fr-FR')+' km');
  });return rows.sort((a,b)=>rank[a.severity]-rank[b.severity]||(a.date||'9999').localeCompare(b.date||'9999'));
 };
 const oldPlanning=renderPlanning;renderPlanning=function(){oldPlanning();decorateAgenda();};
 const oldRender=render;render=function(){oldRender();enhance();};
 const oldReports=renderReports;renderReports=function(){oldReports();enhanceReports();};
 q('#entityForm').addEventListener('submit',e=>{const type=q('#entityType').value,status=q('#entityForm [name=status]')?.value;if((type==='lesson'||type==='invoice')&&status==='Annulée'&&!confirm('Confirmer l’annulation ? Cette action restera visible dans l’historique.')){e.preventDefault();e.stopImmediatePropagation();}},true);
 document.addEventListener('click',async e=>{
  const permanent=e.target.closest('[data-permanent-student]');
  if(permanent){e.preventDefault();e.stopImmediatePropagation();const student=DATA.students.find(x=>x.id===+permanent.dataset.permanentStudent);if(!student||!confirm('SUPPRESSION DÉFINITIVE de '+student.name+' ?\n\nLe candidat, ses séances, examens, documents, forfaits, factures et paiements seront supprimés sans possibilité de récupération.'))return;permanent.disabled=true;try{await api('/api/driving/student?id='+student.id+'&permanent=1',{method:'DELETE'});q('#profileModal').classList.remove('show');toast('Candidat et toutes ses données supprimés définitivement');await reload();}catch(err){toast(err.message,true);}finally{permanent.disabled=false;}return;}
  const remove=e.target.closest('[data-lesson-delete]');
  if(remove){e.preventDefault();e.stopImmediatePropagation();const lesson=DATA.lessons.find(x=>x.id===+remove.dataset.lessonDelete);if(!lesson||!confirm('Supprimer uniquement cette séance programmée ?\n\nLe candidat, ses autres séances et toutes ses données seront conservés.'))return;remove.disabled=true;try{await api('/api/driving/lesson?id='+lesson.id,{method:'DELETE'});toast('Séance supprimée — candidat conservé');await reload();}catch(err){toast(err.message,true);}finally{remove.disabled=false;}return;}
  const b=e.target.closest('[data-lesson-cancel]');if(!b)return;e.preventDefault();e.stopImmediatePropagation();const lesson=DATA.lessons.find(x=>x.id===+b.dataset.lessonCancel);if(!lesson||!confirm('Annuler cette séance ? Elle restera visible dans l’historique du candidat.'))return;
  b.disabled=true;try{await api('/api/driving/lesson',{method:'POST',body:JSON.stringify({...lesson,status:'Annulée'})});toast('Séance annulée et historique mis à jour');await reload();}catch(err){toast(err.message,true);}finally{b.disabled=false;}
 },true);
 enhance();
})();

/* Purposeful transparent illustrations supplied for this application. */
(() => {
 const q=s=>document.querySelector(s);
 const picture=(src,className,alt)=>{const img=document.createElement('img');img.src=src;img.className=className;img.alt=alt;img.draggable=false;img.decoding='async';return img;};
 const auth=q('#authScreen'),authVisual=q('#authScreen .authvisual');
 if(auth&&authVisual&&!authVisual.querySelector('.pilot-login-illustration')){
  auth.classList.add('pilot-auth-illustrated');
  authVisual.append(picture('/icons/scene-login.png','pilot-login-illustration','Assistant de gestion Auto-École Pilot Pro'));
 }
 const dashboardHero=q('#dashboard .hero');
 if(dashboardHero&&!dashboardHero.querySelector('.pilot-dashboard-illustration'))dashboardHero.append(picture('/icons/scene-dashboard.png','pilot-dashboard-illustration',''));
 const decorateViews=()=>{
  const planning=q('#planningSummary');
  if(planning&&!planning.querySelector('.pilot-planning-illustration'))planning.append(picture('/icons/scene-planning.png','pilot-planning-illustration',''));
  const studentEmpty=q('#studentsGrid .empty');
  if(studentEmpty&&!studentEmpty.querySelector('.pilot-empty-illustration'))studentEmpty.prepend(picture('/icons/scene-candidates.png','pilot-empty-illustration','Illustration candidats'));
  const vehicles=q('#vehiclesGrid');
  if(vehicles&&!vehicles.querySelector('.pilot-vehicle-illustration')){
   vehicles.classList.add('pilot-with-illustration');
   const visual=document.createElement('aside');visual.className='pilot-vehicle-illustration';visual.setAttribute('aria-hidden','true');visual.append(picture('/icons/scene-vehicle.png','',''));vehicles.append(visual);
  }
 };
 const currentRender=render;render=function(){currentRender();decorateViews();};
 const currentPlanning=renderPlanning;renderPlanning=function(){currentPlanning();decorateViews();};
 decorateViews();
})();
