import { DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { ActivatedRoute } from '@angular/router'
import { ConfigurationService } from './configuration.service'
import { SuiteManagerService } from './suite-manager.service'

@Injectable()
export class SuitesPageService {
  readonly configuration = inject(ConfigurationService)
  readonly manager = inject(SuiteManagerService)
  readonly creating = signal(false)

  constructor() { inject(ActivatedRoute).paramMap.pipe(takeUntilDestroyed(inject(DestroyRef))).subscribe(() => { this.creating.set(false); this.configuration.select('suites') }) }
  openCreation(): void { if (!this.canLeave()) return; this.manager.close(); this.configuration.select('suites'); this.creating.set(true) }
  closeCreation(): void { if (this.canLeave()) { this.configuration.form.markAsPristine(); this.creating.set(false) } }
  async create(): Promise<void> { if (await this.configuration.create()) this.creating.set(false) }
  async inspect(id: string): Promise<void> { if (!this.canLeave()) return; this.configuration.form.markAsPristine(); this.creating.set(false); await this.manager.open(id) }
  canLeave(): boolean { return !(this.creating() && this.configuration.form.dirty || this.manager.opened() && this.manager.form.dirty) || window.confirm('These suite changes are not saved. Leave without saving?') }
}
