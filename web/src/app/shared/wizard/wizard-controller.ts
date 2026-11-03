import { signal } from '@angular/core'

export class WizardController {
  readonly step = signal(0)
  readonly reached = signal(0)
  constructor(readonly length: number, private readonly validate: (step: number) => boolean) {}

  go(target: number): boolean {
    if (target < 0 || target >= this.length) return false
    if (target > this.step()) for (let index = 0; index < target; index++) if (!this.validate(index)) { this.step.set(index); return false }
    this.step.set(target); this.reached.update(value => Math.max(value, target))
    return true
  }
  next(): boolean { return this.go(this.step() + 1) }
  back(): void { this.go(this.step() - 1) }
  reset(): void { this.step.set(0); this.reached.set(0) }
  ready(): boolean { for (let index = 0; index < this.length - 1; index++) if (!this.validate(index)) { this.step.set(index); return false }; return true }
}
