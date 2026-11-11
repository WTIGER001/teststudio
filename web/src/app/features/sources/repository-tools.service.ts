import { computed, effect, inject, Injectable, signal } from '@angular/core';
import { FormControl } from '@angular/forms';
import { AccessClientService } from '../../core/access-client.service';
import type { SourceRepositoryAccessCheck, SourceRepositoryInspection } from '../../core/generated/access.types';
import { randomId } from '../../core/random-id';
import { RepositoriesService } from './repositories.service';
@Injectable()
export class RepositoryToolsService {
    readonly repositories = inject(RepositoriesService);
    private readonly client = inject(AccessClientService);
    readonly access = signal<SourceRepositoryAccessCheck | null>(null);
    readonly expiryMessage = computed(() => { const expiry = this.access()?.credential.expiresAt; if (!expiry)
        return ''; const days = (Date.parse(expiry) - Date.now()) / 86400000; return days <= 0 ? 'This credential has expired. Replace it before the next source refresh.' : days <= 14 ? 'This credential expires within 14 days. Rotate or replace it before expiry.' : ''; });
    readonly inspection = signal<SourceRepositoryInspection | null>(null);
    readonly replacement = new FormControl('', { nonNullable: true });
    readonly busy = signal(false);
    readonly error = signal('');
    readonly message = signal('');
    private generation = 0;
    private rotationKey = randomId();
    constructor() { effect(() => { const id = this.repositories.selected()?.id; void id; this.generation++; this.access.set(null); this.inspection.set(null); this.error.set(''); this.message.set(''); this.replacement.setValue('', { emitEvent: false }); this.rotationKey = randomId(); if (id && this.repositories.sources.workspace.can('sources.manage'))
        void this.loadStatus(); }); }
    private async loadStatus(): Promise<void> { const generation = this.generation; try {
        const status = await this.client.call('readRepositoryAccessStatus', this.scope());
        if (generation === this.generation)
            this.access.set(status);
    }
    catch (error) {
        if (generation === this.generation)
            this.error.set(error instanceof Error ? error.message : 'Access status could not be loaded.');
    } }
    private scope() { return { project: this.repositories.sources.project(), source: this.repositories.selected()!.id }; }
    private async perform(work: () => Promise<void>): Promise<void> { if (this.busy())
        return; this.busy.set(true); this.error.set(''); try {
        await work();
    }
    catch (error) {
        this.error.set(error instanceof Error ? error.message : 'Repository request failed.');
    }
    finally {
        this.busy.set(false);
    } }
    async validate(): Promise<void> { const generation = this.generation; await this.perform(async () => { const result = await this.client.call('checkRepositoryAccess', { ...this.scope(), body: { credentialReference: this.replacement.value || undefined } }); if (generation === this.generation) {
        this.access.set(result);
        this.message.set('Repository identity and access validated. No tests started.');
    } }); }
    async inspect(): Promise<void> { const generation = this.generation; await this.perform(async () => { const result = await this.client.call('inspectRepositorySource', this.scope()); if (generation === this.generation) {
        this.inspection.set(result);
        this.message.set('Source inspected at the shown commit. No test code executed.');
    } }); }
    async replace(): Promise<void> { const row = this.repositories.selected(); if (!row || !this.replacement.value)
        return; const generation = this.generation; await this.perform(async () => { await this.client.call('checkRepositoryAccess', { ...this.scope(), body: { credentialReference: this.replacement.value } }); const saved = await this.client.call('saveRepositorySource', { ...this.scope(), revision: row.revision, body: { ...row.data.configuration, credentialReference: this.replacement.value } }); if (generation === this.generation) {
        await this.repositories.inspect(saved);
        this.message.set('Validated replacement saved. Revoke the previous provider credential after all consumers have switched.');
    } }); }
    async rotate(): Promise<void> { if (!window.confirm('Rotate the GitLab project token now? The previous token stops working immediately. Recovery may require operator replacement.'))
        return; const generation = this.generation; await this.perform(async () => { const result = await this.client.call('rotateRepositoryCredential', { ...this.scope(), key: this.rotationKey }); if (generation === this.generation) {
        this.message.set(result.message);
        if (result.state === 'rotated')
            this.rotationKey = randomId();
        this.access.set(null);
    } }); }
}
