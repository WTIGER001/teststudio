import { computed, Injectable } from '@angular/core'
import { toSignal } from '@angular/core/rxjs-interop'
import { FormControl, FormGroup, Validators } from '@angular/forms'
import { map, startWith } from 'rxjs'
import type { HTTPCheck } from './catalog.models'
import { httpCheckForm, parseHTTPCheck } from './http-check.utils'

@Injectable()
export class HTTPCheckEditorService {
  readonly form = new FormGroup({
    endpoint: new FormControl('api', { nonNullable: true, validators: Validators.required }),
    path: new FormControl('/healthz', { nonNullable: true, validators: Validators.required }),
    method: new FormControl('GET', { nonNullable: true, validators: Validators.required }),
    statusCodes: new FormControl('200', { nonNullable: true, validators: Validators.required }),
    headers: new FormControl('', { nonNullable: true }), json: new FormControl('', { nonNullable: true }),
    requiredText: new FormControl('', { nonNullable: true }), forbiddenText: new FormControl('', { nonNullable: true }),
    maxResponseMs: new FormControl(0, { nonNullable: true, validators: Validators.required }),
    minCertificateDays: new FormControl(0, { nonNullable: true, validators: Validators.required })
  })
  private readonly value = toSignal(this.form.valueChanges.pipe(map(() => this.form.getRawValue()), startWith(this.form.getRawValue())), { requireSync: true })
  private readonly status = toSignal(this.form.statusChanges.pipe(startWith(this.form.status)), { requireSync: true })
  private readonly parsed = computed(() => parseHTTPCheck(this.value()))
  readonly errors = computed(() => this.parsed().errors)
  readonly valid = computed(() => this.status() === 'VALID' && !!this.parsed().check)

  reset(check?: HTTPCheck): void { this.form.reset(httpCheckForm(check)) }
  check(): HTTPCheck | undefined { return this.parsed().check }
}
