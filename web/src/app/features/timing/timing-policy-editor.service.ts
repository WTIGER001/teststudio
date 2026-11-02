import { Injectable } from '@angular/core'
import { toSignal } from '@angular/core/rxjs-interop'
import { FormControl, FormGroup, Validators } from '@angular/forms'
import { startWith } from 'rxjs'
import type { TimingPolicy } from './timing.models'
import { defaultTimingPolicy, timingPolicyIssue } from './timing.utils'

@Injectable()
export class TimingPolicyEditorService {
  readonly override = new FormControl(false, { nonNullable: true })
  readonly form = new FormGroup({
    mode: new FormControl<TimingPolicy['mode']>('fail', { nonNullable: true }),
    slowdownFactor: new FormControl(2, { nonNullable: true, validators: [Validators.required, Validators.min(1), Validators.max(100)] }),
    minimumIncreaseMs: new FormControl(100, { nonNullable: true, validators: [Validators.required, Validators.min(0), Validators.max(3600000)] }),
    minimumSamples: new FormControl(5, { nonNullable: true, validators: [Validators.required, Validators.min(3), Validators.max(1000), Validators.pattern(/^\d+$/)] }),
    window: new FormControl(50, { nonNullable: true, validators: [Validators.required, Validators.min(3), Validators.max(1000), Validators.pattern(/^\d+$/)] }),
    maxAgeDays: new FormControl(30, { nonNullable: true, validators: [Validators.required, Validators.min(1), Validators.max(3650), Validators.pattern(/^\d+$/)] }),
    timeoutMode: new FormControl<TimingPolicy['timeoutMode']>('adaptive', { nonNullable: true }),
    timeoutFactor: new FormControl(3, { nonNullable: true, validators: [Validators.required, Validators.min(1), Validators.max(100)] }),
    timeoutMarginMs: new FormControl(250, { nonNullable: true, validators: [Validators.required, Validators.min(0), Validators.max(3600000)] }),
    minimumTimeoutMs: new FormControl(1000, { nonNullable: true, validators: [Validators.required, Validators.min(1), Validators.max(3600000), Validators.pattern(/^\d+$/)] })
  })
  readonly value = toSignal(this.form.valueChanges.pipe(startWith(this.form.getRawValue())))
  readonly enabled = toSignal(this.override.valueChanges.pipe(startWith(false)))

  reset(policy?: TimingPolicy): void { this.override.setValue(!!policy); this.form.reset(policy ?? defaultTimingPolicy()) }
  policy(): TimingPolicy | undefined { return this.override.value ? this.form.getRawValue() : undefined }
  issue(): string { return this.override.value ? timingPolicyIssue(this.form.getRawValue()) : '' }
  valid(): boolean { return !this.override.value || (this.form.valid && !this.issue()) }
}
