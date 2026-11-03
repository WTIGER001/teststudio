import { computed, DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed, toSignal } from '@angular/core/rxjs-interop'
import { ActivatedRoute, Router } from '@angular/router'
import { map } from 'rxjs'
import { ConfigurationService, type Kind } from '../configuration/configuration.service'
import { applicationResources, findApplication, summarizeApplications } from './applications.utils'

@Injectable()
export class ApplicationsService {
  readonly registration = inject(ConfigurationService)
  private readonly route = inject(ActivatedRoute)
  private readonly router = inject(Router)
  private readonly destroy = inject(DestroyRef)
  private disposed = false
  readonly applicationId = toSignal(this.route.paramMap.pipe(map(params => params.get('applicationId') ?? '')), { initialValue: this.route.snapshot.paramMap.get('applicationId') ?? '' })
  readonly current = computed(() => findApplication(this.registration.applications(), this.applicationId()))
  readonly applications = computed(() => summarizeApplications(this.registration.applications(), this.registration.environments(), this.registration.suites()))
  readonly related = computed(() => applicationResources(this.applicationId(), this.registration.environments(), this.registration.suites()))
  readonly formOpen = signal(false)

  constructor() {
    this.route.paramMap.pipe(takeUntilDestroyed(this.destroy)).subscribe(() => { this.formOpen.set(false); this.registration.form.markAsPristine() })
    this.destroy.onDestroy(() => { this.disposed = true })
  }

  openForm(kind: Kind): void {
    if (this.formOpen() && this.registration.form.dirty && !window.confirm('Discard these unfinished registration changes?')) return
    this.registration.select(kind)
    this.registration.form.reset({ id: '', name: '', application: kind === 'applications' ? '' : this.applicationId(), stage: 'development', protected: false, endpoint: 'api', url: '', requiresTarget: true })
    if (kind === 'applications') this.registration.form.controls.application.enable()
    else this.registration.form.controls.application.disable()
    this.formOpen.set(true)
  }

  closeForm(): void { if (!this.registration.form.dirty || window.confirm('Discard these unfinished registration changes?')) { this.registration.form.markAsPristine(); this.formOpen.set(false) } }

  async create(): Promise<void> {
    const id = this.registration.form.controls.id.value, kind = this.registration.kind(), project = this.registration.projectId(), application = this.applicationId()
    if (!await this.registration.create() || this.disposed || project !== this.registration.projectId() || application !== this.applicationId()) return
    this.formOpen.set(false)
    if (kind === 'applications') void this.router.navigate(['/projects', this.registration.projectId(), 'applications', id])
  }

  canLeave(): boolean { return !this.formOpen() || !this.registration.form.dirty || window.confirm('This registration has unsaved changes. Leave without saving?') }
}
