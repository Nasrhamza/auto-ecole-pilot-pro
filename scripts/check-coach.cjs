// Isolated UI fixture. Never connects to an installed application or user data.
const fs=require('fs'),path=require('path'),assert=require('assert');
const {chromium}=require('C:/Users/user/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/playwright');
const root=path.resolve(__dirname,'..');
(async()=>{
 const browser=await chromium.launch({executablePath:'C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe',headless:true});
 try{
 const page=await browser.newPage({viewport:{width:1366,height:900}}),errors=[],writes=[];
 page.on('pageerror',e=>{errors.push(e.message);console.error('BROWSER:',e.stack);});
 const today=new Date().toISOString().slice(0,10);
 const data={settings:{theme:'#174c65',business:{name:'Moniteur Walid Nafougui'}},driving_config:{default_lesson_minutes:60,open_time:'08:00',close_time:'18:00',queue_order:[]},
 students:Array.from({length:8},(_,i)=>({id:i+1,name:'Élève '+(i+1)+' Nom composé',phone:'96753257',cin:'12345678',status:'Actif',permit:'B',package:1200,driving_target:20.5,theory_target:20,document_status:'Incomplet'})),
 instructors:[{id:1,name:'Walid',status:'Actif',permit:'B'}],vehicles:[{id:1,brand:'Renault',model:'Clio',plate:'123 TU 4567',status:'Disponible',category:'B',mileage:25000}],
 lessons:[{id:1,student_id:1,instructor_id:1,vehicle_id:1,start:today+'T09:00',duration:60,kind:'Conduite',status:'Terminée'},{id:2,student_id:2,instructor_id:1,vehicle_id:1,start:today+'T14:00',duration:90,kind:'Conduite',status:'Planifiée'}],
 exams:[{id:1,student_id:1,type:'Conduite',date:today,result:'Prévu',center:'Centre'}],
 driving_payments:[{id:1,student_id:1,time:today+'T09:00',amount:200,method:'Espèces'}],expenses:[{id:1,time:today,category:'Carburant',label:'Carburant du véhicule',amount:80}],
 driving_invoices:[{id:1,student_id:1,instructor_id:1,date:today,label:'Formation à la conduite du véhicule catégorie B',amount:1200,status:'À payer'}],
 driving_documents:[{id:1,student_id:1,category:'CIN',name:'CIN recto.png'},{id:2,student_id:1,category:'CIN',name:'CIN verso.png'}],driving_packages:[],package_sales:[],vehicle_events:[],instructor_absences:[],driving_audits:[]};
 await page.route('http://coach.test/**',async route=>{
  const req=route.request(),p=new URL(req.url()).pathname;
  if(p.startsWith('/api/')){
   if(p==='/api/branding/logo')return route.fulfill({contentType:'image/png',body:fs.readFileSync(path.join(root,'assets/logo.png'))});
   let result=p==='/api/data'?data:p==='/api/license/status'?{activated:true}:p==='/api/auth/status'?{authenticated:true,configured:true,username:'Walid',role:'Administrateur'}:p==='/api/public/branding'?{business:data.settings.business,theme:'#174c65'}:[];
   if(req.method()==='POST'){writes.push({url:p,body:req.postDataJSON()});result={id:100};}
   return route.fulfill({json:{ok:true,data:result}});
  }
  const file=path.join(root,'web',p==='/'?'index.html':p.slice(1));
  if(!fs.existsSync(file))return route.fulfill({status:404,body:'Not found'});
  return route.fulfill({body:fs.readFileSync(file),contentType:p.endsWith('.js')?'text/javascript':p.endsWith('.css')?'text/css':p.endsWith('.png')?'image/png':'text/html'});
 });
 await page.goto('http://coach.test/');await page.locator('#studentsGrid .studentcard').first().waitFor({state:'attached'});
 assert(await page.locator('.pilot-login-illustration').evaluate(img=>img.complete&&img.naturalWidth>0),'login illustration');
 assert.equal(await page.locator('#studentsGrid .studentavatar img').count(),data.students.length,'candidate avatars');
 assert(await page.locator('#studentsGrid .studentavatar img').first().evaluate(img=>img.complete&&img.naturalWidth>0),'candidate avatar loaded');
 assert.equal(await page.locator('#password').getAttribute('minlength'),null,'legacy password submission must not be blocked by HTML');
 for(const width of [1920,1366,1024]){
  await page.setViewportSize({width,height:900});
  for(const sec of ['dashboard','students','schedule','exams','finance','fleet','settings']){
   await page.locator('#nav [data-page='+sec+']').click();
   assert(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),'page overflow '+sec+' '+width);
   const clipped=await page.locator('#'+sec).evaluate(el=>[...el.querySelectorAll('.tablewrap,.agenda-item,.queuerow,.card')].filter(n=>n.getClientRects().length && n.scrollWidth>n.clientWidth+3).map(n=>({className:n.className,clientWidth:n.clientWidth,scrollWidth:n.scrollWidth,children:[...n.children].map(c=>({className:c.className,clientWidth:c.clientWidth,scrollWidth:c.scrollWidth}))})));
   if(clipped.length)console.error('OVERFLOW',sec,width,JSON.stringify(clipped));
   assert.deepEqual(clipped,[],'content overflow '+sec+' '+width);
  }
 }
 await page.setViewportSize({width:1366,height:900});
 await page.locator('#nav [data-page=schedule]').click();
 await page.locator('#scheduleDate').fill('2026-09-12');await page.locator('#scheduleDate').dispatchEvent('change');
 await page.locator('#weekGrid [data-day="2026-09-13"]').click();assert.equal(await page.locator('#scheduleDate').inputValue(),'2026-09-13','Sunday selection');assert((await page.locator('#agendaTitle').textContent()).includes('13'),'Sunday agenda');
 await page.locator('[data-planning-view=month]').click();assert.equal(await page.locator('#weekGrid [data-day]').count(),30,'September month');
 await page.locator('#weekNext').click();assert.equal(await page.locator('#weekGrid [data-day]').count(),31,'October month');await page.locator('#weekGrid [data-day="2026-10-31"]').click();assert.equal(await page.locator('#scheduleDate').inputValue(),'2026-10-31');
 await page.screenshot({path:path.join(root,'output/coach-month.png'),fullPage:true});
 await page.locator('[data-planning-view=week]').click();await page.locator('#scheduleDate').fill(today);await page.locator('#scheduleDate').dispatchEvent('change');
 const before=await page.locator('#scheduleDate').inputValue();await page.locator('#weekNext').click();
 assert.notEqual(await page.locator('#scheduleDate').inputValue(),before,'next week');await page.locator('#weekPrev').click();assert.equal(await page.locator('#scheduleDate').inputValue(),before);
 await page.screenshot({path:path.join(root,'output/coach-planning.png'),fullPage:true});
 await page.locator('#schedule [data-open=lesson]').first().click();
 await page.locator('#formFields [name=student_id]').selectOption('1');await page.locator('#formFields [name=start]').fill(today+'T11:00');await page.locator('#formFields [name=duration]').fill('60');await page.locator('#repeatMode').selectOption('weekly');await page.locator('#formFields [name=repeat_weeks]').fill('2');
 await page.locator('#entityForm').evaluate(f=>f.requestSubmit());await page.waitForFunction(()=>!document.querySelector('#modal').classList.contains('show'));
 assert.equal(writes.filter(w=>w.url.endsWith('/lesson')).length,2);assert(writes.find(w=>w.url.endsWith('/lesson')).body.start.endsWith('T11:00'),'local time preserved');
 for(const [mode,start,count] of [['week','2026-10-05T11:00',3],['month','2026-10-01T11:00',13]]){
  await page.evaluate(()=>openForm('lesson'));await page.locator('#formFields [name=student_id]').selectOption('1');await page.locator('#formFields [name=start]').fill(start);await page.locator('#repeatMode').selectOption(mode);
  for(const n of [0,1,2,3,4,5,6])await page.locator('#repeatDays input[value="'+n+'"]').setChecked([1,3,5].includes(n));
  assert((await page.locator('#repeatPreview').textContent()).startsWith(count+' séance'),'preview '+mode);
  const before=writes.filter(w=>w.url.endsWith('/lesson')).length;await page.locator('#entityForm').evaluate(f=>f.requestSubmit());await page.waitForFunction(()=>!document.querySelector('#modal').classList.contains('show'));
  const created=writes.filter(w=>w.url.endsWith('/lesson')).slice(before);assert.equal(created.length,count,mode+' count');assert(created.every(w=>w.body.start.endsWith('T11:00')),mode+' local hour');assert(created.every(w=>[1,3,5].includes(new Date(w.body.start).getDay())),mode+' weekdays');
 }
 await page.locator('#nav [data-page=students]').click();await page.locator('#studentSearch').fill('Élève 1');assert.equal(await page.locator('#studentsGrid .coachhours').count(),1);
 await page.evaluate(()=>openProfile(1));assert.equal(await page.locator('#profileDocs a').count(),2,'both CIN files');await page.locator('#profileClose').click();
 await page.locator('#nav [data-page=settings]').click();await page.getByRole('tab',{name:'Horaires & documents'}).click();assert(await page.locator('#configForm').isVisible());assert(!(await page.locator('#settingsForm').isVisible()));
 await page.mouse.move(1230,600);await page.mouse.wheel(0,500);await page.waitForTimeout(200);assert(await page.evaluate(()=>scrollY>0),'wheel scroll');
 await page.evaluate(()=>scrollTo(0,0));
 const cdp=await page.context().newCDPSession(page);await cdp.send('Emulation.setTouchEmulationEnabled',{enabled:true});
 await cdp.send('Input.dispatchTouchEvent',{type:'touchStart',touchPoints:[{x:1230,y:700}]});
 for(const y of [650,580,500,420]){await cdp.send('Input.dispatchTouchEvent',{type:'touchMove',touchPoints:[{x:1230,y}]});await page.waitForTimeout(35);}
 await cdp.send('Input.dispatchTouchEvent',{type:'touchEnd',touchPoints:[]});await page.waitForTimeout(200);assert(await page.evaluate(()=>scrollY>0),'native touch scroll');await page.evaluate(()=>scrollTo(0,0));
 await page.screenshot({path:path.join(root,'output/coach-settings.png'),fullPage:true});
 await page.locator('#nav [data-page=finance]').click();await page.screenshot({path:path.join(root,'output/coach-finance.png'),fullPage:true});
 assert.deepEqual(errors,[],'JavaScript errors');console.log('PASS: 21 page/viewport checks, navigation, forms, repeated local times, dossier, settings.');
 }finally{await browser.close();}
})().catch(e=>{console.error(e);process.exitCode=1;});
