import { DestroyRef, inject, Injectable, signal } from '@angular/core'
import { FormControl, FormGroup, Validators } from '@angular/forms'
import { ActivatedRoute, Router } from '@angular/router'
import { ApiService, type RecordValue, type Page } from '../../core/api.service'
import { WorkspaceService } from '../../core/workspace.service'
import { randomId } from '../../core/random-id'
import { WizardController } from '../../shared/wizard/wizard-controller'
import { suggestedIdentifier } from '../../shared/wizard/creation.utils'
import { parseProbeAddress } from '../catalog/probe-address.utils'
import type { Application, Environment, Suite } from '../configuration/configuration.service'
import type { TestDefinition } from '../catalog/catalog.models'

export interface SetupRow { key: string; url: string; title: string; stage: string; application: string; protected: boolean; environmentId?: string; testId?: string; probeId?: string; error?: string; targetSaved?: boolean; testSaved?: boolean; endpoint?: string; path?: string }
interface ExistingTest extends RecordValue<TestDefinition> { suiteId: string; suiteName: string }
@Injectable()
export class ProbeSetupService {
 readonly workspace = inject(WorkspaceService)
 private readonly api = inject(ApiService)
 private readonly route = inject(ActivatedRoute)
 private readonly router = inject(Router)
 readonly project = this.route.snapshot.paramMap.get('projectId') ?? ''
 readonly adopt = this.route.snapshot.data['adopt'] === true
 readonly suites = signal<RecordValue<Suite>[]>([])
 readonly apps = signal<RecordValue<Application>[]>([])
 readonly environments = signal<RecordValue<Environment>[]>([])
 readonly tests = signal<ExistingTest[]>([])
 readonly rows = signal<SetupRow[]>([])
 readonly loading = signal(false)
 readonly busy = signal(false)
 readonly locked = signal(false)
 readonly saved = signal(false)
 readonly error = signal('')
 readonly progress = signal('')
 readonly form = new FormGroup({ addresses: new FormControl('',{nonNullable:true}), suite: new FormControl('',{nonNullable:true,validators:Validators.required}), test: new FormControl('',{nonNullable:true}), environment: new FormControl('',{nonNullable:true}), safety: new FormControl('needs-review',{nonNullable:true}), rationale: new FormControl('Production safety has not been reviewed.',{nonNullable:true,validators:Validators.required}) })
 readonly flow = new WizardController(3, step => this.validate(step))
 eligibleTests(): ExistingTest[] { return this.tests().filter(t => t.suiteId === this.form.controls.suite.value) }
 private disposed = false
 constructor() { inject(DestroyRef).onDestroy(() => { this.disposed = true }); void this.load() }
 base(): string { return `/api/v1/projects/${encodeURIComponent(this.project)}` }
 async load(): Promise<void> {
  this.loading.set(true);this.error.set('')
  try {
   const [suites,apps,envs] = await Promise.all([this.api.request<Page<RecordValue<Suite>>>(`${this.base()}/suites`),this.api.request<Page<RecordValue<Application>>>(`${this.base()}/applications`),this.api.request<Page<RecordValue<Environment>>>(`${this.base()}/environments`)])
   if (this.disposed) return
   this.suites.set(suites.items.filter(s=>!s.data.archived));this.apps.set(apps.items);this.environments.set(envs.items)
   if (suites.items.length===1) this.form.controls.suite.setValue(suites.items[0].id)
   if (this.adopt) {
    let cursor='';const tests:ExistingTest[]=[]
    do { const page=await this.api.request<Page<ExistingTest>>(`${this.base()}/tests?type=https&limit=100${cursor?'&cursor='+encodeURIComponent(cursor):''}`);tests.push(...page.items);cursor=page.cursor??'';if(tests.length>1000)throw new Error('Narrow the project catalog before adopting more than 1000 definitions.') } while(cursor)
    this.tests.set(tests)
   }
  } catch(e){this.error.set(e instanceof Error?e.message:'Setup options are unavailable.')}
  finally{this.loading.set(false)}
 }
 applications(): RecordValue<Application>[] {
  const suite=this.suites().find(s=>s.id===this.form.controls.suite.value)?.data
  return this.apps().filter(a=>!suite?.applications.length||suite.applications.includes(a.id))
 }
 matches(row:SetupRow): { environment: RecordValue<Environment>; endpoint:string }[] {
  const parsed=parseProbeAddress(row.url); if(!parsed)return []
  return this.environments().filter(e=>e.data.applicationId===row.application).flatMap(environment=>Object.entries(environment.data.endpoints).filter(([,url])=>{const u=new URL(url);return u.origin===parsed.origin && u.pathname.replace(/\/$/,'')===''}).map(([endpoint])=>({environment,endpoint})))
 }
 update(index:number,patch:Partial<SetupRow>):void {if(!this.locked())this.rows.update(rows=>rows.map((row,i)=>i===index?{...row,...patch}:row))}
 validate(step:number):boolean {
  this.error.set('')
  if (this.locked()) return true
  if (!this.form.controls.suite.value){this.error.set('Choose a destination suite.');return false}
  if(this.adopt){
   const t=this.tests().find(t=>t.id===this.form.controls.test.value&&t.suiteId===this.form.controls.suite.value)
   const env=this.environments().find(e=>e.id===this.form.controls.environment.value)
   if(!t?.data.check||!env||!env.data.endpoints[t.data.check.endpoint]){this.error.set('Choose a native check and its matching target environment.');return false}
   const row:SetupRow={key:t.id,url:env.data.endpoints[t.data.check.endpoint]+t.data.check.path,title:t.data.title,stage:env.data.stage,application:env.data.applicationId,protected:env.data.protected,environmentId:env.id,testId:t.id,endpoint:t.data.check.endpoint}
   this.rows.set([row]);return true
  }
  if(step===0){
   const addresses=this.form.controls.addresses.value.split(/\r?\n/).map(v=>v.trim()).filter(Boolean)
   if(!addresses.length||addresses.length>30||addresses.some(v=>!parseProbeAddress(v))||new Set(addresses.map(v=>new URL(v).href)).size!==addresses.length){this.error.set('Enter 1 through 30 unique complete HTTP or HTTPS URLs, one per line, without credentials or fragments.');return false}
   const old=this.rows()
   this.rows.set(addresses.map(url=>old.find(r=>r.url===url)??{key:randomId().slice(0,12),url,title:parseProbeAddress(url)!.hostname,stage:'development',application:this.applications().length===1?this.applications()[0].id:'',protected:false}))
  }
  if(step===1){
   if(!this.form.controls.rationale.value.trim()){this.error.set('Provide a safety rationale for the new checks.');return false}
   for(const row of this.rows()){
    const matches=this.matches(row)
    const selected=matches.length===1?matches[0]:matches.find(m=>m.environment.id===row.environmentId)
    if(matches.length&&!selected){this.error.set('Choose an explicit matching target for every ambiguous address.');return false}
    if(selected){row.environmentId=selected.environment.id;row.endpoint=selected.endpoint;row.stage=selected.environment.data.stage;row.protected=selected.environment.data.protected}
    const address=parseProbeAddress(row.url)!
    if(!row.title.trim()||!this.applications().some(a=>a.id===row.application)){this.error.set('Give every probe a title and choose an application linked to the suite.');return false}
    if(address.protocol==='http:'&&row.stage!=='local'){this.error.set('Plain HTTP requires an explicitly chosen Local target stage.');return false}
   }
  }
  return true
 }
 async save():Promise<void>{
  if(this.busy()||!this.flow.ready()||this.saved())return
  this.locked.set(true);this.busy.set(true);this.error.set('')
  let failed=0
  for(let i=0;i<this.rows().length;i++){
   if(this.disposed)break
   const row={...this.rows()[i]};if(row.probeId)continue
   try {
    const parsed=parseProbeAddress(row.url)!
    if(!this.adopt){
     const matches=this.environments().flatMap(e=>Object.entries(e.data.endpoints).filter(([,url])=>new URL(url).origin===parsed.origin&&new URL(url).pathname.replace(/\/$/,'')==='').map(([endpoint])=>({environment:e,endpoint}))).filter(m=>m.environment.data.applicationId===row.application)
     if(!row.environmentId&&matches.length>1)throw new Error('Several existing targets match this origin. Use Add existing check or the single-probe wizard to choose the target explicitly.')
     if(!row.environmentId&&matches.length===1){row.environmentId=matches[0].environment.id;row.endpoint=matches[0].endpoint;row.stage=matches[0].environment.data.stage;row.protected=matches[0].environment.data.protected}
     if(!row.environmentId)row.environmentId=suggestedIdentifier(parsed.hostname+'-'+row.stage,row.key)
     if(!row.endpoint)row.endpoint='api'
     if(!row.targetSaved&&!matches.some(m=>m.environment.id===row.environmentId)){
      if(!this.workspace.can('environments.manage'))throw new Error('Environment management permission is required for this new address.')
      const target:Environment={id:row.environmentId,name:row.title+' target',applicationId:row.application,stage:row.stage,protected:row.protected,disabled:false,endpoints:{[row.endpoint]:parsed.origin}}
      try{await this.api.request(`${this.base()}/environments/${row.environmentId}`,'PUT',target,0)}catch(e){const existing=await this.api.request<RecordValue<Environment>>(`${this.base()}/environments/${row.environmentId}`);if(existing.data.applicationId!==target.applicationId||existing.data.stage!==target.stage||existing.data.protected!==target.protected||existing.data.endpoints[row.endpoint]!==parsed.origin)throw e}
      this.environments.update(rows=>[...rows.filter(e=>e.id!==target.id),{id:target.id,revision:1,updatedAt:'',data:target}]);row.targetSaved=true
     }
     row.testId??=suggestedIdentifier(row.title,row.key)
     this.record(i,row)
     if(!row.testSaved){
      const test={id:row.testId,title:row.title,applicationId:row.application,type:'https',runner:'http',state:'enabled',safety:this.form.controls.safety.value,rationale:this.form.controls.rationale.value,category:'Availability',description:'',tags:[],dependencies:[],requiredBindings:[],timeoutSeconds:10,check:{endpoint:row.endpoint,path:parsed.path,method:'GET',statusCodes:[200],minCertificateDays:parsed.protocol==='https:'?14:0}}
      try{await this.api.request(`${this.base()}/suites/${this.form.controls.suite.value}/tests`,'POST',test)}catch(e){const existing=await this.api.request<RecordValue<TestDefinition>>(`${this.base()}/suites/${this.form.controls.suite.value}/tests/${row.testId}`);if(!this.sameValues(existing.data,test))throw e}
      row.testSaved=true;this.record(i,row)
     }
    }
    const binding=await this.api.request<{id:string}>(`${this.base()}/probes`,'POST',{name:row.title,suiteId:this.form.controls.suite.value,testId:row.testId,environmentId:row.environmentId,endpoint:row.endpoint??'api'})
    row.probeId=binding.id;row.error=''
   }catch(e){row.error=e instanceof Error?e.message:'This row could not be saved.';failed++}
   this.record(i,row);this.progress.set(`${i+1} of ${this.rows().length} rows processed.`)
  }
  this.busy.set(false)
  if(failed)this.error.set('Some rows remain unfinished. Saved resources are retained. Retry unfinished rows on this page; successful rows are skipped. After reload inspect stored resources before adding them again.')
  else{this.saved.set(true);this.form.markAsPristine()}
 }
 private sameValues(actual:unknown,expected:unknown):boolean {
  if(Array.isArray(expected))return Array.isArray(actual)&&actual.length===expected.length&&expected.every((v,i)=>this.sameValues(actual[i],v))
  if(expected!==null&&typeof expected==='object')return actual!==null&&typeof actual==='object'&&Object.entries(expected).every(([k,v])=>this.sameValues((actual as Record<string,unknown>)[k],v))
  return actual===expected
 }
 private record(index:number,row:SetupRow):void{this.rows.update(rows=>rows.map((r,i)=>i===index?{...row}:r))}
 canLeave():boolean{return this.saved()||!this.locked()&&!this.form.dirty||window.confirm('Leave probe setup? Saved resources remain. Unsaved entries and in-page retry receipts will be lost.')}
 async done():Promise<void>{await this.router.navigate(['/projects',this.project,'probes'])}
}
