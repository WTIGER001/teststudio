import { FormArray, FormControl, FormGroup } from '@angular/forms'
import type { RoleRule } from '../../core/generated/access.types'
import { roleRuleFields } from './role-policy.utils'

export function roleRuleForm(value?: RoleRule) { const fields = roleRuleFields(value); return new FormGroup({ permission: new FormControl(fields.permission, { nonNullable: true }), effect: new FormControl(fields.effect, { nonNullable: true }), object: new FormControl(fields.object, { nonNullable: true }), expiresAt: new FormControl(fields.expiresAt, { nonNullable: true }) }) }
export type RoleRuleForm = ReturnType<typeof roleRuleForm>
export function rolePolicyForm() { return new FormGroup({ id: new FormControl('', { nonNullable: true }), name: new FormControl('', { nonNullable: true }), description: new FormControl('', { nonNullable: true }), reason: new FormControl('', { nonNullable: true }), rules: new FormArray<RoleRuleForm>([]), inherits: new FormArray<FormControl<string>>([]) }) }
