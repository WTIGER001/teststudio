import { DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { FormControl, FormGroup, Validators } from '@angular/forms'
import { ActivatedRoute } from '@angular/router'
import { AccessClientService } from '../../core/access-client.service'
import { ApiError } from '../../core/api.service'
import type { SuiteLifecycle } from '../../core/generated/access.types'
import { ConfigurationService } from './configuration.service'

@Injectable()
export class SuiteManagerService {
  readonly configuration = inject(ConfigurationService)
  private readonly client = inject(AccessClientService)
  readonly opened = signal(false)
  readonly state = signal<SuiteLifecycle | null>(null)
  readonly busy = signal(false)
  readonly conflict = signal(false)
  readonly error = signal('')
  readonly message = signal('')
  readonly form = new FormGroup({
    name: new FormControl('', { nonNullable: true, validators: [Validators.required, Validators.maxLength(200), Validators.pattern(/\S/)] }),
    description: new FormControl('', { nonNullable: true, validators: Validators.maxLength(4096) }),
    applications: new FormControl<string[]>([], { nonNullable: true }),
    requiresTarget: new FormControl(true, { nonNullable: true })
  })
  private suite = ''
  private generation = 0

  constructor() {
    const destroy = inject(DestroyRef)
    inject(ActivatedRoute).paramMap.pipe(takeUntilDestroyed(destroy)).subscribe(() => this.close())
    destroy.onDestroy(() => { this.generation++ })
  }

  async open(suite: string): Promise<void> { this.suite = suite; this.opened.set(true); this.state.set(null); this.error.set(''); this.message.set(''); await this.reload() }
  close(): void { this.generation++; this.suite = ''; this.opened.set(false); this.state.set(null); this.busy.set(false); this.conflict.set(false); this.error.set(''); this.message.set(''); this.form.reset() }

  async reload(): Promise<void> {
    if (!this.suite) return
    const generation = ++this.generation
    this.busy.set(true); this.error.set('')
    try { const state = await this.client.call('readSuiteLifecycle', this.scope()); if (generation === this.generation) this.apply(state) }
    catch (error) { if (generation === this.generation) this.fail(error) }
    finally { if (generation === this.generation) this.busy.set(false) }
  }

  async save(): Promise<void> {
    const state = this.state(), generation = this.generation
    if (!state?.canManage || state.suite.archived || this.form.invalid || this.busy() || this.conflict()) return
    this.busy.set(true); this.error.set(''); this.message.set('')
    try {
      const updated = await this.client.call('updateSuiteSettings', { ...this.scope(), revision: state.revision, body: this.form.getRawValue() })
      if (generation !== this.generation) return
      this.apply(updated); this.message.set('Suite settings saved. Releases, history, and evidence remain unchanged.')
    } catch (error) { if (generation === this.generation) this.fail(error) }
    finally { if (generation === this.generation) this.busy.set(false) }
  }

  async changeState(action: 'archive' | 'restore'): Promise<void> {
    const state = this.state(), generation = this.generation
    if (!state?.canManage || this.busy() || this.conflict() || this.form.dirty) return
    this.busy.set(true); this.error.set(''); this.message.set('')
    try {
      const updated = await this.client.call('changeSuiteLifecycle', { ...this.scope(), action, revision: state.revision })
      if (generation !== this.generation) return
      this.apply(updated); this.message.set(action === 'archive' ? 'Suite archived. New dispatch is blocked. Retained releases, results, evidence, and schedule pins remain available.' : 'Suite restored with the same identifier, active release, and schedule pins.')
    } catch (error) { if (generation === this.generation) this.fail(error) }
    finally { if (generation === this.generation) this.busy.set(false) }
  }

  private scope() { return { project: this.configuration.projectId(), suite: this.suite } }
  private apply(state: SuiteLifecycle): void {
    this.state.set(state); this.conflict.set(false); this.form.reset({ name: state.suite.name, description: state.suite.description, applications: state.suite.applications ?? [], requiresTarget: state.suite.requiresTarget }); this.configuration.applySuite(state)
  }
  private fail(error: unknown): void {
    if (error instanceof ApiError && error.status === 409) { this.conflict.set(true); this.error.set('The suite changed. Reload suite settings and review the current values before saving or changing its lifecycle.'); return }
    this.error.set(error instanceof Error ? error.message : 'Suite settings could not be updated.')
    if (error instanceof ApiError && (error.status === 401 || error.status === 403)) { this.state.set(null); this.form.reset() }
  }
}
