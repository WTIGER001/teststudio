import { DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { FormControl, FormGroup, Validators } from '@angular/forms'
import { PreparationService } from './preparation.service'
import { bundleFileError } from './bundle.utils'

@Injectable()
export class BundleUploadService {
  readonly preparation = inject(PreparationService)
  readonly file = signal<File | null>(null)
  readonly error = signal('')
  readonly form = new FormGroup({ manifest: new FormControl('teststudio-suite.json', { nonNullable: true, validators: [Validators.required, Validators.maxLength(512)] }) })
  private key = crypto.randomUUID()

  constructor() {
    this.form.valueChanges.pipe(takeUntilDestroyed(inject(DestroyRef))).subscribe(() => { this.key = crypto.randomUUID() })
  }

  select(file: File | null): void { this.file.set(file); this.error.set(bundleFileError(file)); this.key = crypto.randomUUID() }

  async upload(): Promise<void> {
    const file = this.file(), error = bundleFileError(file)
    this.error.set(error)
    if (!file || error || this.form.invalid) return
    await this.preparation.uploadBundle(file, this.form.controls.manifest.value, this.key)
  }
}
