import { inject, Injectable, signal } from '@angular/core'
import { ApiService, type RecordValue } from '../../core/api.service'
import { randomId } from '../../core/random-id'
import type { ProbeBinding, ProbeSummary } from './probe.models'

interface BulkRow { probe: ProbeSummary; key: string; done: boolean; runId?: string; error?: string }
@Injectable()
export class ProbeBulkService {
 private readonly api=inject(ApiService)
 readonly selected=signal<ProbeSummary[]>([])
 readonly rows=signal<BulkRow[]>([])
 readonly kind=signal<'run'|'archive'>('run')
 readonly opened=signal(false)
 readonly busy=signal(false)
 readonly ready=signal(false)
 readonly error=signal('')
 toggle(probe:ProbeSummary,checked:boolean):void{this.selected.update(rows=>checked?[...rows.filter(r=>r.id!==probe.id),probe]:rows.filter(r=>r.id!==probe.id))}
 has(id:string):boolean{return this.selected().some(r=>r.id===id)}
 async review(project:string,kind:'run'|'archive'):Promise<void>{
  if(this.rows().some(r=>r.done)&&this.rows().some(r=>!r.done)){this.opened.set(true);this.error.set('Resume the unfinished reviewed actions below. Successful rows and their request identities are retained.');return}
  this.kind.set(kind);this.rows.set(this.selected().map(probe=>({probe,key:randomId(),done:false})));this.opened.set(true);this.ready.set(false);this.busy.set(true);this.error.set('')
  try{
   for(const row of this.rows()){
    const current=await this.api.request<RecordValue<ProbeBinding>>(`/api/v1/projects/${project}/probes/${row.probe.id}`)
    if(current.revision!==row.probe.revision||current.data.archived)throw new Error('A selected probe changed. Refresh the overview and select it again.')
    if(kind==='run'){
     if(!row.probe.activeReleaseId)throw new Error('Save each selected test and wait for setup before running its probes.')
     await this.api.request(`/api/v1/projects/${project}/runs/estimate`,'POST',this.request(row.probe))
    }
   }
   this.ready.set(true)
  }catch(e){this.error.set(e instanceof Error?e.message:'Selection review failed.')}
  finally{this.busy.set(false)}
 }
 private request(row:ProbeSummary):unknown{return {suiteId:row.suiteId,testIds:[row.testId],environmentId:row.environmentId,executionLocation:row.executionLocation,releaseId:row.activeReleaseId,timeoutSeconds:60,retries:0}}
 async confirm(project:string):Promise<void>{
  if(this.busy()||!this.ready())return
  this.busy.set(true);this.error.set('')
  for(let i=0;i<this.rows().length;i++){
   const row={...this.rows()[i]};if(row.done)continue
   try{
    if(this.kind()==='run'){const run=await this.api.request<{id:string}>(`/api/v1/projects/${project}/runs`,'POST',this.request(row.probe),undefined,row.key);row.runId=run.id}
    else{const p=row.probe;await this.api.request(`/api/v1/projects/${project}/probes/${p.id}`,'PUT',{id:p.id,name:p.name,suiteId:p.suiteId,testId:p.testId,environmentId:p.environmentId,endpoint:p.endpoint,executionLocation:p.executionLocation,archived:true},p.revision)}
    row.done=true;row.error=''
   }catch(e){
    row.error=e instanceof Error?e.message:'This action failed.'
    if(this.kind()==='archive'){
     try{const current=await this.api.request<RecordValue<ProbeBinding>>(`/api/v1/projects/${project}/probes/${row.probe.id}`);if(current.data.archived){row.done=true;row.error=''}}catch{}
    }
   }
   this.rows.update(rows=>rows.map((v,j)=>j===i?row:v))
  }
  this.busy.set(false)
  if(this.rows().some(r=>!r.done))this.error.set('Some actions remain unfinished. Retry here with the same request identities; completed rows are skipped. After reload inspect stored runs and configuration before repeating uncertain actions.')
  else this.selected.set([])
 }
 close():void{if(!this.busy())this.opened.set(false)}
}
